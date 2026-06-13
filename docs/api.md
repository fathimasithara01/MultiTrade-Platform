# API Documentation

## Base URL

```
http://localhost:8080/api/v1
https://api.multitrade.com/api/v1  (Production)
```

## Authentication

All endpoints (except `/auth/register` and `/auth/login`) require JWT token in the Authorization header:

```
Authorization: Bearer <access_token>
```

## Response Format

All responses follow this format:

```json
{
  "success": true,
  "data": {},
  "message": "Success message",
  "statusCode": 200
}
```

Error response:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Error description"
  },
  "statusCode": 400
}
```

## Endpoints

### Authentication

#### Register
```
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "username",
  "password": "Password@123",
  "confirmPassword": "Password@123"
}
```

Response: 201 Created

#### Login
```
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "Password@123"
}
```

Response:
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "username",
  "role": "USER",
  "accessToken": "...",
  "refreshToken": "...",
  "expiresIn": 900
}
```

#### Get Current User
```
GET /auth/me
Authorization: Bearer <token>
```

#### Refresh Token
```
POST /auth/refresh
(Token sent as cookie or in body)
```

### Wallet

#### Get Wallet
```
GET /wallet
Authorization: Bearer <token>
```

#### Deposit
```
POST /wallet/deposit
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 100,
  "currency": "USD",
  "method": "bank_transfer"
}
```

#### Withdraw
```
POST /wallet/withdraw
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 50,
  "currency": "USD",
  "address": "0x..."
}
```

#### Get Transactions
```
GET /wallet/transactions?page=1&pageSize=20
Authorization: Bearer <token>
```

### Orders

#### Create Order
```
POST /orders
Authorization: Bearer <token>
Content-Type: application/json

{
  "assetId": 1,
  "side": "BUY",
  "type": "LIMIT",
  "quantity": 10,
  "price": 100
}
```

#### Get Orders
```
GET /orders?page=1&pageSize=20&status=PENDING
Authorization: Bearer <token>
```

#### Get Order Details
```
GET /orders/{orderId}
Authorization: Bearer <token>
```

#### Cancel Order
```
DELETE /orders/{orderId}
Authorization: Bearer <token>
```

### Assets

#### Get All Assets
```
GET /assets?page=1&pageSize=20
```

#### Get Asset
```
GET /assets/{assetId}
```

### Portfolio

#### Get Portfolio
```
GET /portfolio
Authorization: Bearer <token>
```

#### Get Portfolio Assets
```
GET /portfolio/assets?page=1&pageSize=20
Authorization: Bearer <token>
```

## Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created
- `204 No Content`: Request successful, no content
- `400 Bad Request`: Invalid input
- `401 Unauthorized`: Missing/invalid token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `422 Unprocessable Entity`: Business logic error
- `429 Too Many Requests`: Rate limited
- `500 Internal Server Error`: Server error

## Rate Limiting

API has rate limiting of 100 requests per second per user.

Response headers include:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1609459200
```

## Pagination

All list endpoints support pagination:

```
GET /endpoint?page=1&pageSize=20
```

Response:
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "totalCount": 100,
    "totalPages": 5
  }
}
```

## Error Codes

- `INVALID_CREDENTIALS`: Wrong email or password
- `UNAUTHORIZED`: Missing token
- `FORBIDDEN`: Insufficient permissions
- `USER_NOT_FOUND`: User doesn't exist
- `ORDER_NOT_FOUND`: Order doesn't exist
- `INSUFFICIENT_BALANCE`: Not enough funds
- `INVALID_AMOUNT`: Invalid amount value
- `DUPLICATE_ENTRY`: Resource already exists
- `INTERNAL_ERROR`: Server error

## WebSocket

Connect to real-time updates:

```
ws://localhost:8080/ws
wss://api.multitrade.com/ws (Production)

Headers:
Authorization: Bearer <token>
```

Message types:
- `price_update`: Asset price updated
- `order_update`: Order status changed
- `trade_executed`: Trade executed
- `notification`: User notification

## Examples

### Complete Trading Flow

```bash
# 1. Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"trader@example.com","username":"trader","password":"Trader@123","confirmPassword":"Trader@123"}'

# 2. Login
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"trader@example.com","password":"Trader@123"}' | jq -r '.accessToken')

# 3. Get Wallet
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/wallet

# 4. Deposit Funds
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000,"currency":"USD","method":"bank_transfer"}'

# 5. Get Available Assets
curl http://localhost:8080/api/v1/assets?pageSize=10

# 6. Create Buy Order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assetId":1,"side":"BUY","type":"LIMIT","quantity":10,"price":150}'

# 7. Get Orders
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/orders

# 8. Get Portfolio
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/portfolio
```
