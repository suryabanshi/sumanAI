package multivendor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type Order struct {
	ID              uuid.UUID       `json:"id"               db:"id"`
	OrderNumber     string          `json:"order_number"     db:"order_number"`
	BuyerID         uuid.UUID       `json:"buyer_id"         db:"buyer_id"`
	SubOrders       []*SubOrder     `json:"sub_orders"`
	ShippingAddress Address         `json:"shipping_address" db:"shipping_address"`
	BillingAddress  Address         `json:"billing_address"  db:"billing_address"`
	Subtotal        decimal.Decimal `json:"subtotal"         db:"subtotal"`
	ShippingTotal   decimal.Decimal `json:"shipping_total"   db:"shipping_total"`
	TaxTotal        decimal.Decimal `json:"tax_total"        db:"tax_total"`
	DiscountTotal   decimal.Decimal `json:"discount_total"   db:"discount_total"`
	Total           decimal.Decimal `json:"total"            db:"total"`
	CouponCode      string          `json:"coupon_code,omitempty" db:"coupon_code"`
	Notes           string          `json:"notes,omitempty"  db:"notes"`
	Status          OrderStatus     `json:"status"           db:"status"`
	PaymentID       uuid.UUID       `json:"payment_id"       db:"payment_id"`
	PlacedAt        time.Time       `json:"placed_at"        db:"placed_at"`
	UpdatedAt       time.Time       `json:"updated_at"       db:"updated_at"`
}

// SubOrder is one vendor's slice of a multi-vendor order.
type SubOrder struct {
	ID                uuid.UUID       `json:"id"                  db:"id"`
	OrderID           uuid.UUID       `json:"order_id"            db:"order_id"`
	VendorID          uuid.UUID       `json:"vendor_id"           db:"vendor_id"`
	VendorName        string          `json:"vendor_name"`
	Items             []*OrderItem    `json:"items"`
	Subtotal          decimal.Decimal `json:"subtotal"            db:"subtotal"`
	ShippingCost      decimal.Decimal `json:"shipping_cost"       db:"shipping_cost"`
	TaxAmount         decimal.Decimal `json:"tax_amount"          db:"tax_amount"`
	Total             decimal.Decimal `json:"total"               db:"total"`
	CommissionRate    decimal.Decimal `json:"commission_rate"     db:"commission_rate"`
	CommissionAmount  decimal.Decimal `json:"commission_amount"   db:"commission_amount"`
	VendorPayout      decimal.Decimal `json:"vendor_payout"       db:"vendor_payout"`
	Status            OrderStatus     `json:"status"              db:"status"`
	TrackingNumber    string          `json:"tracking_number,omitempty" db:"tracking_number"`
	Carrier           string          `json:"carrier,omitempty"   db:"carrier"`
	EstimatedDelivery *time.Time      `json:"estimated_delivery,omitempty" db:"estimated_delivery"`
	CreatedAt         time.Time       `json:"created_at"          db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"          db:"updated_at"`
}

type OrderItem struct {
	ID          uuid.UUID       `json:"id"           db:"id"`
	SubOrderID  uuid.UUID       `json:"sub_order_id" db:"sub_order_id"`
	ProductID   uuid.UUID       `json:"product_id"   db:"product_id"`
	VariantID   *uuid.UUID      `json:"variant_id,omitempty" db:"variant_id"`
	ProductName string          `json:"product_name" db:"product_name"`
	VariantName string          `json:"variant_name,omitempty" db:"variant_name"`
	SKU         string          `json:"sku"          db:"sku"`
	Quantity    int             `json:"quantity"     db:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"   db:"unit_price"`
	LineTotal   decimal.Decimal `json:"line_total"   db:"line_total"`
	ImageURL    string          `json:"image_url,omitempty" db:"image_url"`
}

type CartInput struct {
	BuyerID         uuid.UUID
	Items           []CartItemInput
	ShippingAddress Address
	BillingAddress  Address
	CouponCode      string
	Notes           string
}

type CartItemInput struct {
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
	Quantity  int        `json:"quantity"`
}

type OrderStore interface {
	Create(ctx context.Context, o *Order) error
	CreateSubOrder(ctx context.Context, s *SubOrder) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetByBuyer(ctx context.Context, buyerID uuid.UUID) ([]*Order, error)
	GetByVendor(ctx context.Context, vendorID uuid.UUID, f OrderFilter) ([]*SubOrder, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
	UpdateSubOrderStatus(ctx context.Context, id uuid.UUID, status OrderStatus, t TrackingUpdate) error
}

type OrderFilter struct {
	Status    OrderStatus
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

type TrackingUpdate struct {
	TrackingNumber    string
	Carrier           string
	EstimatedDelivery *time.Time
}

type OrderService struct {
	store       OrderStore
	productSvc  *ProductService
	vendorStore VendorStore
	events      EventPublisher
}

func NewOrderService(store OrderStore, productSvc *ProductService, vendorStore VendorStore, events EventPublisher) *OrderService {
	return &OrderService{store: store, productSvc: productSvc, vendorStore: vendorStore, events: events}
}

// PlaceOrder splits cart items by vendor and creates one SubOrder per vendor.
// Inventory is reserved transactionally; on failure a compensating release is issued.
func (s *OrderService) PlaceOrder(ctx context.Context, cart CartInput) (*Order, error) {
	// Map product IDs to vendors
	type productMeta struct {
		product *Product
	}
	productMap := make(map[uuid.UUID]*Product)
	for _, item := range cart.Items {
		p, err := s.productSvc.store.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s: %w", item.ProductID, err)
		}
		productMap[item.ProductID] = p
	}

	// Group items by vendor
	vendorItems := make(map[uuid.UUID][]CartItemInput)
	for _, item := range cart.Items {
		vid := productMap[item.ProductID].VendorID
		vendorItems[vid] = append(vendorItems[vid], item)
	}

	order := &Order{
		ID:              uuid.New(),
		OrderNumber:     generateOrderNumber(),
		BuyerID:         cart.BuyerID,
		ShippingAddress: cart.ShippingAddress,
		BillingAddress:  cart.BillingAddress,
		CouponCode:      cart.CouponCode,
		Notes:           cart.Notes,
		Status:          OrderStatusPending,
		PlacedAt:        time.Now(),
		UpdatedAt:       time.Now(),
	}

	var orderTotal decimal.Decimal
	for vendorID, items := range vendorItems {
		sub, err := s.buildSubOrder(ctx, order.ID, vendorID, items, productMap)
		if err != nil {
			return nil, fmt.Errorf("sub-order for vendor %s: %w", vendorID, err)
		}
		order.SubOrders = append(order.SubOrders, sub)
		orderTotal = orderTotal.Add(sub.Total)
	}
	order.Total = orderTotal

	if err := s.store.Create(ctx, order); err != nil {
		return nil, err
	}

	// Reserve inventory — compensate on partial failure
	var reserved []CartItemInput
	for _, item := range cart.Items {
		if err := s.productSvc.ReserveInventory(ctx, item.ProductID, item.Quantity); err != nil {
			for _, done := range reserved {
				_ = s.productSvc.ReleaseInventory(ctx, done.ProductID, done.Quantity)
			}
			return nil, fmt.Errorf("reserving %s: %w", item.ProductID, err)
		}
		reserved = append(reserved, item)
	}

	s.events.Publish(EventOrderPlaced{
		OrderID:     order.ID,
		BuyerID:     cart.BuyerID,
		VendorCount: len(order.SubOrders),
		Total:       order.Total,
	})
	return order, nil
}

func (s *OrderService) buildSubOrder(ctx context.Context, orderID, vendorID uuid.UUID, items []CartItemInput, productMap map[uuid.UUID]*Product) (*SubOrder, error) {
	v, err := s.vendorStore.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	sub := &SubOrder{
		ID:             uuid.New(),
		OrderID:        orderID,
		VendorID:       vendorID,
		VendorName:     v.Name,
		CommissionRate: decimal.NewFromFloat(v.CommissionRate),
		Status:         OrderStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	var subtotal decimal.Decimal
	for _, item := range items {
		p := productMap[item.ProductID]
		unit := p.Price
		line := unit.Mul(decimal.NewFromInt(int64(item.Quantity)))
		sub.Items = append(sub.Items, &OrderItem{
			ID:          uuid.New(),
			SubOrderID:  sub.ID,
			ProductID:   p.ID,
			ProductName: p.Name,
			SKU:         p.SKU,
			Quantity:    item.Quantity,
			UnitPrice:   unit,
			LineTotal:   line,
		})
		subtotal = subtotal.Add(line)
	}
	shipping := freeShippingOver50(subtotal)
	commission := subtotal.Mul(sub.CommissionRate)
	sub.Subtotal = subtotal
	sub.ShippingCost = shipping
	sub.Total = subtotal.Add(shipping)
	sub.CommissionAmount = commission
	sub.VendorPayout = sub.Total.Sub(commission)
	return sub, nil
}

func (s *OrderService) AddTracking(ctx context.Context, subOrderID uuid.UUID, u TrackingUpdate) error {
	if err := s.store.UpdateSubOrderStatus(ctx, subOrderID, OrderStatusShipped, u); err != nil {
		return err
	}
	s.events.Publish(EventOrderShipped{SubOrderID: subOrderID, TrackingNumber: u.TrackingNumber})
	return nil
}

func freeShippingOver50(subtotal decimal.Decimal) decimal.Decimal {
	if subtotal.GreaterThanOrEqual(decimal.NewFromFloat(50)) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(5.99)
}

func generateOrderNumber() string {
	return fmt.Sprintf("MV-%d", time.Now().UnixMilli())
}
