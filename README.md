# MultiTrade Platform

A production-grade, multi-role trading platform backend built in Go. Features a real-time limit-order matching engine, wallet management with strict consistency guarantees, role-based access control, Redis caching, Kafka event streaming, WebSocket price feeds, Prometheus metrics, and a full admin API.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP Clients / WebSocket                    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Gin HTTP Server    │  :8080
                    │  Auth · Rate Limit   │
                    │  Prometheus Metrics  │
                    │  Swagger UI          │
                    └──────────┬──────────┘
          ┌────────────────────┼─────────────────────┐
          │                    │                     │
   ┌──────▼──────┐   ┌─────────▼──────┐   ┌─────────▼──────┐
   │   Handlers  │   │   WebSocket Hub │   │   /metrics     │
   │  auth/wallet│   │  Real-time feed │   │  Prometheus    │
   │  asset/order│   └─────────────────┘   └────────────────┘
   │  admin      │
   └──────┬──────┘
          │  calls
   ┌──────▼──────────────────────────────────┐
   │              Services Layer             │
   │  AuthService  WalletService  AssetSvc   │
   │  OrderService AdminService              │
   └──────┬──────────────────────────────────┘
          │  uses interfaces
   ┌──────▼──────────────────────────────────┐
   │           Repository Layer              │
   │  UserRepo  WalletRepo  OrderRepo        │
   │  AssetRepo TradeRepo   PortfolioRepo    │
   │  AuditRepo                              │
   └──────┬──────────────────────────────────┘
          │
   ┌──────▼──────────────────────────────────┐
   │           Infrastructure                │
   │  PostgreSQL  Redis  Kafka  WebSocket    │
   └─────────────────────────────────────────┘

   ┌──────────────────────────────────────────┐
   │         Matching Engine (goroutine)      │
   │  Channel-based · Price-Time Priority     │
   │  Atomic DB transactions · FOR UPDATE     │
   │  Publishes to Kafka + Redis Pub/Sub      │
   └──────────────────────────────────────────┘
```

See [`docs/TradeVerse Architecture Diagram.png`](docs/TradeVerse%20Architecture%20Diagram.png) for the full system diagram.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.24 |
| HTTP Framework | Gin |
| Database | PostgreSQL (sqlx + raw SQL migrations) |
| Cache | Redis (go-redis/v9) |
| Message Queue | Kafka (segmentio/kafka-go) |
| Auth | JWT (golang-jwt/v5) + bcrypt |
| Real-time | WebSocket (gorilla/websocket) |
| Metrics | Prometheus (prometheus/client_golang) |
| API Docs | Swagger (swaggo/swag) |
| Logging | Zerolog (structured JSON) |
| Testing | Testify + integration tests against real Postgres |

---

## Project Structure

```
MultiTrade-Platform/
├── cmd/api/
│   └── main.go              # Entry point — wires all dependencies, starts server
├── internal/
│   ├── admin/               # Admin service: user management, analytics, health
│   ├── auth/                # JWT service, auth service, unit tests
│   ├── cache/               # Redis client wrapper (price cache, pub/sub, rate limit)
│   ├── config/              # Viper-based config loader
│   ├── cronjobs/            # Scheduled maintenance (expire orders, reconcile wallets)
│   ├── database/            # sqlx connection + custom migration runner
│   ├── docs/                # Swagger generated spec (swag init output)
│   ├── handlers/            # Gin HTTP handlers (auth, wallet, asset, order, admin)
│   ├── kafka/               # Producer, consumers (notification, audit, analytics)
│   ├── matchingengine/      # Price-time priority matching engine + tests
│   ├── middleware/          # JWT auth, RBAC, rate limiting, Prometheus
│   ├── models/              # Domain structs (User, Wallet, Order, Trade, Asset...)
│   ├── repository/          # Postgres implementations of all domain repositories
│   ├── trading/             # Asset service + Order service
│   ├── wallet/              # Wallet service + integration tests
│   └── websocket/           # Hub, client, upgrade handler
├── migrations/              # Sequential SQL migrations (000001_*.up.sql, ...)
├── scripts/
│   └── smoke_e2e.go         # Full e2e smoke test (go run scripts/smoke_e2e.go)
├── postman/                 # Postman collection + environment
├── deployments/             # Docker/K8s configs (add as needed)
├── docs/                    # Architecture diagrams
├── .env                     # Local environment variables (never commit secrets)
├── docker-compose.yml       # Postgres + Redis + Kafka local dev stack
└── go.mod
```

---

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose (for Postgres, Redis, Kafka)

### 1. Start infrastructure

```bash
docker-compose up -d
```

This starts:
- PostgreSQL on port `5433`
- Redis on port `6379`
- Kafka (Redpanda) on port `9092`

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env if needed — defaults work with docker-compose out of the box
```

### 3. Run the server

```bash
go run ./cmd/api
```

Migrations run automatically on startup. The server starts on **http://localhost:8080**.

### 4. Explore the API

| URL | Description |
|---|---|
| http://localhost:8080/swagger/index.html | Interactive Swagger UI |
| http://localhost:8080/metrics | Prometheus metrics scrape endpoint |
| http://localhost:8080/health | Service health check |
| ws://localhost:8080/ws | WebSocket real-time price feed |

---

## API Overview

### Authentication

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/auth/register` | Public | Register (role: admin/broker/trader/support) |
| POST | `/auth/login` | Public (rate-limited: 10/min/IP) | Login, returns JWT pair |
| POST | `/auth/refresh` | Public | Exchange refresh token for new access token |
| GET | `/auth/me` | Bearer | Current user info |

### Wallet (rate-limited: 30/min/user)

| Method | Path | Description |
|---|---|---|
| GET | `/wallet` | Current balance |
| POST | `/wallet/deposit` | Deposit funds |
| POST | `/wallet/withdraw` | Withdraw funds (rejects if insufficient) |
| GET | `/wallet/transactions` | Paginated transaction history |

### Assets

| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/assets` | Any | List active assets (Redis-cached prices) |
| GET | `/assets/:id` | Any | Single asset detail |
| POST | `/assets` | BROKER | Create asset |
| PATCH | `/assets/:id` | BROKER (owner) | Update price/quantity |
| PATCH | `/assets/:id/disable` | BROKER (owner) | Disable asset |

### Orders (TRADER only, rate-limited: 60/min/user)

| Method | Path | Description |
|---|---|---|
| POST | `/orders` | Place limit order (BUY/SELL) with optional `Idempotency-Key` header |
| GET | `/orders` | Own order history (paginated) |
| GET | `/orders/:id` | Single order detail |
| DELETE | `/orders/:id` | Cancel PENDING/PARTIALLY_FILLED order |

### Admin (ADMIN only)

| Method | Path | Description |
|---|---|---|
| GET | `/admin/users` | List users (filter by role/status) |
| PATCH | `/admin/users/:id/status` | Suspend or activate a user |
| GET | `/admin/analytics/volume` | Hourly trade volume (last N hours) |
| GET | `/admin/analytics/suspicious` | Users exceeding trade frequency threshold |
| GET | `/admin/health` | DB + Redis + Kafka connectivity |
| GET | `/admin/audit` | Paginated audit logs |

---

## Key Design Decisions

**Matching Engine**
The engine runs as a single goroutine consuming from a buffered `chan *models.Order`. Order processing is serialised through the channel; within each match, both order rows are locked with `SELECT … FOR UPDATE` in ascending ID order to prevent deadlocks. The entire trade — order fills, wallet debits/credits, portfolio updates, trade record — commits in one DB transaction.

**Wallet Consistency**
Every balance mutation uses `SELECT … FOR UPDATE` inside a transaction and bumps a `version` column. The `CHECK (balance >= 0)` constraint on the DB acts as a final safety net against race conditions.

**Idempotency**
Order placement accepts an `Idempotency-Key` header. The key is stored with a partial unique index `(user_id, idempotency_key)`. Duplicate requests return the original order without creating a second row.

**Rate Limiting**
Redis-backed sliding window rate limiter (`ZADD` + `ZCARD` pipeline). Fails open if Redis is unavailable. Returns `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` headers.

**Observability**
- Structured JSON logs via Zerolog
- Prometheus metrics at `/metrics`: request count/latency, trades executed, matching engine duration, WebSocket connections, wallet transaction counts, rate limit hits
- Audit logs in Postgres for every trade, order, and admin action

---

## Running Tests

```bash
# All tests (unit + integration, requires local Postgres)
go test ./...

# Specific packages
go test -v ./internal/auth/...
go test -v ./internal/wallet/...
go test -v ./internal/matchingengine/...

# Full e2e smoke test (server must be running on :8080)
go run scripts/smoke_e2e.go
```

**Test coverage:**

| Package | Tests | Type |
|---|---|---|
| `internal/auth` | 9 tests | Unit (stub repo, no DB) |
| `internal/wallet` | 8 tests | Integration (real Postgres) |
| `internal/matchingengine` | 2 tests | Integration — full + partial fill, 25 concurrent orders |
| `scripts/smoke_e2e.go` | 14 checks | E2E against live server |

---

## Roles

| Role | Permissions |
|---|---|
| `admin` | Full admin API access, user management, analytics |
| `broker` | Create/manage assets |
| `trader` | Place/cancel orders, wallet operations |
| `support` | Read-only access (extend as needed) |

---

## Postman Collection

Import [`postman/TradeVerse.postman_collection.json`](postman/TradeVerse.postman_collection.json) and [`postman/TradeVerse_Environment.json`](postman/TradeVerse_Environment.json) into Postman. Set `base_url` to `http://localhost:8080` and use the Register/Login requests to populate the `access_token` variable automatically.

---

## License

MIT — see [LICENSE](LICENSE).
