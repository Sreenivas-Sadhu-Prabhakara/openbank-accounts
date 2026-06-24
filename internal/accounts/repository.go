package accounts

import (
	"context"
	"errors"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// ErrNotFound is returned by a Repository when no account matches the lookup.
var ErrNotFound = errors.New("account not found")

// Repository is the persistence port for the account read model. Both the
// in-memory and the Postgres implementations satisfy it, and the service layer
// depends only on this interface — so the same business-logic tests run against
// either store.
type Repository interface {
	// ListAccounts returns every account in the estate.
	ListAccounts(ctx context.Context) ([]Account, error)
	// GetAccount returns a single account by its AccountId, or ErrNotFound.
	GetAccount(ctx context.Context, accountID string) (*Account, error)

	// ListBalances returns the balances for one account. Returns ErrNotFound
	// if the account does not exist.
	ListBalances(ctx context.Context, accountID string) ([]Balance, error)
	// ListTransactions returns the transactions for one account. Returns
	// ErrNotFound if the account does not exist.
	ListTransactions(ctx context.Context, accountID string) ([]Transaction, error)

	// ListAllBalances returns the balances across every account (the bulk
	// /balances endpoint).
	ListAllBalances(ctx context.Context) ([]Balance, error)
	// ListAllTransactions returns the transactions across every account (the
	// bulk /transactions endpoint).
	ListAllTransactions(ctx context.Context) ([]Transaction, error)

	// FindByIdentification resolves an account by its OBIE account
	// Identification (e.g. sort-code+account-number), used by the internal
	// funds-confirmation lookup. Returns ErrNotFound if unknown.
	FindByIdentification(ctx context.Context, identification string) (*Account, error)

	// AvailableBalance returns the account's InterimAvailable balance. Returns
	// ErrNotFound if the account does not exist.
	AvailableBalance(ctx context.Context, accountID string) (obie.Amount, error)
}
