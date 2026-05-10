package multivendor

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentStatus string
type PayoutStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"

	PayoutStatusPending   PayoutStatus = "pending"
	PayoutStatusScheduled PayoutStatus = "scheduled"
	PayoutStatusPaid      PayoutStatus = "paid"
	PayoutStatusFailed    PayoutStatus = "failed"
)

type Payment struct {
	ID            uuid.UUID       `json:"id"              db:"id"`
	OrderID       uuid.UUID       `json:"order_id"        db:"order_id"`
	BuyerID       uuid.UUID       `json:"buyer_id"        db:"buyer_id"`
	Amount        decimal.Decimal `json:"amount"          db:"amount"`
	Currency      string          `json:"currency"        db:"currency"`
	Method        string          `json:"method"          db:"method"`
	Status        PaymentStatus   `json:"status"          db:"status"`
	GatewayTxnID  string          `json:"gateway_txn_id" db:"gateway_txn_id"`
	GatewayRef    string          `json:"gateway_ref"    db:"gateway_ref"`
	FailureReason string          `json:"failure_reason,omitempty" db:"failure_reason"`
	CreatedAt     time.Time       `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"     db:"updated_at"`
}

// VendorPayout tracks platform-to-vendor settlement after commission deduction.
type VendorPayout struct {
	ID               uuid.UUID       `json:"id"               db:"id"`
	VendorID         uuid.UUID       `json:"vendor_id"        db:"vendor_id"`
	SubOrderID       uuid.UUID       `json:"sub_order_id"     db:"sub_order_id"`
	GrossAmount      decimal.Decimal `json:"gross_amount"     db:"gross_amount"`
	CommissionAmount decimal.Decimal `json:"commission_amount" db:"commission_amount"`
	NetAmount        decimal.Decimal `json:"net_amount"       db:"net_amount"`
	Currency         string          `json:"currency"         db:"currency"`
	Status           PayoutStatus    `json:"status"           db:"status"`
	ScheduledFor     *time.Time      `json:"scheduled_for,omitempty" db:"scheduled_for"`
	PaidAt           *time.Time      `json:"paid_at,omitempty" db:"paid_at"`
	TransferID       string          `json:"transfer_id,omitempty" db:"transfer_id"`
	CreatedAt        time.Time       `json:"created_at"       db:"created_at"`
}

type PaymentGateway interface {
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
	Refund(ctx context.Context, txnID string, amount decimal.Decimal) error
	Transfer(ctx context.Context, req TransferRequest) (*TransferResult, error)
}

type ChargeRequest struct {
	Amount      decimal.Decimal
	Currency    string
	Method      string
	Token       string
	Description string
}

type ChargeResult struct {
	TransactionID string
	Reference     string
	Status        string
}

type TransferRequest struct {
	Amount        decimal.Decimal
	Currency      string
	DestinationID string
	Description   string
}

type TransferResult struct {
	TransferID string
	Status     string
}

type PaymentStore interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetByOrder(ctx context.Context, orderID uuid.UUID) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
	CreatePayout(ctx context.Context, po *VendorPayout) error
	GetPendingPayouts(ctx context.Context, vendorID *uuid.UUID) ([]*VendorPayout, error)
	UpdatePayout(ctx context.Context, po *VendorPayout) error
}

type PaymentService struct {
	store   PaymentStore
	gateway PaymentGateway
	events  EventPublisher
}

func NewPaymentService(store PaymentStore, gateway PaymentGateway, events EventPublisher) *PaymentService {
	return &PaymentService{store: store, gateway: gateway, events: events}
}

func (s *PaymentService) ProcessPayment(ctx context.Context, order *Order, token, method string) (*Payment, error) {
	result, err := s.gateway.Charge(ctx, ChargeRequest{
		Amount:      order.Total,
		Currency:    "USD",
		Method:      method,
		Token:       token,
		Description: "Order " + order.OrderNumber,
	})
	if err != nil {
		return nil, err
	}
	pay := &Payment{
		ID:           uuid.New(),
		OrderID:      order.ID,
		BuyerID:      order.BuyerID,
		Amount:       order.Total,
		Currency:     "USD",
		Method:       method,
		Status:       PaymentStatusCompleted,
		GatewayTxnID: result.TransactionID,
		GatewayRef:   result.Reference,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.store.Create(ctx, pay); err != nil {
		return nil, err
	}
	// Create a pending payout record for every vendor sub-order
	for _, sub := range order.SubOrders {
		po := &VendorPayout{
			ID:               uuid.New(),
			VendorID:         sub.VendorID,
			SubOrderID:       sub.ID,
			GrossAmount:      sub.Subtotal,
			CommissionAmount: sub.CommissionAmount,
			NetAmount:        sub.VendorPayout,
			Currency:         "USD",
			Status:           PayoutStatusPending,
			CreatedAt:        time.Now(),
		}
		if err := s.store.CreatePayout(ctx, po); err != nil {
			return nil, err
		}
	}
	s.events.Publish(EventPaymentCompleted{PaymentID: pay.ID, OrderID: order.ID, Amount: pay.Amount})
	return pay, nil
}

// ProcessPayouts is called by a scheduled job (e.g. daily) to settle vendor balances.
func (s *PaymentService) ProcessPayouts(ctx context.Context) error {
	payouts, err := s.store.GetPendingPayouts(ctx, nil)
	if err != nil {
		return err
	}
	for _, po := range payouts {
		if err := s.settleVendorPayout(ctx, po); err != nil {
			s.events.Publish(EventPayoutFailed{PayoutID: po.ID, Error: err.Error()})
		}
	}
	return nil
}

func (s *PaymentService) settleVendorPayout(ctx context.Context, po *VendorPayout) error {
	if po.NetAmount.IsZero() || po.NetAmount.IsNegative() {
		return errors.New("invalid payout amount")
	}
	res, err := s.gateway.Transfer(ctx, TransferRequest{
		Amount:      po.NetAmount,
		Currency:    po.Currency,
		Description: "Payout sub-order " + po.SubOrderID.String(),
	})
	if err != nil {
		po.Status = PayoutStatusFailed
		_ = s.store.UpdatePayout(ctx, po)
		return err
	}
	now := time.Now()
	po.Status = PayoutStatusPaid
	po.PaidAt = &now
	po.TransferID = res.TransferID
	return s.store.UpdatePayout(ctx, po)
}

func (s *PaymentService) Refund(ctx context.Context, orderID uuid.UUID, amount decimal.Decimal) error {
	pay, err := s.store.GetByOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if pay.Status != PaymentStatusCompleted {
		return errors.New("only completed payments can be refunded")
	}
	if err := s.gateway.Refund(ctx, pay.GatewayTxnID, amount); err != nil {
		return err
	}
	pay.Status = PaymentStatusRefunded
	pay.UpdatedAt = time.Now()
	return s.store.Update(ctx, pay)
}
