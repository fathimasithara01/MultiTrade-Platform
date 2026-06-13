# MultiTrade Platform - Quick Start Guide

## 🚀 Getting Started in 5 Minutes

### Prerequisites
- Docker & Docker Compose installed
- Go 1.24.5+ (for local development)
- Git

### Option 1: Docker Compose (Recommended)

```bash
# 1. Clone repository
git clone https://github.com/fathimasithara01/multitrade-platform.git
cd multitrade-platform

# 2. Create environment file
cp .env.example .env

# 3. Start all services
docker-compose up -d

# 4. Verify services are running
docker-compose ps

# 5. Access API
curl http://localhost:8080/health
```

### Option 2: Local Development

```bash
# 1. Install dependencies
go mod download

# 2. Start services (requires PostgreSQL, Redis, Kafka installed)
# Or use: docker-compose up -d postgres redis kafka zookeeper

# 3. Run migrations
# TODO: Set up migration tool

# 4. Start application
go run ./cmd/api/main.go

# 5. Access API
curl http://localhost:8080/health
```

## 📝 First API Call

### 1. Register a User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "testuser",
    "password": "Password@123",
    "confirmPassword": "Password@123"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "email": "user@example.com",
    "username": "testuser"
  },
  "statusCode": 201
}
```

### 2. Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "Password@123"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "email": "user@example.com",
    "username": "testuser",
    "role": "USER",
    "accessToken": "eyJhbGc...",
    "refreshToken": "eyJhbGc...",
    "expiresIn": 900
  },
  "statusCode": 200
}
```

### 3. Get Wallet (Authenticated)
```bash
export TOKEN="<accessToken from login>"

curl -X GET http://localhost:8080/api/v1/wallet \
  -H "Authorization: Bearer $TOKEN"
```

## 🛠️ Common Commands

```bash
# Start services
make docker-up

# Stop services
make docker-down

# View logs
docker-compose logs -f api

# Rebuild API image
make docker-rebuild

# Run tests
make test

# Format code
make fmt

# Lint code
make lint

# Development with hot reload
make air
```

## 📊 Database

### Connection Details
- Host: localhost:5432
- User: postgres
- Password: postgres
- Database: multitrade

### Connect with psql
```bash
psql -h localhost -U postgres -d multitrade
```

### View Tables
```bash
\dt
```

## 🔗 Quick Links

- **API Documentation**: [docs/api.md](docs/api.md)
- **Architecture Guide**: [docs/architecture.md](docs/architecture.md)
- **Configuration**: [.env.example](.env.example)
- **Docker Compose**: [docker-compose.yml](docker-compose.yml)

## 🚨 Troubleshooting

### Port Already in Use
```bash
# Find and kill process
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Or change port in .env
SERVER_PORT=8081
```

### Database Connection Failed
```bash
# Check PostgreSQL is running
docker-compose logs postgres

# Restart services
docker-compose restart postgres
```

### Redis Connection Issues
```bash
# Test connection
redis-cli ping

# Should respond with PONG
```

## 📚 Next Steps

1. Read [docs/api.md](docs/api.md) for full API documentation
2. Check [docs/architecture.md](docs/architecture.md) to understand the system
3. Create sample assets: Check [scripts/seed.sh](scripts/seed.sh)
4. Explore the source code: [internal/](internal/)

## 🤝 Support

- Issues: GitHub Issues
- Documentation: [docs/](docs/)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)

## 📄 License

MIT License - See [LICENSE](LICENSE)

---

**Happy Trading! 🎉**
