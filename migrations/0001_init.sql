-- Accounts (AIS) service schema. Owned exclusively by the accounts
-- microservice; no other service reads or writes these tables.
CREATE SCHEMA IF NOT EXISTS accounts;

CREATE TABLE IF NOT EXISTS accounts.accounts (
    account_id      TEXT PRIMARY KEY,
    status          TEXT NOT NULL,
    currency        TEXT NOT NULL,
    account_type    TEXT NOT NULL,
    account_subtype TEXT NOT NULL,
    nickname        TEXT NOT NULL DEFAULT '',
    -- OBIE account identifier block
    scheme_name     TEXT NOT NULL,
    identification  TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT ''
);

-- The internal funds-confirmation lookup resolves an account by its OBIE
-- Identification, so index it.
CREATE INDEX IF NOT EXISTS idx_accounts_identification ON accounts.accounts (identification);

CREATE TABLE IF NOT EXISTS accounts.balances (
    account_id   TEXT NOT NULL REFERENCES accounts.accounts (account_id),
    type         TEXT NOT NULL,
    credit_debit TEXT NOT NULL,
    amount       TEXT NOT NULL,
    currency     TEXT NOT NULL,
    dt           TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (account_id, type)
);

CREATE TABLE IF NOT EXISTS accounts.transactions (
    transaction_id TEXT PRIMARY KEY,
    account_id     TEXT NOT NULL REFERENCES accounts.accounts (account_id),
    credit_debit   TEXT NOT NULL,
    status         TEXT NOT NULL,
    amount         TEXT NOT NULL,
    currency       TEXT NOT NULL,
    booking_dt     TIMESTAMPTZ NOT NULL,
    information    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_transactions_account ON accounts.transactions (account_id);

-- Demo seed data: the PSU "Kelvin Smith" with a current account (22289) and a
-- savings account (31820). Mirrors NewMemRepository so both backends behave
-- identically. ON CONFLICT DO NOTHING keeps re-runs idempotent.
INSERT INTO accounts.accounts
    (account_id, status, currency, account_type, account_subtype, nickname, scheme_name, identification, name)
VALUES
    ('22289', 'Enabled', 'GBP', 'Personal', 'CurrentAccount', 'Bills',
        'UK.OBIE.SortCodeAccountNumber', '80200110203345', 'Mr Kelvin Smith'),
    ('31820', 'Enabled', 'GBP', 'Personal', 'Savings', 'Rainy Day',
        'UK.OBIE.SortCodeAccountNumber', '80200110203348', 'Kelvin Smith Savings')
ON CONFLICT (account_id) DO NOTHING;

INSERT INTO accounts.balances
    (account_id, type, credit_debit, amount, currency, dt)
VALUES
    ('22289', 'InterimAvailable', 'Credit', '1230.00', 'GBP', '2026-06-01T09:00:00Z'),
    ('31820', 'InterimAvailable', 'Credit', '5000.00', 'GBP', '2026-06-01T09:00:00Z')
ON CONFLICT (account_id, type) DO NOTHING;

INSERT INTO accounts.transactions
    (transaction_id, account_id, credit_debit, status, amount, currency, booking_dt, information)
VALUES
    ('22289-001', '22289', 'Debit',  'Booked', '12.50',  'GBP', '2026-06-01T09:00:00Z', 'Payment to ACME'),
    ('22289-002', '22289', 'Credit', 'Booked', '500.00', 'GBP', '2026-06-01T09:00:00Z', 'Salary'),
    ('31820-001', '31820', 'Credit', 'Booked', '250.00', 'GBP', '2026-06-01T09:00:00Z', 'Transfer from current account')
ON CONFLICT (transaction_id) DO NOTHING;
