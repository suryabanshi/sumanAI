package multivendor

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestExchangeRatesToBase(t *testing.T) {
	er := DefaultExchangeRates()

	// USD is the base: converting it leaves the amount unchanged.
	got, err := er.ToBase(dec("100"), "USD")
	if err != nil {
		t.Fatalf("ToBase USD: %v", err)
	}
	if !got.Equal(dec("100")) {
		t.Fatalf("USD ToBase = %s, want 100", got)
	}

	// EUR is quoted at 0.92 per USD, so 92 EUR must be 100 USD. The prototype
	// dropped this division, which made conversions have no effect.
	got, err = er.ToBase(dec("92"), "EUR")
	if err != nil {
		t.Fatalf("ToBase EUR: %v", err)
	}
	if !got.Round(2).Equal(dec("100")) {
		t.Fatalf("92 EUR ToBase = %s, want 100", got.Round(2))
	}
}

func TestExchangeRatesRoundTrip(t *testing.T) {
	er := DefaultExchangeRates()
	base, err := er.ToBase(dec("155"), "JPY")
	if err != nil {
		t.Fatalf("ToBase: %v", err)
	}
	if !base.Round(2).Equal(dec("1")) {
		t.Fatalf("155 JPY = %s USD, want 1", base.Round(2))
	}
	back, err := er.FromBase(base, "JPY")
	if err != nil {
		t.Fatalf("FromBase: %v", err)
	}
	if !back.Round(0).Equal(dec("155")) {
		t.Fatalf("round trip = %s JPY, want 155", back.Round(0))
	}
}

func TestExchangeRatesUnknownCurrency(t *testing.T) {
	er := DefaultExchangeRates()
	if _, err := er.ToBase(dec("10"), "XYZ"); err != ErrUnknownCurrency {
		t.Fatalf("ToBase unknown = %v, want ErrUnknownCurrency", err)
	}
	if _, err := er.Convert(dec("10"), "USD", "XYZ"); err != ErrUnknownCurrency {
		t.Fatalf("Convert unknown = %v, want ErrUnknownCurrency", err)
	}
}

func TestLedgerDepositSplitReconciles(t *testing.T) {
	l := NewLedger(DefaultExchangeRates())
	split := []SplitRule{
		SplitPercent("Main Coach", 70),
		SplitPercent("Partner Guardian", 30),
	}
	entries, err := l.Deposit(dec("100"), "USD", "ref-1", split)
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if !l.Balance("Main Coach").Equal(dec("70")) {
		t.Fatalf("Main Coach = %s, want 70", l.Balance("Main Coach"))
	}
	if !l.Balance("Partner Guardian").Equal(dec("30")) {
		t.Fatalf("Partner Guardian = %s, want 30", l.Balance("Partner Guardian"))
	}
	if !l.Total().Equal(dec("100")) {
		t.Fatalf("Total = %s, want 100", l.Total())
	}
}

// TestLedgerSplitPennyExact checks that rounding never loses or invents money:
// splitting an amount that does not divide evenly still sums back exactly.
func TestLedgerSplitPennyExact(t *testing.T) {
	l := NewLedger(DefaultExchangeRates())
	split := []SplitRule{
		SplitPercent("a", 70),
		SplitPercent("b", 30),
	}
	// 0.70 * 10.01 = 7.007 -> 7.01; 0.30 * 10.01 = 3.003 -> 3.00; sum 10.01.
	if _, err := l.Deposit(dec("10.01"), "USD", "ref", split); err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if !l.Total().Equal(dec("10.01")) {
		t.Fatalf("Total = %s, want 10.01", l.Total())
	}
	sum := l.Balance("a").Add(l.Balance("b"))
	if !sum.Equal(dec("10.01")) {
		t.Fatalf("a+b = %s, want 10.01", sum)
	}
}

func TestLedgerDepositForeignCurrency(t *testing.T) {
	l := NewLedger(DefaultExchangeRates())
	split := []SplitRule{
		SplitPercent(PlatformAccount, 15),
		SplitPercent("vendor", 85),
	}
	// 92 EUR = 100 USD; platform 15 USD, vendor 85 USD.
	if _, err := l.Deposit(dec("92"), "EUR", "ref", split); err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if !l.Balance(PlatformAccount).Equal(dec("15")) {
		t.Fatalf("platform = %s, want 15", l.Balance(PlatformAccount))
	}
	if !l.Balance("vendor").Equal(dec("85")) {
		t.Fatalf("vendor = %s, want 85", l.Balance("vendor"))
	}
	if e := l.Entries(); len(e) != 2 || e[0].SourceCurrency != "EUR" || !e[0].SourceAmount.Equal(dec("92")) {
		t.Fatalf("entry source metadata not preserved: %+v", e)
	}
}

func TestLedgerDepositValidation(t *testing.T) {
	l := NewLedger(DefaultExchangeRates())
	full := []SplitRule{SplitPercent("a", 100)}

	if _, err := l.Deposit(dec("0"), "USD", "r", full); err != ErrNonPositive {
		t.Fatalf("zero amount = %v, want ErrNonPositive", err)
	}
	if _, err := l.Deposit(dec("-1"), "USD", "r", full); err != ErrNonPositive {
		t.Fatalf("negative amount = %v, want ErrNonPositive", err)
	}
	if _, err := l.Deposit(dec("10"), "USD", "r", nil); err != ErrEmptySplit {
		t.Fatalf("nil split = %v, want ErrEmptySplit", err)
	}
	bad := []SplitRule{SplitPercent("a", 70), SplitPercent("b", 20)}
	if _, err := l.Deposit(dec("10"), "USD", "r", bad); err != ErrSplitNotWhole {
		t.Fatalf("split sum 0.9 = %v, want ErrSplitNotWhole", err)
	}
	if _, err := l.Deposit(dec("10"), "XYZ", "r", full); err != ErrUnknownCurrency {
		t.Fatalf("unknown currency = %v, want ErrUnknownCurrency", err)
	}
}

func TestVendorAccount(t *testing.T) {
	id := uuid.New()
	if got := VendorAccount(id); got != "vendor:"+id.String() {
		t.Fatalf("VendorAccount = %s", got)
	}
}
