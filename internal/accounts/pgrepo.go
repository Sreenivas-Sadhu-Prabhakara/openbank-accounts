package accounts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// PgRepository is the Postgres-backed Repository. The accounts service owns the
// "accounts" schema; this type touches nothing outside it.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns a Postgres repository over the given pool.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

const accountColumns = `account_id, status, currency, account_type, account_subtype,
	nickname, scheme_name, identification, name`

const balanceColumns = `account_id, type, credit_debit, amount, currency, dt`

const transactionColumns = `transaction_id, account_id, credit_debit, status,
	amount, currency, booking_dt, information`

func (r *PgRepository) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+accountColumns+` FROM accounts.accounts ORDER BY account_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *PgRepository) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+accountColumns+` FROM accounts.accounts WHERE account_id = $1`, accountID)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *PgRepository) ListBalances(ctx context.Context, accountID string) ([]Balance, error) {
	if err := r.assertAccountExists(ctx, accountID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+balanceColumns+` FROM accounts.balances WHERE account_id = $1 ORDER BY type`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *PgRepository) ListTransactions(ctx context.Context, accountID string) ([]Transaction, error) {
	if err := r.assertAccountExists(ctx, accountID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+transactionColumns+` FROM accounts.transactions WHERE account_id = $1 ORDER BY transaction_id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *PgRepository) ListAllBalances(ctx context.Context) ([]Balance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+balanceColumns+` FROM accounts.balances ORDER BY account_id, type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *PgRepository) ListAllTransactions(ctx context.Context) ([]Transaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+transactionColumns+` FROM accounts.transactions ORDER BY account_id, transaction_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *PgRepository) FindByIdentification(ctx context.Context, identification string) (*Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+accountColumns+` FROM accounts.accounts WHERE identification = $1`, identification)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *PgRepository) AvailableBalance(ctx context.Context, accountID string) (obie.Amount, error) {
	balances, err := r.ListBalances(ctx, accountID)
	if err != nil {
		return obie.Amount{}, err
	}
	amt, _ := availableBalance(balances)
	return amt, nil
}

// assertAccountExists returns ErrNotFound when no account row matches, so the
// per-account balance/transaction queries 404 for an unknown account rather
// than silently returning an empty slice.
func (r *PgRepository) assertAccountExists(ctx context.Context, accountID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM accounts.accounts WHERE account_id = $1)`, accountID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

// scanAccount reads a row in accountColumns order into an Account.
func scanAccount(row pgx.Row) (*Account, error) {
	var a Account
	if err := row.Scan(
		&a.AccountID, &a.Status, &a.Currency, &a.AccountType, &a.AccountSubType,
		&a.Nickname, &a.SchemeName, &a.Identification, &a.Name,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

// scanBalances reads balance rows in balanceColumns order.
func scanBalances(rows pgx.Rows) ([]Balance, error) {
	var out []Balance
	for rows.Next() {
		var (
			b                Balance
			amount, currency string
		)
		if err := rows.Scan(
			&b.AccountID, &b.Type, &b.CreditDebitIndicator, &amount, &currency, &b.DateTime,
		); err != nil {
			return nil, err
		}
		amt, err := obie.NewAmount(amount, currency)
		if err != nil {
			return nil, err
		}
		b.Amount = amt
		out = append(out, b)
	}
	return out, rows.Err()
}

// scanTransactions reads transaction rows in transactionColumns order.
func scanTransactions(rows pgx.Rows) ([]Transaction, error) {
	var out []Transaction
	for rows.Next() {
		var (
			t                Transaction
			amount, currency string
		)
		if err := rows.Scan(
			&t.TransactionID, &t.AccountID, &t.CreditDebitIndicator, &t.Status,
			&amount, &currency, &t.BookingDateTime, &t.TransactionInformation,
		); err != nil {
			return nil, err
		}
		amt, err := obie.NewAmount(amount, currency)
		if err != nil {
			return nil, err
		}
		t.Amount = amt
		out = append(out, t)
	}
	return out, rows.Err()
}
