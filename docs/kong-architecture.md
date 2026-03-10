# Kong Gateway Architecture – IAM Service

## 1. 🔍 **Overview**

Kong acts as the **edge gateway** for the IAM service, handling:
- ✅ Traffic routing
- ✅ Rate limiting
- ✅ Request logging
- ✅ Security headers
- ✅ Request size limiting
- ✅ Health monitoring

**Important Rule:** Kong handles **traffic-level concerns only**. Authentication, sessions, and CSRF protection remain in the backend.

## 2. 🏗️ **Architecture Diagram**

```
Browser
   │
   ▼
Kong Gateway (Port 8000)
   │
   ▼
IAM Backend (Port 8090 – Internal)
   │
   ├── Redis (Session Store)
   ├── PostgreSQL (Users)
   └── Keycloak (OIDC Identity Provider)
```

## 3. ⚙️ **Service Configuration**

| Property | Value |
|----------|-------|
| **Service Name** | `iam-backend` |
| **Upstream URL** | `http://iam-api:8090` |
| **Tags** | `["iam", "phase1"]` |

```yaml
services:
  - name: iam-backend
    url: http://iam-api:8090
    tags: ["iam", "phase1"]
```

## 4. 🛣️ **Route Definitions**

| Path | Methods | Type | Purpose |
|------|---------|------|---------|
| `/api` | GET, POST, PUT, DELETE | Protected | API endpoints |
| `/dashboard` | GET, POST, PUT, DELETE | Protected | Dashboard page |
| `/oauth` | GET, POST | Public | OAuth login flow |
| `/health` | GET | Public | Service health check |
| `/auth/logout` | POST | POST only | User logout |
| `/auth/logout-all` | POST | POST only | Logout from all devices |
| `/gateway/health` | GET | Public | Gateway health check |

## 5. 🔄 **Request Flow Examples**

### Example 1: Health Check (Public)
Browser → GET /health
→ Kong (Port 8000)
→ IAM Backend (Port 8090)
← 200 OK {"status":"ok"}
← Kong → Browser


### Example 2: OAuth Login Flow

Browser → GET /oauth/login
→ Kong (Port 8000)
→ IAM Backend (Port 8090)
← Redirect to Keycloak (302)
← Kong → Browser

Browser → Keycloak Login Page (Port 8081)
→ Enter credentials
← Redirect to /oauth/callback with code

Browser → GET /oauth/callback?code=xxx
→ Kong (Port 8000)
→ IAM Backend exchanges code
← Session created, cookie set
← Redirect to /dashboard

### Example 3: Protected API Request

Browser → GET /api/me
→ Kong (Port 8000)
→ Kong checks rate limit (60/min)
→ Kong adds X-Forwarded-For header
→ IAM Backend validates session cookie
← 200 OK with user data
← Kong adds security headers
← Browser receives response

### Example 4: Rate Limited Request

Browser → GET /health (60 requests in 1 minute)
→ Kong (Port 8000)
→ Kong rate-limiting plugin triggers
← 429 Too Many Requests
← Kong → Browser

### Example 5: Gateway Health Check

Monitoring → GET /gateway/health
→ Kong (Port 8000)
→ request-termination plugin
← 200 OK {"status":"ok"}
← Kong → Monitoring

## 6. 🔌 **Enabled Plugins**

| Plugin | Configuration | Purpose |
|--------|---------------|---------|
| **rate-limiting** | `minute: 60`, `policy: local`, `limit_by: ip` | Prevents brute force and API abuse |
| **file-log** | `path: /dev/stdout` | Request logging for debugging and audit |
| **request-size-limiting** | `allowed_payload_size: 2 (MB)` | Prevents large payload attacks |
| **response-transformer** | Security headers | Adds browser protection headers |
| **request-termination** | On `/gateway/health` only | Returns gateway health status |

### Security Headers Added

| Header | Value | Protection |
|--------|-------|------------|
| `X-Frame-Options` | `DENY` | Prevents clickjacking |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME sniffing |
| `Referrer-Policy` | `no-referrer` | Prevents referrer leaks |

---

## 7. 🌐 **Network Architecture**

```yaml
networks:
  kong-net:
    driver: bridge
  iam-network:
    external: true
    name: iam-service_default
```

### Ports Exposed

- **8000** – Kong Proxy (Public)
- **8001** – Kong Admin API (Internal – localhost only)
- **8090** – IAM Backend (Internal only)
- **8081** – Keycloak (Internal only)

---

## 8. 🚀 **How to Start Kong**

### Start All Services
```bash
# Start dependencies
docker compose up  iam-postgres iam-redis
docker compose -f docker-compose.keycloak.yml up 

# Start IAM backend
docker compose up  iam-api

# Start Kong
docker compose -f docker-compose.kong.yml up 

# Verify all services
docker ps
```
