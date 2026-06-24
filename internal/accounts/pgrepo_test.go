//go:build integration

package accounts

import (
	"context"
	"os"
	"testing"

	"github.com/sreeni/openbank-bian/pkg/pg"
	"github.com/sreeni/openbank-bian/pkg/testutil"
)

// newPgRepo spins up a throwaway Postgres, applies migrations (which also seed
// the demo data) and returns a Postgres-backed repository. Migrations are read
// from the module's migrations directory relative to this test package.
func newPgRepo(t *testing.T) *PgRepository {
	t.Helper()
	ctx := context.Background()
	dsn := testutil.PostgresDSN(t)

	pool, err := pg.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pg.RunMigrations(ctx, pool, os.DirFS("../.."), "migrations", "accounts"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewPgRepository(pool)
}

func TestPgRepositorySeedRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newPgRepo(t)

	// The migration seeds two accounts.
	accs, err := repo.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accs) != 2 {
		t.Fatalf("got %d accounts, want 2", len(accs))
	}

	a, err := repo.GetAccount(ctx, "22289")
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if a.AccountSubType != "CurrentAccount" || a.Identification != "80200110203345" {
		t.Fatalf("unexpected account %+v", a)
	}

	balances, err := repo.ListBalances(ctx, "22289")
	if err != nil {
		t.Fatalf("balances: %v", err)
	}
	if len(balances) != 1 || balances[0].Amount.String() != "1230" {
		t.Fatalf("unexpected balances %+v", balances)
	}

	txns, err := repo.ListTransactions(ctx, "22289")
	if err != nil {
		t.Fatalf("transactions: %v", err)
	}
	if len(txns) != 2 {
		t.Fatalf("got %d transactions, want 2", len(txns))
	}

	allTxns, err := repo.ListAllTransactions(ctx)
	if err != nil {
		t.Fatalf("all transactions: %v", err)
	}
	if len(allTxns) != 3 {
		t.Fatalf("got %d total transactions, want 3", len(allTxns))
	}
}

func TestPgRepositoryFindByIdentificationAndAvailableBalance(t *testing.T) {
	ctx := context.Background()
	repo := newPgRepo(t)

	a, err := repo.FindByIdentification(ctx, "80200110203348")
	if err != nil {
		t.Fatalf("find by identification: %v", err)
	}
	if a.AccountID != "31820" || a.AccountSubType != "Savings" {
		t.Fatalf("unexpected account %+v", a)
	}

	avail, err := repo.AvailableBalance(ctx, "31820")
	if err != nil {
		t.Fatalf("available balance: %v", err)
	}
	if avail.String() != "5000" || avail.Currency != "GBP" {
		t.Fatalf("available = %s %s, want 5000 GBP", avail.String(), avail.Currency)
	}
}

func TestPgRepositoryNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newPgRepo(t)

	if _, err := repo.GetAccount(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("GetAccount err = %v, want ErrNotFound", err)
	}
	if _, err := repo.FindByIdentification(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("FindByIdentification err = %v, want ErrNotFound", err)
	}
	if _, err := repo.ListBalances(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("ListBalances err = %v, want ErrNotFound", err)
	}
	if _, err := repo.AvailableBalance(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("AvailableBalance err = %v, want ErrNotFound", err)
	}
}
