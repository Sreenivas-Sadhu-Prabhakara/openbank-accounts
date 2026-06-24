# openbank-accounts

[![CI](https://github.com/Sreenivas-Sadhu-Prabhakara/openbank-accounts/actions/workflows/ci.yml/badge.svg)](https://github.com/Sreenivas-Sadhu-Prabhakara/openbank-accounts/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)

The **Accounts (AIS)** microservice — the BIAN *Customer Position / Current Account* service domain, exposing the OBIE **AISP** read APIs: accounts, balances and transactions.

Every request is consent-protected. The caller supplies the consent id via the `x-consent-id` header; the service validates it against the consent service (type `account-access`, status `Authorised`, and the right `Read*` permission) before returning data.

## Endpoints

| Method | Path | Permission |
|---|---|---|
| GET | `/accounts` | `ReadAccountsBasic`/`ReadAccountsDetail` |
| GET | `/accounts/{accountId}` | `ReadAccountsBasic`/`ReadAccountsDetail` |
| GET | `/accounts/{accountId}/balances` | `ReadBalances` |
| GET | `/accounts/{accountId}/transactions` | `ReadTransactions*` |
| GET | `/balances` | `ReadBalances` |
| GET | `/transactions` | `ReadTransactions*` |
| GET | `/internal/funds-confirmation?identification=&amount=&currency=` | _(none — service-to-service)_ |
| GET | `/health` | — |

The internal funds-confirmation endpoint is the single source of truth used by the CBPII (funds) service.

## Configuration

| Env | Default | Notes |
|---|---|---|
| `ADDR` | `:8082` | Listen address |
| `BASE_URL` | `http://localhost:8082` | Used for `Links.Self` |
| `DATABASE_URL` | _(unset)_ | Postgres DSN; **unset → in-memory store** (seeded demo data) |
| `CONSENT_URL` | `http://localhost:8081` | Consent service base URL |

Demo data: accounts `22289` (`80200110203345`, £1230.00 available) and `31820` (`80200110203348`, £5000.00).

## Run

```bash
go run .                              # in-memory + demo data
docker build -t openbank/accounts . && docker run -p 8082:8082 openbank/accounts
# AIS calls require a valid, authorised account-access consent id:
curl localhost:8082/accounts -H "x-consent-id: <consentId>"
```

## Test

```bash
go test ./...                       # unit + handler tests (fake consent client, no Docker)
go test -tags=integration ./...     # Postgres repo tests via testcontainers (needs Docker)
```

## Layout notes

- `internal/accounts/` — domain, `Repository` port (in-memory + Postgres), service (consent enforcement), OBIE handlers.
- `migrations/` — SQL owned by this service, applied on startup when `DATABASE_URL` is set.
- `pkg/` — vendored shared OBIE library, wired via `replace ... => ./pkg`.
- Simplification: an authorised account-access consent grants read access to all seeded accounts (PSU↔account selection at authorisation is not modelled).
