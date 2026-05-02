# Session Authority

Production-grade IAM service with OAuth/OIDC authentication, Redis session management, CSRF-protected endpoints, Kong gateway enforcement, and mutual TLS between gateway and backend.

> Product name: **Session Authority**.
> Repository slug remains `iam-service` for compatibility with existing history, links, and imports.

## Project Identity

**Session Authority** delivers:
- Session lifecycle management with Redis persistence
- Policy enforcement at the gateway edge
- Comprehensive authentication and authorization controls

## Key Capabilities

- OAuth/OIDC login flow with callback handling and PKCE
- Session lifecycle in Redis (create, validate, invalidate, logout-all)
- CSRF-protected logout endpoints
- Protected API and web dashboard routes
- Kong as edge gateway for routing, rate limiting, security headers, request correlation
- Host-based route isolation (`api.localhost` vs `internal.localhost`)
- mTLS on upstream path (Kong -> IAM API)

## 1-Page Architecture

```text
Browser
  |
  |  http://api.localhost:8000
  v
Kong Gateway (DB-less)
  - Host/path routing
  - rate-limiting
  - response-transformer (security headers)
  - correlation-id (X-Request-ID)
  - ip-restriction on internal host/route
  |
  +--> static-frontend (nginx, dashboard test UI)
  |
  +--> iam-backend (https://iam-api:8443, mTLS)
          |
          +--> Redis (session store)
          +--> Postgres (users + identity data)
          +--> Keycloak (OIDC provider)
```

## Route Overview

Public host: `api.localhost`
- `/` and `/dashboard.html` -> frontend test UI
- `/oauth/*` -> OAuth login/callback
- `/health`
- `/api/*` (requires session auth)
- `/dashboard` (requires session auth)
- `/auth/logout` and `/auth/logout-all` (session + CSRF)
- `/gateway/health` (Kong self-check)

Internal host: `internal.localhost`
- `/admin/*` (session auth + IP restriction at Kong)

## Quick Start (Demo Mode)

1) Add host entries

```bash
sudo sh -c 'grep -q "api.localhost" /etc/hosts || echo "127.0.0.1 api.localhost" >> /etc/hosts'
sudo sh -c 'grep -q "internal.localhost" /etc/hosts || echo "127.0.0.1 internal.localhost" >> /etc/hosts'
```

2) Start services

```bash
docker compose up -d iam-postgres iam-redis
docker compose -f docker-compose.keycloak.yml up -d
docker compose up -d iam-api
docker compose -f docker-compose.kong.yml up -d
```

3) Open demo UI
- [http://api.localhost:8000/](http://api.localhost:8000/)

## Demo Flow

1. Open `http://api.localhost:8000/`
2. Click **Enter Dashboard**
3. Click **Login with OAuth**
4. Complete IdP login
5. On dashboard, verify:
   - session check returns authenticated
   - request correlation header present (`X-Request-ID`)
6. Trigger **Logout** and verify protected endpoint returns 401
7. Trigger **Logout All** and confirm all user sessions invalidated
8. Demonstrate route isolation:
   - `api.localhost/admin` returns 404 (not exposed on public host)
   - `internal.localhost/admin` enforces IP allowlist + authentication

## Validation Commands

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/gateway/health
curl -i -H "Host: api.localhost" http://localhost:8000/health
curl -i -H "Host: api.localhost" http://localhost:8000/api/ping
curl -i -H "Host: api.localhost" http://localhost:8000/admin
curl -i -H "Host: internal.localhost" http://localhost:8000/admin/users
```

## Runbooks

- Auth flow: `auth-flow-test.md`
- Kong validation: `kong-test-runbook.md`

## Tech Stack

- Go + Gin
- Redis
- PostgreSQL
- Keycloak (OIDC)
- Kong Gateway (DB-less)
- Docker Compose
