package multivendor

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	vendorSvc  *VendorService
	productSvc *ProductService
	orderSvc   *OrderService
	paymentSvc *PaymentService
}

func NewHandler(v *VendorService, p *ProductService, o *OrderService, pay *PaymentService) *Handler {
	return &Handler{vendorSvc: v, productSvc: p, orderSvc: o, paymentSvc: pay}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Vendors
	mux.HandleFunc("POST /api/v1/vendors", h.RegisterVendor)
	mux.HandleFunc("GET /api/v1/vendors", h.ListVendors)
	mux.HandleFunc("GET /api/v1/vendors/{id}", h.GetVendor)
	// Admin
	mux.HandleFunc("POST /api/v1/admin/vendors/{id}/approve", h.ApproveVendor)
	mux.HandleFunc("POST /api/v1/admin/vendors/{id}/suspend", h.SuspendVendor)
	// Products
	mux.HandleFunc("POST /api/v1/vendors/{vendorId}/products", h.CreateProduct)
	mux.HandleFunc("GET /api/v1/vendors/{vendorId}/products", h.ListVendorProducts)
	mux.HandleFunc("GET /api/v1/products", h.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{id}", h.GetProduct)
	// Orders
	mux.HandleFunc("POST /api/v1/orders", h.PlaceOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.GetOrder)
	mux.HandleFunc("GET /api/v1/me/orders", h.GetMyOrders)
	mux.HandleFunc("GET /api/v1/vendors/{vendorId}/orders", h.GetVendorOrders)
	mux.HandleFunc("PUT /api/v1/sub-orders/{id}/tracking", h.AddTracking)
	// Payments
	mux.HandleFunc("POST /api/v1/orders/{id}/refund", h.RefundOrder)
}

func (h *Handler) RegisterVendor(w http.ResponseWriter, r *http.Request) {
	var req CreateVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	v, err := h.vendorSvc.Register(r.Context(), userID(r), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) ListVendors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	vendors, total, err := h.vendorSvc.store.List(r.Context(), VendorFilter{
		Category:  q.Get("category"),
		Search:    q.Get("search"),
		Page:      qInt(q, "page", 1),
		Limit:     qInt(q, "limit", 20),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vendors")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": vendors, "total": total})
}

func (h *Handler) GetVendor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vendor id")
		return
	}
	v, err := h.vendorSvc.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "vendor not found")
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) ApproveVendor(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(r.PathValue("id"))
	if err := h.vendorSvc.Approve(r.Context(), id); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *Handler) SuspendVendor(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(r.PathValue("id"))
	var body struct{ Reason string `json:"reason"` }
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.vendorSvc.Suspend(r.Context(), id, body.Reason); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(r.PathValue("vendorId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vendor id")
		return
	}
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p, err := h.productSvc.Create(r.Context(), vendorID, req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) ListVendorProducts(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(r.PathValue("vendorId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vendor id")
		return
	}
	products, total, err := h.productSvc.store.GetByVendor(r.Context(), vendorID, ProductFilter{
		Page: qInt(r.URL.Query(), "page", 1), Limit: qInt(r.URL.Query(), "limit", 20),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products, "total": total})
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	products, total, err := h.productSvc.store.List(r.Context(), ProductFilter{
		Category: q.Get("category"), Search: q.Get("search"),
		Page: qInt(q, "page", 1), Limit: qInt(q, "limit", 20),
		SortBy: q.Get("sort_by"), SortOrder: q.Get("sort_order"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products, "total": total})
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	p, err := h.productSvc.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var cart CartInput
	if err := json.NewDecoder(r.Body).Decode(&cart); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cart.BuyerID = userID(r)
	order, err := h.orderSvc.PlaceOrder(r.Context(), cart)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	order, err := h.orderSvc.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orderSvc.store.GetByBuyer(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) GetVendorOrders(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(r.PathValue("vendorId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vendor id")
		return
	}
	subs, total, err := h.orderSvc.store.GetByVendor(r.Context(), vendorID, OrderFilter{
		Page: qInt(r.URL.Query(), "page", 1), Limit: qInt(r.URL.Query(), "limit", 20),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch vendor orders")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": subs, "total": total})
}

func (h *Handler) AddTracking(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sub-order id")
		return
	}
	var u TrackingUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.orderSvc.AddTracking(r.Context(), id, u); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "tracking updated"})
}

func (h *Handler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var body struct {
		Amount decimal.Decimal `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.paymentSvc.Refund(r.Context(), id, body.Amount); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "refund initiated"})
}

// helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func userID(r *http.Request) uuid.UUID {
	id, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	return id
}

func qInt(q interface{ Get(string) string }, key string, def int) int {
	v := q.Get(key)
	if v == "" {
		return def
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}
