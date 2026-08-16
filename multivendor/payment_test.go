package multivendor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeGateway struct {
	lastCharge ChargeRequest
}

func (g *fakeGateway) Charge(_ context.Context, req ChargeRequest) (*ChargeResult, error) {
	g.lastCharge = req
	return &ChargeResult{TransactionID: "txn_1", Reference: "ref_1", Status: "ok"}, nil
}
func (g *fakeGateway) Refund(context.Context, string, decimal.Decimal) error { return nil }
func (g *fakeGateway) Transfer(context.Context, TransferRequest) (*TransferResult, error) {
	return &TransferResult{TransferID: "tr_1", Status: "ok"}, nil
}

type fakeStore struct {
	payouts []*VendorPayout
}

func (s *fakeStore) Create(context.Context, *Payment) error                  { return nil }
func (s *fakeStore) GetByID(context.Context, uuid.UUID) (*Payment, error)    { return nil, nil }
func (s *fakeStore) GetByOrder(context.Context, uuid.UUID) (*Payment, error) { return nil, nil }
func (s *fakeStore) Update(context.Context, *Payment) error                  { return nil }
func (s *fakeStore) CreatePayout(_ context.Context, po *VendorPayout) error {
	s.payouts = append(s.payouts, po)
	return nil
}
func (s *fakeStore) GetPendingPayouts(context.Context, *uuid.UUID) ([]*VendorPayout, error) {
	return s.payouts, nil
}
func (s *fakeStore) UpdatePayout(context.Context, *VendorPayout) error { return nil }

type noopEvents struct{}

func (noopEvents) Publish(any) {}

func TestProcessPaymentRecordsLedgerSplit(t *testing.T) {
	ledger := NewLedger(DefaultExchangeRates())
	gw := &fakeGateway{}
	svc := NewPaymentService(&fakeStore{}, gw, noopEvents{}).WithLedger(ledger)

	vendorID := uuid.New()
	// A sub-order totalling 92 EUR (= 100 USD) with 20 EUR commission.
	sub := &SubOrder{
		ID:               uuid.New(),
		VendorID:         vendorID,
		Subtotal:         dec("92"),
		Total:            dec("92"),
		CommissionAmount: dec("13.8"), // 15% of 92
		VendorPayout:     dec("78.2"),
	}
	order := &Order{
		ID:          uuid.New(),
		OrderNumber: "MV-1",
		BuyerID:     uuid.New(),
		Currency:    "EUR",
		Total:       dec("92"),
		SubOrders:   []*SubOrder{sub},
	}

	if _, err := svc.ProcessPayment(context.Background(), order, "tok", "card"); err != nil {
		t.Fatalf("ProcessPayment: %v", err)
	}

	// The gateway must be charged in the order's currency, not a hardcoded USD.
	if gw.lastCharge.Currency != "EUR" {
		t.Fatalf("charge currency = %s, want EUR", gw.lastCharge.Currency)
	}

	// 92 EUR settles to 100 USD in the ledger, split platform 15 / vendor 85.
	if !ledger.Total().Round(2).Equal(dec("100")) {
		t.Fatalf("ledger total = %s, want 100", ledger.Total().Round(2))
	}
	if !ledger.Balance(PlatformAccount).Round(2).Equal(dec("15")) {
		t.Fatalf("platform = %s, want 15", ledger.Balance(PlatformAccount).Round(2))
	}
	if !ledger.Balance(VendorAccount(vendorID)).Round(2).Equal(dec("85")) {
		t.Fatalf("vendor = %s, want 85", ledger.Balance(VendorAccount(vendorID)).Round(2))
	}
}

func TestProcessPaymentDefaultsToBaseCurrency(t *testing.T) {
	gw := &fakeGateway{}
	svc := NewPaymentService(&fakeStore{}, gw, noopEvents{})
	order := &Order{
		ID:        uuid.New(),
		BuyerID:   uuid.New(),
		Total:     dec("10"),
		SubOrders: nil,
	}
	if _, err := svc.ProcessPayment(context.Background(), order, "tok", "card"); err != nil {
		t.Fatalf("ProcessPayment: %v", err)
	}
	if gw.lastCharge.Currency != BaseCurrency {
		t.Fatalf("charge currency = %s, want %s", gw.lastCharge.Currency, BaseCurrency)
	}
}
