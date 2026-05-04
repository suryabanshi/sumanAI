# Multi-Vendor Marketplace Architecture

A production-grade reference architecture for a Shopify-like multi-vendor platform.
Backend: Go (`sumanai`). Mobile frontend: SwiftUI (`swiftui`).

---

## 1. System Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Buyers (SwiftUI)                    │
│  Browse → Add to Cart → Checkout → Track Orders         │
└────────────────────────┬────────────────────────────────┘
                         │ HTTPS / REST
┌────────────────────────▼────────────────────────────────┐
│                    API Gateway / BFF                    │
│          Auth · Rate limiting · Request routing         │
└──┬──────────┬──────────┬──────────┬──────────┬──────────┘
   │          │          │          │          │
 ┌─▼──┐   ┌──▼─┐   ┌────▼─┐  ┌────▼─┐  ┌────▼──┐
 │Vnd │   │Prod│   │Order │  │Pay   │  │Search │
 │Svc │   │Svc │   │ Svc  │  │ Svc  │  │ Svc   │
 └─┬──┘   └──┬─┘   └──┬───┘  └──┬───┘  └───────┘
   │          │        │         │
   └──────────┴────────┴─────────┘
                       │
              ┌────────▼────────┐
              │   PostgreSQL    │
              │  (primary DB)   │
              └────────────────┘
                       │
              ┌────────▼────────┐
              │  Event Bus      │
              │ (NATS / Kafka)  │
              └────────────────┘
                       │
         ┌─────────────┼─────────────┐
     ┌───▼───┐   ┌─────▼───┐  ┌─────▼────┐
     │Notify │   │Analytics│  │Payout Job│
     │  Svc  │   │   Svc   │  │(cron)    │
     └───────┘   └─────────┘  └──────────┘
```

---

## 2. Core Domain Entities

### Vendor
Represents a seller on the platform. A vendor is created by a user (owner), goes through an approval flow, and earns a configurable commission split.

| Field | Notes |
|-------|-------|
| `id` | UUID primary key |
| `owner_id` | FK to auth users table |
| `slug` | URL-safe unique name |
| `commission_rate` | Per-category default (e.g. 0.10 = 10%) |
| `status` | `pending → active → suspended/closed` |
| `bank_info` | AES-256 encrypted at rest |

### Product
Belongs to exactly one vendor. Supports variants (size/color/etc.) with per-variant pricing and inventory tracking.

### Order + SubOrder
The key multi-vendor split:
- One **Order** per buyer checkout, regardless of how many vendors.
- One **SubOrder** per vendor in that cart.
- Each SubOrder carries its own `commission_amount` and `vendor_payout` so accounting is unambiguous.

### Payment + VendorPayout
- The buyer pays once to the **platform**.
- The platform holds funds and creates a `VendorPayout` record (status `pending`) per sub-order.
- A nightly cron job processes pending payouts, transfers `net_amount` to each vendor's bank/Stripe Connect account, and marks them `paid`.

---

## 3. Vendor Onboarding Flow

```
User registers
    └─► POST /api/v1/vendors          → status: pending
            │
            ▼
    Platform admin reviews
    POST /api/v1/admin/vendors/{id}/approve
            │
            ▼
        status: active
        EventVendorApproved published
            │
            ▼
    Vendor creates products
    POST /api/v1/vendors/{id}/products → status: draft
            │
            ▼
    Vendor publishes product           → status: active
```

---

## 4. Multi-Vendor Order Flow

```
Buyer adds items from multiple vendors
    └─► POST /api/v1/orders
            │
            ▼
    OrderService.PlaceOrder()
      1. Fetch all products → group by vendor_id
      2. Build SubOrder per vendor
         - commission = subtotal × vendor.commission_rate
         - vendor_payout = total − commission
      3. Reserve inventory (compensating saga on failure)
      4. Persist Order + SubOrders + OrderItems
      5. Publish EventOrderPlaced
            │
            ▼
    PaymentService.ProcessPayment()
      1. Charge buyer via gateway (Stripe/etc.)
      2. Create Payment record (completed)
      3. Create VendorPayout per SubOrder (pending)
      4. Publish EventPaymentCompleted
            │
            ▼
    Vendor ships → PUT /api/v1/sub-orders/{id}/tracking
      SubOrder.status → shipped
      EventOrderShipped published → buyer push notification
            │
            ▼
    Nightly payout job
      GET pending VendorPayouts
      Transfer net_amount to vendor bank
      VendorPayout.status → paid
```

---

## 5. Commission & Payout Model

```
Buyer pays:    $100.00
               ────────
Vendor A
  subtotal:    $ 60.00
  shipping:    $  0.00  (free over $50)
  commission:  $  4.80  (8% electronics)
  payout:      $ 55.20  ← transferred to vendor

Vendor B
  subtotal:    $ 40.00
  shipping:    $  5.99
  commission:  $  6.00  (15% books)
  payout:      $ 39.99  ← transferred to vendor

Platform keeps: $4.80 + $6.00 = $10.80 gross margin
```

Commission rates are set per vendor category at onboarding and stored on the `Vendor` record. They snapshot into `SubOrder.commission_rate` at order time so historical records are immutable.

---

## 6. Inventory Reservation (Saga Pattern)

The `PlaceOrder` function uses a compensating transaction approach:

```go
var reserved []CartItemInput
for _, item := range cart.Items {
    if err := productSvc.ReserveInventory(ctx, item.ProductID, item.Quantity); err != nil {
        // compensate: release all previously reserved items
        for _, done := range reserved {
            _ = productSvc.ReleaseInventory(ctx, done.ProductID, done.Quantity)
        }
        return nil, err
    }
    reserved = append(reserved, item)
}
```

In a distributed system, replace this with a proper saga orchestrator or use `SELECT ... FOR UPDATE` at the DB level for strict consistency.

---

## 7. API Reference

### Vendors
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/vendors` | Register a new vendor |
| `GET` | `/api/v1/vendors` | List vendors (filterable by category, status, search) |
| `GET` | `/api/v1/vendors/{id}` | Get vendor detail |
| `POST` | `/api/v1/admin/vendors/{id}/approve` | Admin: approve vendor |
| `POST` | `/api/v1/admin/vendors/{id}/suspend` | Admin: suspend vendor |

### Products
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/vendors/{vendorId}/products` | Create product |
| `GET` | `/api/v1/vendors/{vendorId}/products` | List vendor's products |
| `GET` | `/api/v1/products` | Browse all active products |
| `GET` | `/api/v1/products/{id}` | Product detail |

### Orders
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/orders` | Place order (splits by vendor automatically) |
| `GET` | `/api/v1/orders/{id}` | Get order with all sub-orders |
| `GET` | `/api/v1/me/orders` | Buyer's order history |
| `GET` | `/api/v1/vendors/{vendorId}/orders` | Vendor's incoming sub-orders |
| `PUT` | `/api/v1/sub-orders/{id}/tracking` | Vendor adds tracking info |
| `POST` | `/api/v1/orders/{id}/refund` | Issue refund |

---

## 8. Database Design Decisions

**Why `SubOrder` as a first-class table?**
Vendors need to independently manage their shipments, update tracking, and view their earnings without seeing other vendors' data. A join table would blur this boundary.

**Why snapshot `commission_rate` into `SubOrder`?**
If the platform changes a vendor's rate later, historical orders must remain accurate. Snapshotting at order time is the correct approach.

**Why encrypt `bank_info`?**
PCI-DSS and regional data protection regulations require sensitive financial data to be encrypted at rest. The Go service decrypts only when initiating a payout transfer.

**Row-Level Security (RLS)**
PostgreSQL RLS policies enforce data isolation at the database layer, not just the application layer — a defense-in-depth measure particularly valuable when using Supabase or direct DB connections.

---

## 9. Event-Driven Integrations

| Event | Consumers |
|-------|-----------|
| `EventVendorRegistered` | Send welcome email; create onboarding checklist |
| `EventVendorApproved` | Notify vendor; enable product creation |
| `EventOrderPlaced` | Analytics; fraud detection; vendor dashboard notification |
| `EventPaymentCompleted` | Receipt email to buyer; trigger fulfillment |
| `EventOrderShipped` | Push notification to buyer; update order status |
| `EventPayoutFailed` | Alert platform ops team; retry queue |

Implement `EventPublisher` with any broker — start with an in-process channel for development, switch to NATS or Kafka for production.

---

## 10. Security Checklist

- [ ] JWT authentication on all non-public endpoints
- [ ] Vendor ownership verified before any product/order mutation (middleware)
- [ ] Admin endpoints behind separate role check
- [ ] `bank_info` encrypted with AES-256-GCM, key in AWS KMS / Vault
- [ ] Payment tokens are single-use and never logged
- [ ] All monetary arithmetic uses `decimal.Decimal` — never float64
- [ ] Input validation before any DB write
- [ ] RLS policies as a second layer of isolation
- [ ] Rate limiting on order placement (prevent cart stuffing)
- [ ] Idempotency keys on payment endpoints

---

## 11. SwiftUI Frontend Structure

```
MultiVendorMarketplace/
├── App/
│   └── MarketplaceApp.swift       # Entry point, environment injection
├── Models/
│   ├── Vendor.swift
│   ├── Product.swift
│   ├── Cart.swift
│   ├── Order.swift
│   └── Payment.swift
├── ViewModels/
│   ├── MarketplaceViewModel.swift  # Home feed, search
│   ├── CartViewModel.swift         # Cart state + checkout
│   └── OrderViewModel.swift        # Order history
├── Views/
│   ├── HomeView.swift              # Featured vendors + products
│   ├── VendorStoreView.swift       # Individual vendor storefront
│   ├── ProductDetailView.swift     # PDP with variant selection
│   ├── CartView.swift              # Cart grouped by vendor
│   ├── CheckoutView.swift          # 3-step: address → payment → review
│   └── OrderDetailView.swift       # Order tracking per sub-order
└── Services/
    └── APIService.swift            # Typed REST client
```

Key SwiftUI patterns used:
- `@EnvironmentObject` for cart and marketplace state shared across the tab bar
- `async/await` + `@MainActor` for all network calls
- `Task.sleep` debounce on search input
- Persistent cart via `UserDefaults` + `Codable`
- `safeAreaInset` for sticky add-to-cart and checkout bars
