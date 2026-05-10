package multivendor

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type VendorStatus string

const (
	VendorStatusPending   VendorStatus = "pending"
	VendorStatusActive    VendorStatus = "active"
	VendorStatusSuspended VendorStatus = "suspended"
	VendorStatusClosed    VendorStatus = "closed"
)

type Vendor struct {
	ID             uuid.UUID    `json:"id"             db:"id"`
	OwnerID        uuid.UUID    `json:"owner_id"       db:"owner_id"`
	Name           string       `json:"name"           db:"name"`
	Slug           string       `json:"slug"           db:"slug"`
	Description    string       `json:"description"    db:"description"`
	LogoURL        string       `json:"logo_url"       db:"logo_url"`
	BannerURL      string       `json:"banner_url"     db:"banner_url"`
	Email          string       `json:"email"          db:"email"`
	Phone          string       `json:"phone"          db:"phone"`
	Website        string       `json:"website"        db:"website"`
	Category       string       `json:"category"       db:"category"`
	CommissionRate float64      `json:"commission_rate" db:"commission_rate"`
	Rating         float64      `json:"rating"         db:"rating"`
	ReviewCount    int          `json:"review_count"   db:"review_count"`
	IsVerified     bool         `json:"is_verified"    db:"is_verified"`
	Status         VendorStatus `json:"status"         db:"status"`
	Address        Address      `json:"address"        db:"address"`
	BankInfo       BankInfo     `json:"-"              db:"bank_info"`
	CreatedAt      time.Time    `json:"created_at"     db:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"     db:"updated_at"`
}

type Address struct {
	Line1      string  `json:"line1"`
	Line2      string  `json:"line2,omitempty"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	PostalCode string  `json:"postal_code"`
	Country    string  `json:"country"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
}

// BankInfo fields are stored encrypted at rest.
type BankInfo struct {
	AccountHolder string `json:"account_holder"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	BankName      string `json:"bank_name"`
	AccountType   string `json:"account_type"`
}

type VendorStore interface {
	Create(ctx context.Context, v *Vendor) error
	GetByID(ctx context.Context, id uuid.UUID) (*Vendor, error)
	GetBySlug(ctx context.Context, slug string) (*Vendor, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Vendor, error)
	List(ctx context.Context, f VendorFilter) ([]*Vendor, int, error)
	Update(ctx context.Context, v *Vendor) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status VendorStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type VendorFilter struct {
	Category   string
	Status     VendorStatus
	IsVerified *bool
	Search     string
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

type VendorService struct {
	store  VendorStore
	events EventPublisher
}

func NewVendorService(store VendorStore, events EventPublisher) *VendorService {
	return &VendorService{store: store, events: events}
}

func (s *VendorService) Register(ctx context.Context, ownerID uuid.UUID, req CreateVendorRequest) (*Vendor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	v := &Vendor{
		ID:             uuid.New(),
		OwnerID:        ownerID,
		Name:           req.Name,
		Slug:           slugify(req.Name),
		Description:    req.Description,
		Email:          req.Email,
		Phone:          req.Phone,
		Website:        req.Website,
		Category:       req.Category,
		CommissionRate: defaultCommissionRate(req.Category),
		Address:        req.Address,
		Status:         VendorStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.store.Create(ctx, v); err != nil {
		return nil, err
	}
	s.events.Publish(EventVendorRegistered{VendorID: v.ID, OwnerID: ownerID})
	return v, nil
}

func (s *VendorService) Approve(ctx context.Context, id uuid.UUID) error {
	v, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if v.Status != VendorStatusPending {
		return errors.New("vendor is not pending approval")
	}
	if err := s.store.UpdateStatus(ctx, id, VendorStatusActive); err != nil {
		return err
	}
	s.events.Publish(EventVendorApproved{VendorID: id})
	return nil
}

func (s *VendorService) Suspend(ctx context.Context, id uuid.UUID, reason string) error {
	if err := s.store.UpdateStatus(ctx, id, VendorStatusSuspended); err != nil {
		return err
	}
	s.events.Publish(EventVendorSuspended{VendorID: id, Reason: reason})
	return nil
}

type CreateVendorRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
	Website     string  `json:"website"`
	Category    string  `json:"category"`
	Address     Address `json:"address"`
}

func (r CreateVendorRequest) validate() error {
	switch {
	case r.Name == "":
		return errors.New("name is required")
	case r.Email == "":
		return errors.New("email is required")
	case r.Category == "":
		return errors.New("category is required")
	}
	return nil
}

func defaultCommissionRate(category string) float64 {
	rates := map[string]float64{
		"electronics": 0.08, "fashion": 0.12, "food": 0.15,
		"beauty": 0.12, "home": 0.10, "sports": 0.10, "books": 0.15,
	}
	if r, ok := rates[category]; ok {
		return r
	}
	return 0.10
}
