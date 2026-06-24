package accounts

import (
	"testing"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

func TestAvailableBalance(t *testing.T) {
	cases := []struct {
		name     string
		balances []Balance
		want     string // expected available amount; "" means not found
		found    bool
	}{
		{
			name: "credit interim-available is taken as-is",
			balances: []Balance{{
				Type:                 BalanceTypeInterimAvailable,
				CreditDebitIndicator: IndicatorCredit,
				Amount:               obie.MustAmount("1230.00", "GBP"),
			}},
			want:  "1230",
			found: true,
		},
		{
			name: "debit interim-available is negated (overdrawn)",
			balances: []Balance{{
				Type:                 BalanceTypeInterimAvailable,
				CreditDebitIndicator: IndicatorDebit,
				Amount:               obie.MustAmount("45.00", "GBP"),
			}},
			want:  "-45",
			found: true,
		},
		{
			name: "ignores non-available balance types",
			balances: []Balance{
				{Type: "ClosingBooked", CreditDebitIndicator: IndicatorCredit, Amount: obie.MustAmount("99", "GBP")},
				{Type: BalanceTypeInterimAvailable, CreditDebitIndicator: IndicatorCredit, Amount: obie.MustAmount("1230.00", "GBP")},
			},
			want:  "1230",
			found: true,
		},
		{
			name:     "no available balance returns zero, not found",
			balances: nil,
			want:     "",
			found:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := availableBalance(tc.balances)
			if found != tc.found {
				t.Fatalf("found = %v, want %v", found, tc.found)
			}
			if found && got.String() != tc.want {
				t.Fatalf("available = %s, want %s", got.String(), tc.want)
			}
		})
	}
}
