-- Multi-vendor marketplace schema
-- PostgreSQL 15+

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ──────────────────────────────────────────────
-- Vendors
-- ──────────────────────────────────────────────

CREATE TYPE vendor_status AS ENUM ('pending', 'active', 'suspended', 'closed');

CREATE TABLE vendors (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id        UUID NOT NULL,              -- references auth.users
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT,
    logo_url        TEXT,
    banner_url      TEXT,
    email           TEXT NOT NULL,
    phone           TEXT,
    website         TEXT,
    category        TEXT NOT NULL,
    commission_rate NUMERIC(5,4) NOT NULL DEFAULT 0.10,
    rating          NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count    INT NOT NULL DEFAULT 0,
    is_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    status          vendor_status NOT NULL DEFAULT 'pending',
    address         JSONB NOT NULL DEFAULT '{}',
    bank_info       BYTEA,                      -- AES-256 encrypted JSON
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vendors_owner   ON vendors (owner_id);
CREATE INDEX idx_vendors_status  ON vendors (status);
CREATE INDEX idx_vendors_category ON vendors (category);
CREATE INDEX idx_vendors_rating  ON vendors (rating DESC);

-- ──────────────────────────────────────────────
-- Products
-- ──────────────────────────────────────────────

CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE products (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vendor_id        UUID NOT NULL REFERENCES vendors (id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL,
    description      TEXT,
    category         TEXT NOT NULL,
    tags             TEXT[] NOT NULL DEFAULT '{}',
    price            NUMERIC(12,2) NOT NULL CHECK (price >= 0),
    compare_at_price NUMERIC(12,2),
    cost_price       NUMERIC(12,2),
    sku              TEXT,
    barcode          TEXT,
    image_urls       TEXT[] NOT NULL DEFAULT '{}',
    inventory        JSONB NOT NULL DEFAULT '{"quantity":0,"track_inventory":true,"allow_backorder":false,"low_stock_alert":5}',
    status           product_status NOT NULL DEFAULT 'draft',
    rating           NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count     INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (vendor_id, slug)
);

CREATE INDEX idx_products_vendor   ON products (vendor_id);
CREATE INDEX idx_products_category ON products (category);
CREATE INDEX idx_products_status   ON products (status);
CREATE INDEX idx_products_price    ON products (price);
CREATE INDEX idx_products_tags     ON products USING GIN (tags);
CREATE INDEX idx_products_search   ON products USING GIN (to_tsvector('english', name || ' ' || COALESCE(description, '')));

CREATE TABLE product_variants (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    options    JSONB NOT NULL DEFAULT '{}',
    price      NUMERIC(12,2) NOT NULL,
    sku        TEXT,
    inventory  INT NOT NULL DEFAULT 0,
    image_url  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_variants_product ON product_variants (product_id);

-- ──────────────────────────────────────────────
-- Orders
-- ──────────────────────────────────────────────

CREATE TYPE order_status AS ENUM ('pending','confirmed','processing','shipped','delivered','cancelled','refunded');

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number     TEXT NOT NULL UNIQUE,
    buyer_id         UUID NOT NULL,
    shipping_address JSONB NOT NULL,
    billing_address  JSONB NOT NULL,
    subtotal         NUMERIC(12,2) NOT NULL DEFAULT 0,
    shipping_total   NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_total        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_total   NUMERIC(12,2) NOT NULL DEFAULT 0,
    total            NUMERIC(12,2) NOT NULL DEFAULT 0,
    coupon_code      TEXT,
    notes            TEXT,
    status           order_status NOT NULL DEFAULT 'pending',
    payment_id       UUID,
    placed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_buyer  ON orders (buyer_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_placed ON orders (placed_at DESC);

-- One row per vendor per order
CREATE TABLE sub_orders (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id           UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    vendor_id          UUID NOT NULL REFERENCES vendors (id),
    subtotal           NUMERIC(12,2) NOT NULL DEFAULT 0,
    shipping_cost      NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_amount         NUMERIC(12,2) NOT NULL DEFAULT 0,
    total              NUMERIC(12,2) NOT NULL DEFAULT 0,
    commission_rate    NUMERIC(5,4) NOT NULL,
    commission_amount  NUMERIC(12,2) NOT NULL DEFAULT 0,
    vendor_payout      NUMERIC(12,2) NOT NULL DEFAULT 0,
    status             order_status NOT NULL DEFAULT 'pending',
    tracking_number    TEXT,
    carrier            TEXT,
    estimated_delivery TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sub_orders_order  ON sub_orders (order_id);
CREATE INDEX idx_sub_orders_vendor ON sub_orders (vendor_id);
CREATE INDEX idx_sub_orders_status ON sub_orders (status);

CREATE TABLE order_items (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sub_order_id UUID NOT NULL REFERENCES sub_orders (id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products (id),
    variant_id   UUID REFERENCES product_variants (id),
    product_name TEXT NOT NULL,
    variant_name TEXT,
    sku          TEXT,
    quantity     INT NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(12,2) NOT NULL,
    line_total   NUMERIC(12,2) NOT NULL,
    image_url    TEXT
);

CREATE INDEX idx_order_items_sub_order ON order_items (sub_order_id);
CREATE INDEX idx_order_items_product   ON order_items (product_id);

-- ──────────────────────────────────────────────
-- Payments & Payouts
-- ──────────────────────────────────────────────

CREATE TYPE payment_status AS ENUM ('pending','completed','failed','refunded');
CREATE TYPE payout_status  AS ENUM ('pending','scheduled','paid','failed');

CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID NOT NULL REFERENCES orders (id),
    buyer_id        UUID NOT NULL,
    amount          NUMERIC(12,2) NOT NULL,
    currency        CHAR(3) NOT NULL DEFAULT 'USD',
    method          TEXT NOT NULL,
    status          payment_status NOT NULL DEFAULT 'pending',
    gateway_txn_id  TEXT,
    gateway_ref     TEXT,
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_order  ON payments (order_id);
CREATE INDEX idx_payments_buyer  ON payments (buyer_id);
CREATE INDEX idx_payments_status ON payments (status);

CREATE TABLE vendor_payouts (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vendor_id         UUID NOT NULL REFERENCES vendors (id),
    sub_order_id      UUID NOT NULL REFERENCES sub_orders (id),
    gross_amount      NUMERIC(12,2) NOT NULL,
    commission_amount NUMERIC(12,2) NOT NULL,
    net_amount        NUMERIC(12,2) NOT NULL,
    currency          CHAR(3) NOT NULL DEFAULT 'USD',
    status            payout_status NOT NULL DEFAULT 'pending',
    scheduled_for     TIMESTAMPTZ,
    paid_at           TIMESTAMPTZ,
    transfer_id       TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payouts_vendor ON vendor_payouts (vendor_id);
CREATE INDEX idx_payouts_status ON vendor_payouts (status);

-- ──────────────────────────────────────────────
-- Row-level security (example for Supabase)
-- ──────────────────────────────────────────────

ALTER TABLE vendors ENABLE ROW LEVEL SECURITY;
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE sub_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

-- Vendors: owner can read/write their own row; platform admins bypass via service role
CREATE POLICY vendor_owner_rw ON vendors
    USING (owner_id = auth.uid())
    WITH CHECK (owner_id = auth.uid());

-- Buyers see only their own orders
CREATE POLICY order_buyer_read ON orders FOR SELECT
    USING (buyer_id = auth.uid());

-- Vendors see only their sub-orders
CREATE POLICY sub_order_vendor_read ON sub_orders FOR SELECT
    USING (vendor_id IN (SELECT id FROM vendors WHERE owner_id = auth.uid()));

-- ──────────────────────────────────────────────
-- Triggers: updated_at maintenance
-- ──────────────────────────────────────────────

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;

CREATE TRIGGER trg_vendors_updated_at   BEFORE UPDATE ON vendors   FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_products_updated_at  BEFORE UPDATE ON products  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_orders_updated_at    BEFORE UPDATE ON orders    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_sub_orders_updated_at BEFORE UPDATE ON sub_orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_payments_updated_at  BEFORE UPDATE ON payments  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
