package multivendor

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	ID             uuid.UUID        `json:"id"              db:"id"`
	VendorID       uuid.UUID        `json:"vendor_id"       db:"vendor_id"`
	Name           string           `json:"name"            db:"name"`
	Slug           string           `json:"slug"            db:"slug"`
	Description    string           `json:"description"     db:"description"`
	Category       string           `json:"category"        db:"category"`
	Tags           []string         `json:"tags"            db:"tags"`
	Price          decimal.Decimal  `json:"price"           db:"price"`
	CompareAtPrice decimal.Decimal  `json:"compare_at_price" db:"compare_at_price"`
	CostPrice      decimal.Decimal  `json:"-"               db:"cost_price"`
	SKU            string           `json:"sku"             db:"sku"`
	Barcode        string           `json:"barcode"         db:"barcode"`
	ImageURLs      []string         `json:"image_urls"      db:"image_urls"`
	Variants       []ProductVariant `json:"variants"`
	Inventory      InventoryInfo    `json:"inventory"       db:"inventory"`
	Status         ProductStatus    `json:"status"          db:"status"`
	Rating         float64          `json:"rating"          db:"rating"`
	ReviewCount    int              `json:"review_count"    db:"review_count"`
	CreatedAt      time.Time        `json:"created_at"      db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"      db:"updated_at"`
}

type ProductVariant struct {
	ID        uuid.UUID         `json:"id"`
	ProductID uuid.UUID         `json:"product_id"`
	Name      string            `json:"name"`
	Options   map[string]string `json:"options"`
	Price     decimal.Decimal   `json:"price"`
	SKU       string            `json:"sku"`
	Inventory int               `json:"inventory"`
	ImageURL  string            `json:"image_url"`
}

type InventoryInfo struct {
	Quantity       int  `json:"quantity"`
	TrackInventory bool `json:"track_inventory"`
	AllowBackorder bool `json:"allow_backorder"`
	LowStockAlert  int  `json:"low_stock_alert"`
}

type ProductStore interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	GetByVendor(ctx context.Context, vendorID uuid.UUID, f ProductFilter) ([]*Product, int, error)
	List(ctx context.Context, f ProductFilter) ([]*Product, int, error)
	Update(ctx context.Context, p *Product) error
	UpdateInventory(ctx context.Context, productID uuid.UUID, delta int) error
	Delete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, query string, f ProductFilter) ([]*Product, int, error)
}

type ProductFilter struct {
	VendorID  *uuid.UUID
	Category  string
	MinPrice  *decimal.Decimal
	MaxPrice  *decimal.Decimal
	Status    ProductStatus
	InStock   *bool
	Tags      []string
	Search    string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type ProductService struct {
	store       ProductStore
	vendorStore VendorStore
	events      EventPublisher
}

func NewProductService(store ProductStore, vendorStore VendorStore, events EventPublisher) *ProductService {
	return &ProductService{store: store, vendorStore: vendorStore, events: events}
}

func (s *ProductService) Create(ctx context.Context, vendorID uuid.UUID, req CreateProductRequest) (*Product, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	v, err := s.vendorStore.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	if v.Status != VendorStatusActive {
		return nil, errors.New("vendor must be active to create products")
	}
	p := &Product{
		ID:          uuid.New(),
		VendorID:    vendorID,
		Name:        req.Name,
		Slug:        slugify(req.Name),
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Price:       req.Price,
		SKU:         req.SKU,
		ImageURLs:   req.ImageURLs,
		Inventory: InventoryInfo{
			Quantity:       req.InitialStock,
			TrackInventory: true,
			LowStockAlert:  5,
		},
		Status:    ProductStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.store.Create(ctx, p); err != nil {
		return nil, err
	}
	s.events.Publish(EventProductCreated{ProductID: p.ID, VendorID: vendorID})
	return p, nil
}

// ReserveInventory decrements stock atomically. Call before confirming payment.
func (s *ProductService) ReserveInventory(ctx context.Context, productID uuid.UUID, qty int) error {
	p, err := s.store.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if p.Inventory.TrackInventory && p.Inventory.Quantity < qty && !p.Inventory.AllowBackorder {
		return ErrInsufficientInventory
	}
	return s.store.UpdateInventory(ctx, productID, -qty)
}

// ReleaseInventory restores stock on order cancellation.
func (s *ProductService) ReleaseInventory(ctx context.Context, productID uuid.UUID, qty int) error {
	return s.store.UpdateInventory(ctx, productID, qty)
}

var ErrInsufficientInventory = errors.New("insufficient inventory")

type CreateProductRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	Tags         []string        `json:"tags"`
	Price        decimal.Decimal `json:"price"`
	SKU          string          `json:"sku"`
	ImageURLs    []string        `json:"image_urls"`
	InitialStock int             `json:"initial_stock"`
}

func (r CreateProductRequest) validate() error {
	switch {
	case r.Name == "":
		return errors.New("name is required")
	case r.Price.IsNegative() || r.Price.IsZero():
		return errors.New("price must be positive")
	}
	return nil
}
