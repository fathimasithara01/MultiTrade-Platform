<div align="center">

# 🏦 MultiTrade Platform

### Production-Grade Multi-Role Trading Backend in Go

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=flat-square&logo=postgresql&logoColor=white)]()
[![Redis](https://img.shields.io/badge/Redis-DC382D?style=flat-square&logo=redis&logoColor=white)]()
[![Kafka](https://img.shields.io/badge/Kafka-231F20?style=flat-square&logo=apachekafka&logoColor=white)]()
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white)]()
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)]()
[![Go Report](https://img.shields.io/badge/Go%20Report-A+-brightgreen?style=flat-square)]()

> A complete trading platform backend built to production standards — real-time order matching, ACID wallet system, event-driven architecture, and full observability.

</div>

---

## ✨ Highlights

- ⚡ **Price-Time Priority Order Matching Engine** — goroutine-safe, channel-based, deadlock-free
- 💰 **ACID Wallet System** — row-level locking, version column, DB-level balance constraint
- 📡 **Real-Time WebSocket Feeds** — live order book updates with topic-based subscriptions
- 🔐 **4-Role RBAC** — Admin, Broker, Trader, Support with JWT access + refresh tokens
- 📊 **Prometheus Observability** — request latency, trade count, WebSocket connections, rate limit hits
- 📨 **Kafka Event Streaming** — async trade notifications, audit trails, analytics consumers
- 🔑 **Idempotency** — duplicate order protection via partial unique index
- 🧪 **Full Test Coverage** — unit, integration (real Postgres), and E2E smoke tests (33 tests total)
- 📖 **Swagger UI** — interactive API documentation

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      HTTP Clients / WebSocket                       │
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
   │   Handlers  │   │  WebSocket Hub  │   │   /metrics     │
   │  auth/wallet│   │  Real-time feed │   │  Prometheus    │
   │  asset/order│   └─────────────────┘   └────────────────┘
   │  admin      │
   └──────┬──────┘
          │
   ┌──────▼──────────────────────────────────┐
   │           Services Layer                │
   │  AuthService · WalletService · AssetSvc │
   │  OrderService · AdminService            │
   └──────┬──────────────────────────────────┘
          │
   ┌──────▼──────────────────────────────────┐
   │           Repository Layer              │
   │  UserRepo · WalletRepo · OrderRepo      │
   │  AssetRepo · TradeRepo · PortfolioRepo  │
   └──────┬──────────────────────────────────┘
          │
   ┌──────▼──────────────────────────────────┐
   │           Infrastructure                │
   │  PostgreSQL · Redis · Kafka · WebSocket │
   └─────────────────────────────────────────┘

   ┌──────────────────────────────────────────┐
   │       Matching Engine (goroutine)        │
   │  Channel-based · Price-Time Priority     │
   │  Atomic DB transactions · FOR UPDATE     │
   │  Publishes to Kafka + Redis Pub/Sub      │
   └──────────────────────────────────────────┘
```

---

## 🛠️ Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.24 |
| HTTP Framework | Gin |
| ORM | GORM + sqlx + raw SQL migrations |
| Database | PostgreSQL |
| Cache | Redis (go-redis/v9) |
| Message Queue | Kafka (segmentio/kafka-go) |
| Auth | JWT (golang-jwt/v5) + bcrypt |
| Real-time | WebSocket (gorilla/websocket) |
| Metrics | Prometheus (prometheus/client_golang) |
| API Docs | Swagger (swaggo/swag) |
| Logging | Zerolog (structured JSON) |
| Testing | Testify + integration tests against real Postgres |
| CI/CD | GitHub Actions |
| Containerization | Docker + Docker Compose |

---

## 🔑 Key Design Decisions

### Matching Engine
Runs as a single goroutine consuming from a buffered `chan *models.Order`. Both order rows are locked with `SELECT … FOR UPDATE` in ascending ID order to prevent deadlocks. The entire trade — order fills, wallet debits/credits, portfolio updates, trade record — commits in **one ACID transaction**.

### Wallet Consistency
Every balance mutation uses `SELECT … FOR UPDATE` inside a transaction + bumps a `version` column. A `CHECK (balance >= 0)` DB constraint acts as the final safety net against race conditions.

### Idempotency
Order placement accepts an `Idempotency-Key` header stored with a partial unique index `(user_id, idempotency_key)`. Duplicate requests return the original order — no double execution.

### Rate Limiting
Redis-backed **sliding window** rate limiter (`ZADD` + `ZCARD` pipeline). Fails **open** if Redis is unavailable. Returns `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers.

### Observability
- Structured JSON logs via Zerolog
- Prometheus metrics at `/metrics`
- Audit logs in Postgres for every trade, order, and admin action

---

## 📁 Project Structure

```
MultiTrade-Platform/
├── cmd/api/
│   └── main.go              # Entry point — wires all dependencies
├── internal/
│   ├── admin/               # Admin service: user management, analytics
│   ├── auth/                # JWT service, auth service, unit tests
│   ├── cache/               # Redis client (price cache, pub/sub, rate limit)
│   ├── config/              # Viper-based config loader
│   ├── cronjobs/            # Scheduled maintenance (expire orders, reconcile wallets)
│   ├── handlers/            # Gin HTTP handlers
│   ├── kafka/               # Producer, consumers (notification, audit, analytics)
│   ├── matchingengine/      # Price-time priority matching engine + tests
│   ├── middleware/          # JWT auth, RBAC, rate limiting, Prometheus
│   ├── models/              # Domain structs
│   ├── repository/          # Postgres implementations
│   ├── trading/             # Asset service + Order service
│   ├── wallet/              # Wallet service + integration tests
│   └── websocket/           # Hub, client, upgrade handler
├── migrations/              # Sequential SQL migrations
├── scripts/
│   └── smoke_e2e.go         # Full E2E smoke test
├── postman/                 # Postman collection + environment
├── docker-compose.yml       # Postgres + Redis + Kafka local dev stack
└── go.mod
```

---

## 🚀 Quick Start

### Prerequisites
- Go 1.24+
- Docker & Docker Compose

### 1. Start infrastructure
```bash
docker-compose up -d
```
Starts PostgreSQL `:5433` · Redis `:6379` · Kafka/Redpanda `:9092`

### 2. Configure environment
```bash
cp .env.example .env
```

### 3. Run the server
```bash
go run ./cmd/api
```
Migrations run automatically. Server starts on **http://localhost:8080**

### 4. Explore
| URL | Description |
|---|---|
| http://localhost:8080/swagger/index.html | Interactive Swagger UI |
| http://localhost:8080/metrics | Prometheus metrics |
| http://localhost:8080/health | Health check |
| ws://localhost:8080/ws | WebSocket real-time feed |

---

## 📡 API Overview

### Auth
| Method | Path | Description |
|---|---|---|
| POST | `/auth/register` | Register (admin/broker/trader/support) |
| POST | `/auth/login` | Login — returns JWT pair (rate-limited: 10/min/IP) |
| POST | `/auth/refresh` | Exchange refresh token |
| GET | `/auth/me` | Current user info |

### Wallet
| Method | Path | Description |
|---|---|---|
| GET | `/wallet` | Current balance |
| POST | `/wallet/deposit` | Deposit funds |
| POST | `/wallet/withdraw` | Withdraw funds |
| GET | `/wallet/transactions` | Paginated transaction history |

### Orders (TRADER only)
| Method | Path | Description |
|---|---|---|
| POST | `/orders` | Place limit order BUY/SELL with Idempotency-Key |
| GET | `/orders` | Own order history |
| DELETE | `/orders/:id` | Cancel pending order |

### Admin
| Method | Path | Description |
|---|---|---|
| GET | `/admin/users` | List users |
| PATCH | `/admin/users/:id/status` | Suspend/activate user |
| GET | `/admin/analytics/volume` | Hourly trade volume |
| GET | `/admin/audit` | Paginated audit logs |
| GET | `/admin/health` | DB + Redis + Kafka health |

---

## 👥 Roles

| Role | Permissions |
|---|---|
| `admin` | Full admin API, user management, analytics |
| `broker` | Create and manage assets |
| `trader` | Place/cancel orders, wallet operations |
| `support` | Read-only access |

---

## 🧪 Tests

```bash
# All tests
go test ./...

# By package
go test -v ./internal/auth/...
go test -v ./internal/wallet/...
go test -v ./internal/matchingengine/...

# E2E (server must be running)
go run scripts/smoke_e2e.go
```

| Package | Tests | Type |
|---|---|---|
| `internal/auth` | 9 tests | Unit (stub repo, no DB) |
| `internal/wallet` | 8 tests | Integration (real Postgres) |
| `internal/matchingengine` | 2 tests | Integration — 25 concurrent orders |
| `scripts/smoke_e2e.go` | 14 checks | E2E against live server |

---

## 📬 Postman Collection

Import `postman/TradeVerse.postman_collection.json` and `postman/TradeVerse_Environment.json` into Postman. Set `base_url` to `http://localhost:8080` — Register/Login requests auto-populate the `access_token` variable.

---

## 👩‍💻 Author

**Fathima Sithara** — Backend Engineer (Go · Distributed Systems)

[![GitHub](https://img.shields.io/badge/GitHub-fathimasithara01-black?style=flat-square&logo=github)](https://github.com/fathimasithara01)
[![Email](https://img.shields.io/badge/Email-fathimasithara011%40gmail.com-red?style=flat-square&logo=gmail)](mailto:fathimasithara011@gmail.com)

> Also check out:
> - 💬 [Real-Time Chat App](https://github.com/fathimasithara01/chat-app) — Distributed messaging with Go microservices
> - 🛒 [E-Commerce Platform](https://github.com/fathimasithara01/fullstack-ecommerce-platform) — Next.js + Node.js full-stack app

---

<div align="center">

**⭐ If this project helped you, please give it a star!**

</div>
