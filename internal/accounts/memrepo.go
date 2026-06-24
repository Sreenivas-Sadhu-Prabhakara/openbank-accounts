package accounts

import (
	"context"
	"sync"
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// MemRepository is an in-memory Repository used by unit and handler tests and
// for running the service without a database. It is safe for concurrent use.
// Unlike the consent store it is read-only after construction: the account
// read model is fixed seed data, so there are no mutating methods.
type MemRepository struct {
	mu           sync.RWMutex
	accounts     map[string]Account       // keyed by AccountId
	balances     map[string][]Balance     // keyed by AccountId
	transactions map[string][]Transaction // keyed by AccountId
	order        []string                 // AccountIds in deterministic insertion order
}

// NewMemRepository returns an in-memory repository pre-loaded with the demo
// data the cross-service walkthrough relies on (two accounts for the PSU
// "Kelvin Smith"). The same data is seeded into Postgres by the migration SQL,
// so both backends behave identically.
func NewMemRepository() *MemRepository {
	r := &MemRepository{
		accounts:     make(map[string]Account),
		balances:     make(map[string][]Balance),
		transactions: make(map[string][]Transaction),
	}
	for _, s := range seedAccounts() {
		r.accounts[s.account.AccountID] = s.account
		r.balances[s.account.AccountID] = s.balances
		r.transactions[s.account.AccountID] = s.transactions
		r.order = append(r.order, s.account.AccountID)
	}
	return r
}

// seedRow bundles an account with its balances and transactions for seeding.
type seedRow struct {
	account      Account
	balances     []Balance
	transactions []Transaction
}

// seedTime is a fixed clock for the demo data so responses are stable.
var seedTime = time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

// seedAccounts returns the canonical demo dataset shared with the migration
// SQL. Account A (22289) is a current account with two booked transactions;
// account B (31820) is a savings account with one.
func seedAccounts() []seedRow {
	return []seedRow{
		{
			account: Account{
				AccountID:      "22289",
				Status:         "Enabled",
				Currency:       "GBP",
				AccountType:    "Personal",
				AccountSubType: "CurrentAccount",
				Nickname:       "Bills",
				SchemeName:     "UK.OBIE.SortCodeAccountNumber",
				Identification: "80200110203345",
				Name:           "Mr Kelvin Smith",
			},
			balances: []Balance{{
				AccountID:            "22289",
				Type:                 BalanceTypeInterimAvailable,
				CreditDebitIndicator: IndicatorCredit,
				Amount:               obie.MustAmount("1230.00", "GBP"),
				DateTime:             seedTime,
			}},
			transactions: []Transaction{
				{
					TransactionID:          "22289-001",
					AccountID:              "22289",
					CreditDebitIndicator:   IndicatorDebit,
					Status:                 "Booked",
					Amount:                 obie.MustAmount("12.50", "GBP"),
					BookingDateTime:        seedTime,
					TransactionInformation: "Payment to ACME",
				},
				{
					TransactionID:          "22289-002",
					AccountID:              "22289",
					CreditDebitIndicator:   IndicatorCredit,
					Status:                 "Booked",
					Amount:                 obie.MustAmount("500.00", "GBP"),
					BookingDateTime:        seedTime,
					TransactionInformation: "Salary",
				},
			},
		},
		{
			account: Account{
				AccountID:      "31820",
				Status:         "Enabled",
				Currency:       "GBP",
				AccountType:    "Personal",
				AccountSubType: "Savings",
				Nickname:       "Rainy Day",
				SchemeName:     "UK.OBIE.SortCodeAccountNumber",
				Identification: "80200110203348",
				Name:           "Kelvin Smith Savings",
			},
			balances: []Balance{{
				AccountID:            "31820",
				Type:                 BalanceTypeInterimAvailable,
				CreditDebitIndicator: IndicatorCredit,
				Amount:               obie.MustAmount("5000.00", "GBP"),
				DateTime:             seedTime,
			}},
			transactions: []Transaction{
				{
					TransactionID:          "31820-001",
					AccountID:              "31820",
					CreditDebitIndicator:   IndicatorCredit,
					Status:                 "Booked",
					Amount:                 obie.MustAmount("250.00", "GBP"),
					BookingDateTime:        seedTime,
					TransactionInformation: "Transfer from current account",
				},
			},
		},
	}
}

func (r *MemRepository) ListAccounts(_ context.Context) ([]Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Account, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.accounts[id])
	}
	return out, nil
}

func (r *MemRepository) GetAccount(_ context.Context, accountID string) (*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.accounts[accountID]
	if !ok {
		return nil, ErrNotFound
	}
	out := a
	return &out, nil
}

func (r *MemRepository) ListBalances(_ context.Context, accountID string) ([]Balance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.accounts[accountID]; !ok {
		return nil, ErrNotFound
	}
	return append([]Balance(nil), r.balances[accountID]...), nil
}

func (r *MemRepository) ListTransactions(_ context.Context, accountID string) ([]Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.accounts[accountID]; !ok {
		return nil, ErrNotFound
	}
	return append([]Transaction(nil), r.transactions[accountID]...), nil
}

func (r *MemRepository) ListAllBalances(_ context.Context) ([]Balance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Balance
	for _, id := range r.order {
		out = append(out, r.balances[id]...)
	}
	return out, nil
}

func (r *MemRepository) ListAllTransactions(_ context.Context) ([]Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Transaction
	for _, id := range r.order {
		out = append(out, r.transactions[id]...)
	}
	return out, nil
}

func (r *MemRepository) FindByIdentification(_ context.Context, identification string) (*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, id := range r.order {
		a := r.accounts[id]
		if a.Identification == identification {
			out := a
			return &out, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemRepository) AvailableBalance(_ context.Context, accountID string) (obie.Amount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.accounts[accountID]; !ok {
		return obie.Amount{}, ErrNotFound
	}
	amt, _ := availableBalance(r.balances[accountID])
	return amt, nil
}
