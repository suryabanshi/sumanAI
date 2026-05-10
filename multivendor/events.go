package multivendor

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EventPublisher is implemented by any async message bus (Kafka, NATS, in-process).
type EventPublisher interface {
	Publish(event any)
}

type EventVendorRegistered struct {
	VendorID  uuid.UUID `json:"vendor_id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Timestamp time.Time `json:"timestamp"`
}

type EventVendorApproved struct {
	VendorID  uuid.UUID `json:"vendor_id"`
	Timestamp time.Time `json:"timestamp"`
}

type EventVendorSuspended struct {
	VendorID  uuid.UUID `json:"vendor_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

type EventProductCreated struct {
	ProductID uuid.UUID `json:"product_id"`
	VendorID  uuid.UUID `json:"vendor_id"`
	Timestamp time.Time `json:"timestamp"`
}

type EventOrderPlaced struct {
	OrderID     uuid.UUID       `json:"order_id"`
	BuyerID     uuid.UUID       `json:"buyer_id"`
	VendorCount int             `json:"vendor_count"`
	Total       decimal.Decimal `json:"total"`
	Timestamp   time.Time       `json:"timestamp"`
}

type EventOrderShipped struct {
	SubOrderID     uuid.UUID `json:"sub_order_id"`
	TrackingNumber string    `json:"tracking_number"`
	Timestamp      time.Time `json:"timestamp"`
}

type EventPaymentCompleted struct {
	PaymentID uuid.UUID       `json:"payment_id"`
	OrderID   uuid.UUID       `json:"order_id"`
	Amount    decimal.Decimal `json:"amount"`
	Timestamp time.Time       `json:"timestamp"`
}

type EventPayoutFailed struct {
	PayoutID  uuid.UUID `json:"payout_id"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}
