# Architecture Documentation

## System Overview

MultiTrade Platform is a high-performance, production-ready trading system built with microservices principles in a monolithic Go application.

```
┌─────────────────────────────────────────────────────────────┐
│                     Client Applications                      │
│         (Web UI, Mobile Apps, Desktop Terminals)             │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTPS/WebSocket
                     ▼
        ┌────────────────────────────┐
        │      Nginx Reverse Proxy   │
        │    (Load Balancing, SSL)   │
        └────────────────┬───────────┘
                         │
        ┌────────────────▼───────────┐
        │   API Gateway & Router     │
        │    (Gin Framework)         │
        └────────────────┬───────────┘
                         │
        ┌────────────────▼───────────────────────────┐
        │         Middleware Stack                    │
        │  ┌─ Authentication (JWT)                   │
        │  ├─ Authorization (RBAC)                   │
        │  ├─ Rate Limiting                         │
        │  ├─ Logging & Tracing                     │
        │  ├─ CORS & Security                       │
        │  └─ Recovery & Error Handling             │
        └────────────────┬───────────────────────────┘
                         │
      ┌──────────────────┼──────────────────┐
      │                  │                  │
      ▼                  ▼                  ▼
   Auth Service   Business Logic      Event System
   - Register       Services           - Kafka Producer
   - Login        - Wallet             - WebSocket Hub
   - Refresh      - Order              - Pub/Sub
                  - Trading
                  - Portfolio
```

## Core Components

### 1. Authentication & Authorization

**JWT Token Flow:**
```
1. User registers → Email validation → Password hashing → User stored
2. User login → Credentials validated → Access + Refresh tokens generated
3. API requests → Access token verified → User context set
4. Token expiry → Refresh token used → New access token issued
```

**RBAC (Role-Based Access Control):**
- Admin: Full system access
- User: Trading and portfolio operations
- Broker: Asset management and market data

### 2. Wallet Management

**Transaction Safety:**
- Row-level locking prevents race conditions
- ACID transactions ensure consistency
- Balance verification before withdrawal
- Fee calculation and tracking

**Transaction Flow:**
```
Request → Validation → Lock Row → Update Balance
       → Create TX Record → Commit → Response
```

### 3. Order Management

**Order Lifecycle:**
```
PENDING → (PARTIALLY_FILLED | FILLED | CANCELLED | REJECTED | EXPIRED)
  ↓
Matching Engine processes order
  ↓
Creates trades when matched
  ↓
Updates wallets and portfolios
  ↓
Emits events via Kafka
```

**Order Types:**
- Market: Immediate execution at current price
- Limit: Execute only at specified price
- Stop: Convert to market order when price triggers

### 4. Matching Engine

**Price-Time Priority Algorithm:**
```
For each incoming order:
  1. Search order book for counter-side orders
  2. Match by price (best price first)
  3. Match by time (FIFO for same price)
  4. Execute partial matches if needed
  5. Create trade records
  6. Update wallets and portfolios
  7. Emit trade events
```

**Concurrency:**
- Single goroutine processes orders sequentially
- Channel-based communication from API handlers
- Prevents race conditions in order book

### 5. Data Persistence

**Database Schema Highlights:**
- Normalized relational schema
- Strategic indexes for query performance
- Foreign keys with cascade deletes
- Audit log table for compliance

**Connection Management:**
- Connection pooling (25 open, 5 idle)
- 5-minute connection lifetime
- Automatic reconnection on failure

### 6. Caching Layer

**Redis Usage:**
- Asset price caching (60-second TTL)
- Session/token caching
- Rate limiting counters
- Order book snapshots
- Real-time price pub/sub

### 7. Event Streaming

**Kafka Topics:**
- order-created: New orders
- order-matched: Matching results
- trade-executed: Trade execution
- notification: User notifications

**Event Processing:**
- Asynchronous event consumption
- Order-level processing
- Notification delivery
- Analytics aggregation

## Deployment Architecture

### Development Environment
```
Docker Compose Stack:
├── PostgreSQL: 5432
├── Redis: 6379
├── Kafka: 9092
├── API: 8080
└── Optional: Prometheus, Grafana
```

### Production Environment
```
Kubernetes / Docker:
├── API Pod(s): Replicated for HA
├── PostgreSQL: Managed database
├── Redis: Cluster or Sentinel
├── Kafka: Cluster
├── Nginx: Reverse proxy + LB
└── Monitoring: Prometheus, ELK, Datadog
```

## Performance Considerations

### Database
- Indexes on frequently queried columns
- Query optimization (explain analyze)
- Connection pooling
- Statement caching

### Caching
- Asset prices cached for 60 seconds
- Order book snapshots in Redis
- User sessions in cache
- Rate limit counters

### API
- Request validation before processing
- Early error returns
- Efficient JSON marshaling
- Connection keep-alive

### Matching Engine
- Single-threaded for consistency
- In-memory order book
- Efficient data structures (priority queue)
- Batch processing support

## Security Features

### Authentication
- Bcrypt password hashing (cost factor: 12)
- JWT with HS256 signing
- Refresh token rotation
- HTTP-only cookies for tokens

### Authorization
- Role-based access control
- Resource ownership checks
- Admin-only operations
- Audit logging of all actions

### Network Security
- HTTPS/TLS encryption
- CORS protection
- Rate limiting per user
- SQL injection prevention

### Data Protection
- Sensitive data logging prevention
- Database transaction isolation
- Wallet transaction immutability
- Audit trail of all changes

## Scalability Strategy

### Horizontal Scaling
- Stateless API servers behind load balancer
- Shared PostgreSQL database
- Redis cluster for caching
- Kafka for distributed events

### Vertical Scaling
- Database connection pooling optimization
- Redis memory management
- API server resource limits
- Matching engine efficiency

### Monitoring & Alerting
- Request latency tracking
- Error rate monitoring
- Database connection pool utilization
- Cache hit rates
- Kafka lag monitoring

## Disaster Recovery

### Backup Strategy
- PostgreSQL continuous archiving
- Daily database backups
- Redis persistence (RDB/AOF)
- Event replay capability

### High Availability
- Database replication (primary/standby)
- Redis sentinel for failover
- Kafka replication factor 3
- Load balanced API servers

## Testing Strategy

### Unit Tests
- Service logic validation
- Repository data access
- Utility function testing

### Integration Tests
- API endpoint testing
- Database transaction testing
- Wallet operations safety
- Order matching accuracy

### Load Testing
- Concurrent user simulation
- Peak load handling
- Database connection exhaustion
- Rate limit enforcement

## Future Enhancements

1. **Advanced Matching**: Support multiple matching algorithms
2. **Derivatives**: Options and futures support
3. **Margin Trading**: Leverage trading capabilities
4. **Advanced Analytics**: ML-based price prediction
5. **Multi-Chain**: Cryptocurrency bridge support
6. **Mobile App**: Native iOS/Android applications
