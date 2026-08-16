package multivendor

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BaseCurrency is the settlement currency every ledger entry is normalized to.
// Deposits made in any other currency are converted to this before being
// recorded so balances across the platform are directly comparable.
const BaseCurrency = "USD"

// moneyScale is the number of fractional digits kept for monetary amounts.
const moneyScale int32 = 2

var (
	ErrUnknownCurrency = errors.New("multivendor: unknown currency")
	ErrNonPositive     = errors.New("multivendor: amount must be positive")
	ErrEmptySplit      = errors.New("multivendor: split has no rules")
	ErrSplitNotWhole   = errors.New("multivendor: split percentages must sum to 100%")
)

// ExchangeRates converts monetary amounts between currencies. Each rate is
// quoted as the number of units of that currency per one unit of the base
// currency (e.g. base=USD, "EUR": 0.92 means 1 USD = 0.92 EUR). The base
// currency always has an implicit rate of 1. The zero value is not usable;
// construct one with NewExchangeRates or DefaultExchangeRates.
type ExchangeRates struct {
	mu    sync.RWMutex
	base  string
	rates map[string]decimal.Decimal
}

// NewExchangeRates builds a rate table for the given base currency. The base
// is always registered at a rate of 1, even if omitted from rates.
func NewExchangeRates(base string, rates map[string]float64) *ExchangeRates {
	er := &ExchangeRates{
		base:  base,
		rates: map[string]decimal.Decimal{base: decimal.NewFromInt(1)},
	}
	for code, r := range rates {
		er.rates[code] = decimal.NewFromFloat(r)
	}
	return er
}

// DefaultExchangeRates returns a USD-based table seeded with a fixed set of
// reference rates. In production these would be refreshed from a rate feed;
// they are fixed here so conversions are deterministic and testable.
func DefaultExchangeRates() *ExchangeRates {
	return NewExchangeRates(BaseCurrency, map[string]float64{
		"USD": 1.0,
		"EUR": 0.92,
		"GBP": 0.79,
		"JPY": 155.0,
		"CAD": 1.36,
		"AUD": 1.51,
	})
}

// SetRate adds or updates the rate for a currency (units per one base unit).
func (er *ExchangeRates) SetRate(currency string, rate float64) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.rates[currency] = decimal.NewFromFloat(rate)
}

func (er *ExchangeRates) rateFor(currency string) (decimal.Decimal, bool) {
	er.mu.RLock()
	defer er.mu.RUnlock()
	r, ok := er.rates[currency]
	return r, ok
}

// ToBase converts an amount denominated in currency into the base currency.
// Because rates are quoted as foreign-per-base, this divides by the rate —
// the step the prototype dropped, which left conversions with no effect.
func (er *ExchangeRates) ToBase(amount decimal.Decimal, currency string) (decimal.Decimal, error) {
	rate, ok := er.rateFor(currency)
	if !ok {
		return decimal.Zero, ErrUnknownCurrency
	}
	if rate.IsZero() {
		return decimal.Zero, ErrUnknownCurrency
	}
	return amount.DivRound(rate, moneyScale+4), nil
}

// FromBase converts a base-currency amount into the target currency.
func (er *ExchangeRates) FromBase(amount decimal.Decimal, currency string) (decimal.Decimal, error) {
	rate, ok := er.rateFor(currency)
	if !ok {
		return decimal.Zero, ErrUnknownCurrency
	}
	return amount.Mul(rate), nil
}

// Convert converts an amount from one currency to another, routing through the
// base currency.
func (er *ExchangeRates) Convert(amount decimal.Decimal, from, to string) (decimal.Decimal, error) {
	base, err := er.ToBase(amount, from)
	if err != nil {
		return decimal.Zero, err
	}
	return er.FromBase(base, to)
}

// PlatformAccount is the ledger account that accrues platform commission.
const PlatformAccount = "platform"

// VendorAccount returns the ledger account name that holds a vendor's balance.
func VendorAccount(vendorID uuid.UUID) string {
	return "vendor:" + vendorID.String()
}

// SplitRule directs a fraction of a deposit to a named ledger account.
type SplitRule struct {
	Account string
	// Percent is a fraction in [0,1]; the percents of a split must sum to 1.
	Percent decimal.Decimal
}

// SplitPercent is a convenience for building a SplitRule from a whole percent.
func SplitPercent(account string, percent float64) SplitRule {
	return SplitRule{Account: account, Percent: decimal.NewFromFloat(percent / 100)}
}

// LedgerEntry is a single credit to one account, recorded in the base currency
// together with the original amount and currency it originated from.
type LedgerEntry struct {
	ID             uuid.UUID       `json:"id"`
	Account        string          `json:"account"`
	Amount         decimal.Decimal `json:"amount"`
	SourceAmount   decimal.Decimal `json:"source_amount"`
	SourceCurrency string          `json:"source_currency"`
	Reference      string          `json:"reference"`
	CreatedAt      time.Time       `json:"created_at"`
}

// Ledger tracks per-account balances in the base currency and the entries that
// produced them. It is safe for concurrent use.
type Ledger struct {
	mu       sync.Mutex
	rates    *ExchangeRates
	balances map[string]decimal.Decimal
	entries  []LedgerEntry
}

// NewLedger creates an empty ledger backed by the given rate table.
func NewLedger(rates *ExchangeRates) *Ledger {
	return &Ledger{
		rates:    rates,
		balances: make(map[string]decimal.Decimal),
	}
}

// Deposit converts amount from currency into the base currency and splits it
// across the supplied accounts. Every base cent is accounted for: shares are
// rounded to the base scale and any rounding remainder is assigned to the last
// rule so the credited totals always reconcile exactly with the deposit.
func (l *Ledger) Deposit(amount decimal.Decimal, currency, reference string, split []SplitRule) ([]LedgerEntry, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrNonPositive
	}
	if len(split) == 0 {
		return nil, ErrEmptySplit
	}
	var pct decimal.Decimal
	for _, r := range split {
		pct = pct.Add(r.Percent)
	}
	if !pct.Equal(decimal.NewFromInt(1)) {
		return nil, ErrSplitNotWhole
	}

	base, err := l.rates.ToBase(amount, currency)
	if err != nil {
		return nil, err
	}
	base = base.Round(moneyScale)

	shares := make([]decimal.Decimal, len(split))
	var allocated decimal.Decimal
	for i, r := range split {
		s := base.Mul(r.Percent).Round(moneyScale)
		shares[i] = s
		allocated = allocated.Add(s)
	}
	// Push the rounding remainder onto the last share so the split is exact.
	shares[len(shares)-1] = shares[len(shares)-1].Add(base.Sub(allocated))

	now := time.Now()
	entries := make([]LedgerEntry, len(split))

	l.mu.Lock()
	defer l.mu.Unlock()
	for i, r := range split {
		entries[i] = LedgerEntry{
			ID:             uuid.New(),
			Account:        r.Account,
			Amount:         shares[i],
			SourceAmount:   amount,
			SourceCurrency: currency,
			Reference:      reference,
			CreatedAt:      now,
		}
		l.balances[r.Account] = l.balances[r.Account].Add(shares[i])
		l.entries = append(l.entries, entries[i])
	}
	return entries, nil
}

// Balance returns an account's balance in the base currency.
func (l *Ledger) Balance(account string) decimal.Decimal {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.balances[account]
}

// Balances returns a snapshot of every account balance in the base currency.
func (l *Ledger) Balances() map[string]decimal.Decimal {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]decimal.Decimal, len(l.balances))
	for k, v := range l.balances {
		out[k] = v
	}
	return out
}

// Entries returns a copy of all recorded entries, oldest first.
func (l *Ledger) Entries() []LedgerEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LedgerEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Total returns the sum of all account balances in the base currency.
func (l *Ledger) Total() decimal.Decimal {
	l.mu.Lock()
	defer l.mu.Unlock()
	var total decimal.Decimal
	// Iterate account names in sorted order for a deterministic summation.
	names := make([]string, 0, len(l.balances))
	for k := range l.balances {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		total = total.Add(l.balances[k])
	}
	return total
}
