# Auth Flow Test Runbook

Validates the end-to-end authentication flow including OAuth login, protected API access, session logout, and global session invalidation through the IAM service and Kong gateway.

## Scope

- OAuth login via `/oauth/login`
- Protected API endpoints (`/api/ping`, `/api/me`)
- Protected dashboard route (`/dashboard`)
- Logout and logout-all (`/auth/logout`, `/auth/logout-all`)
- Request correlation header (`X-Request-ID`) propagation

## Prerequisites

- Docker and Docker Compose installed
- Keycloak configured and reachable by IAM service
- Certificates exist in `./certs` (`ca.crt`, `iam.crt`, `iam.key`, `kong.crt`, `kong.key`)
- Local hosts entries:

```bash
sudo sh -c 'grep -q "api.localhost" /etc/hosts || echo "127.0.0.1 api.localhost" >> /etc/hosts'
sudo sh -c 'grep -q "internal.localhost" /etc/hosts || echo "127.0.0.1 internal.localhost" >> /etc/hosts'
```

## Start Services

```bash
# Base services
docker compose up -d iam-postgres iam-redis

# IdP
docker compose -f docker-compose.keycloak.yml up -d

# IAM API (HTTPS + mTLS for upstream)
docker compose up -d iam-api

# Kong + static UI
docker compose -f docker-compose.kong.yml up -d
```

Optional health checks:

```bash
docker ps
curl -i -H "Host: api.localhost" http://localhost:8000/gateway/health
curl -i -H "Host: api.localhost" http://localhost:8000/health
```

Expected:
- `/gateway/health` -> `200` and body contains `{"status":"ok"}`
- `/health` -> `200` from backend

## Browser Auth Flow

1. Open: `http://api.localhost:8000/`
2. Click **Enter Dashboard**.
3. Click **Login with OAuth**.
4. Complete login at IdP.
5. Return to dashboard and verify session is authenticated.

### Validate in Browser DevTools

- Network requests are to `api.localhost:8000` (not direct backend port)
- `/api/ping` returns `200`
- `X-Request-ID` response header is present

## Protected Route Checks

### Without session cookie (incognito)

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/api/ping
curl -i -H "Host: api.localhost" http://localhost:8000/api/me
curl -i -H "Host: api.localhost" http://localhost:8000/dashboard
```

Expected: `401` (or redirect behavior, depending on middleware response policy).

### With active browser session

Use dashboard buttons:
- **Check Session** -> authenticated status
- (Optional) open `http://api.localhost:8000/dashboard` directly -> loads protected page

## Logout / Logout-All Test

From dashboard:
1. Click **Logout** -> expect success message and return to index.
2. Login again.
3. Click **Logout All** -> expect all sessions invalidated.

Post-logout validation:
- `GET /api/ping` should no longer return authenticated response.

## Optional API-level Logout Test (manual cookie+csrf)

If you want to test with `curl`, capture `csrf_token` and session cookie from browser and run:

```bash
curl -i -X POST \
  -H "Host: api.localhost" \
  -H "X-CSRF-Token: <csrf_token_value>" \
  -H "Cookie: <full_cookie_header>" \
  http://localhost:8000/auth/logout
```

`logout-all`:

```bash
curl -i -X POST \
  -H "Host: api.localhost" \
  -H "X-CSRF-Token: <csrf_token_value>" \
  -H "Cookie: <full_cookie_header>" \
  http://localhost:8000/auth/logout-all
```

Expected: `200` or `204`.

## Troubleshooting

- Login redirect loop:
  - Verify Keycloak container is up and redirect URL config is correct.
- `502` from Kong:
  - Check IAM API is up and cert paths are mounted correctly.
- `401` after successful login:
  - Verify browser is using `api.localhost` consistently and cookies are present.
- Missing `X-Request-ID`:
  - Check Kong `correlation-id` plugin in `kong.yaml`.

## Stop / Cleanup

```bash
docker compose -f docker-compose.kong.yml down
docker compose -f docker-compose.keycloak.yml down
docker compose down
```
