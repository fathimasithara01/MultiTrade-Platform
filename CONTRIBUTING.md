# Contributing to MultiTrade Platform

Thank you for your interest in contributing. This document covers the development workflow, coding standards, and review process.

---

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- `golangci-lint` ([install](https://golangci-lint.run/usage/install/))
- `swag` CLI (installed automatically via `make swagger`)

### Setup

```bash
git clone https://github.com/fathimasithara01/multitrade-platform
cd MultiTrade-Platform
cp .env.example .env          # configure local environment
make docker-up                # start Postgres, Redis, Kafka
make run                      # start the API server
```

---

## Development Workflow

1. Create a branch from `main`: `git checkout -b feat/your-feature`
2. Make your changes, following the standards below
3. Run `make fmt vet lint` before committing
4. Run `make test` to ensure all tests pass
5. Open a pull request against `main`

---

## Coding Standards

### Package layout

```
internal/
  <domain>/          # one package per business domain
    service.go       # business logic (the only place that knows "why")
    service_test.go  # unit tests with stub dependencies
  handlers/          # HTTP layer — thin, only translates HTTP ↔ service calls
  repository/        # persistence — one file per domain
  models/            # plain structs, no behaviour, no imports of internal packages
  middleware/        # Gin middleware (auth, RBAC, rate limit, metrics, request ID)
pkg/                 # shared libraries safe to import from anywhere
```

### Rules

- **No business logic in handlers.** Handlers translate HTTP; services own logic.
- **No direct DB access outside repository.** Services call repositories, not `db.Query`.
- **Interface-driven.** Every repository and service consumed by another package must be an interface.
- **Errors.** Return typed sentinel errors (`ErrUserNotFound`, `ErrInsufficientBalance`) — never raw strings. Use `errors.Is` / `errors.As` in callers.
- **Decimal arithmetic.** Use `pkg/decimal` — never `float64` for money.
- **Context.** Every function that touches I/O must accept `context.Context` as the first argument.
- **Tests.** Unit tests use stub/mock dependencies. Integration tests connect to a real DB. New features require at least one test of each type.

### Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(wallet): add idempotent deposit endpoint
fix(matching): prevent self-trade on concurrent orders
test(auth): add suspended-user login test case
docs: update API overview table in README
chore: upgrade Go to 1.24
```

---

## Pull Request Checklist

- [ ] `make fmt vet lint` passes with zero warnings
- [ ] `make test` passes (all packages)
- [ ] New behaviour is covered by at least one test
- [ ] Public functions have doc comments
- [ ] No secrets or `.env` files committed
- [ ] Swagger docs regenerated if handler signatures changed (`make swagger`)
- [ ] PR description explains *what* changed and *why*

---

## Running the Test Suite

```bash
make test-unit          # auth package — no DB required (fast, < 5s)
make test-integration   # wallet + matching engine — requires Docker stack
make test-e2e           # full flow — requires running server on :8080
```

---

## Code Review

All PRs require at least one approval before merging. Reviewers look for:

- Correct use of transactions and row-level locking for financial mutations
- No negative-balance scenarios (wallet `CHECK` + service-level guard)
- Proper error wrapping (`fmt.Errorf("context: %w", err)`)
- No goroutine leaks (context cancellation, proper `defer` cleanup)
