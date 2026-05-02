# Kong Test Runbook

This runbook validates Kong routing, host isolation, plugin behavior, and upstream integration with IAM.

## Current Gateway Topology

- Public host: `api.localhost:8000`
- Internal host: `internal.localhost:8000`
- Upstream service: `https://iam-api:8443`
- Static UI served by `static-server` through Kong
- Admin API is local-only: `127.0.0.1:8001`

## Prerequisites

- Docker Compose stack running
- `/etc/hosts` includes:
  - `127.0.0.1 api.localhost`
  - `127.0.0.1 internal.localhost`

## Start / Reload Kong Stack

```bash
docker compose up -d iam-postgres iam-redis iam-api
docker compose -f docker-compose.kong.yml up -d
```

If config changed:

```bash
docker compose -f docker-compose.kong.yml restart kong
```

## Test Matrix

## 1) Gateway Liveness

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/gateway/health
```

Expected:
- `200 OK`
- Body contains `{"status":"ok"}`

## 2) Public Backend Route Reachability

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/health
curl -i -H "Host: api.localhost" http://localhost:8000/api/ping
```

Expected:
- `/health` returns `200`
- `/api/ping` reaches backend route (may return `401` if no session)

## 3) Static UI via Kong

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/
curl -i -H "Host: api.localhost" http://localhost:8000/index.html
curl -i -H "Host: api.localhost" http://localhost:8000/dashboard.html
```

Expected:
- HTML content is returned (`200`)
- Confirms static UI is served by Kong route, not direct filesystem host serving

## 4) Internal Route Isolation (Public Host)

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/admin
curl -i -H "Host: api.localhost" http://localhost:8000/admin/users
```

Expected:
- Not exposed on public host (typically `404`/no matching route)

## 5) Internal Route Access Policy (Internal Host)

```bash
curl -i -H "Host: internal.localhost" http://localhost:8000/admin/users
```

Expected:
- If source IP not allowed by `ip-restriction`: `403`
- If IP allowed and no auth cookie: `401`
- If IP allowed and authenticated session: upstream protected response

Allowed ranges currently include:
- `127.0.0.1`
- `::1`
- `172.18.0.0/16`

## 6) Correlation ID Header

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/health | rg "X-Request-ID"
```

Expected:
- `X-Request-ID` present on response

## 7) Security Headers

```bash
curl -i -H "Host: api.localhost" http://localhost:8000/health | rg "X-Frame-Options|X-Content-Type-Options|Referrer-Policy"
```

Expected headers:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`

## 8) Rate Limiting

Configured: `60 requests/minute per IP`

Burst test:

```bash
for i in $(seq 1 80); do
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: api.localhost" http://localhost:8000/health)
  echo "$i $code"
done
```

Expected:
- Initial requests mostly `200`
- Eventually `429 Too Many Requests`

## 9) Verify Backend Is Not Publicly Exposed

Current compose removes direct `iam-api` host port mapping.

Host should fail to connect directly:

```bash
curl -k -i https://localhost:8443/health
```

Expected:
- Connection failure from host (or no route), proving Kong is the only public ingress.

## 10) Kong Admin API Local-only Check

```bash
curl -i http://127.0.0.1:8001/status
```

Expected:
- Works locally.
- Not exposed on external interface.

## Useful Logs

```bash
docker logs kong --tail 200
docker logs iam-service-iam-api-1 --tail 200
docker logs static-server --tail 200
```

## Common Failures

- `502 Bad Gateway`:
  - IAM container down, TLS cert mismatch, or upstream handshake issue.
- All requests return `404`:
  - Host header mismatch (`api.localhost` / `internal.localhost` required).
- No `429` during rate test:
  - Confirm requests are from same source IP and within one minute.

## Stop

```bash
docker compose -f docker-compose.kong.yml down
docker compose down
```
