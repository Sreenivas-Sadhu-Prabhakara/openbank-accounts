# Changelog

All notable changes to **openbank-accounts** are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-06-24

Initial release of the Accounts microservice (OBIE AISP / BIAN Customer Position).

### Added

- AISP read endpoints: `/accounts`, `/balances`, `/transactions` with consent and permission enforcement.
- Internal funds-confirmation endpoint — the single source of truth for balances.
- In-memory and Postgres repository adapters; SQL migrations applied on startup.
- OBIE-shaped HTTP API with FAPI interaction-id, structured logging and panic recovery.
- Unit/handler test suite plus Postgres integration tests (testcontainers).
- GitHub Actions CI and MIT license.
