# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| `main`  | ✅ Active  |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues by emailing **engineering@tradeverse.io** with:

- A description of the vulnerability and its potential impact
- Steps to reproduce
- Any proof-of-concept code (if applicable)

You will receive a response within 48 hours. We aim to release a fix within 7 days of confirmation.

## Security Considerations

### Authentication
- Passwords hashed with bcrypt (cost factor 10)
- JWT access tokens expire in 15 minutes; refresh tokens in 7 days
- Suspended accounts are rejected at login time

### Rate Limiting
- Login endpoint: 10 requests/minute per IP (Redis sliding window)
- Order placement: 60 requests/minute per authenticated user
- Wallet operations: 30 requests/minute per authenticated user

### Financial Integrity
- All wallet mutations use `SELECT ... FOR UPDATE` row-level locks inside DB transactions
- Database `CHECK (balance >= 0)` constraint as the final safety net
- Order matching locks both order rows in ascending ID order to prevent deadlocks

### Infrastructure
- Never commit `.env` or credentials to version control
- Use `verify-full` SSL mode for Postgres in production
- `/metrics` endpoint is restricted to internal networks by the Nginx config
- Run the container as a non-root user (`nonroot:nonroot` in the Dockerfile)
