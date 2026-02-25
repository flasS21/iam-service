# 1️⃣ High-Level Architecture (Post-Kong Integration)

## 🧱 Current (Before Kong)

```plaintext
Browser
   │
   ▼
IAM Service (Go + Gin)
   │
   ├── Redis (Sessions)
   ├── PostgreSQL (Users)
   └── Keycloak (OIDC)
```

Limitations:

* Backend is directly exposed
* No centralized rate limiting
* No TLS termination layer
* No traffic policy control

---

## 🛡 Target Architecture (With Kong)

```plaintext
                    ┌──────────────────────┐
                    │        Browser       │
                    └──────────┬───────────┘
                               │ HTTPS
                               ▼
                    ┌──────────────────────┐
                    │        Kong          │
                    │   (API Gateway)      │
                    ├──────────────────────┤
                    │  • TLS termination   │
                    │  • Rate limiting     │
                    │  • Request logging   │
                    │  • Header forwarding │
                    │  • Routing           │
                    └──────────┬───────────┘
                               │ Internal Docker Network
                               ▼
                    ┌──────────────────────┐
                    │     IAM Service      │
                    │     (Go + Gin)       │
                    └──────────┬───────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
   PostgreSQL                Redis                Keycloak
    (Users)               (Sessions)               (OIDC)
```

Important:

* Kong handles traffic control.
* IAM backend handles authentication and session logic.
* CSRF validation remains inside backend.
* Kong does not inspect or validate session tokens.

---

## 🔐 Trust Boundaries

```plaintext
Internet
   ↓
[ Kong ]               ← Public zone
   ↓
[ IAM Service ]        ← Private Docker network
   ↓
[ Redis / Postgres ]   ← Private infra network
```

After Kong integration:

* IAM will NOT be publicly exposed
* Only Kong exposes ports 80/443
* IAM runs on internal Docker network only

---

# 2️⃣ Intern Task Breakdown (Step-by-Step)

This is simplified for clarity.

---

# 📌 PHASE 1 — Infrastructure Setup

### Step 1 — Add Kong

You must:

* Add Kong service container
* Connect Kong to same Docker network as IAM
* Ensure IAM is not publicly exposed
* Ensure only Kong exposes public port

Deliverable:

```plaintext
docker-compose.kong.yml
```

Validation:

* [http://localhost:8000/api/ping](http://localhost:8000/api/ping) works
* Direct backend port access fails

---

# 📌 PHASE 2 — Configure Reverse Proxy

Use declarative configuration (kong.yaml).

Do NOT use Admin API curl commands.

Define:

* Service: iam-service
* Upstream URL: internal Docker service name
* Route paths: /api, /dashboard, /oauth, /health

Browser → Kong → IAM

No backend changes allowed.

---

# 📌 PHASE 3 — Add Rate Limiting

Add rate limiting plugin at service level.

Example policy:

* 60 requests per minute
* Local policy (no Redis yet)

This protects login endpoints from abuse.

Do NOT apply extremely low limits during development.

---

# 📌 PHASE 4 — Logging & Header Forwarding

Ensure:

* X-Forwarded-For is passed correctly
* Client IP is preserved
* Access logs enabled

Backend may later read forwarded IP for auditing.

---

# 📌 PHASE 5 — TLS Termination (Basic)

You will:

* Configure Kong to expose HTTPS
* Use self-signed cert for local dev
* Terminate TLS at Kong
* Forward HTTP internally to backend

Do NOT implement mTLS yet.

---

# 🚫 What You Must NOT Touch

You must not:

* Implement CSRF at Kong
* Validate sessions at Kong
* Modify backend middleware
* Modify session store
* Modify resolver
* Modify database schema
* Add authentication plugins to Kong

Authentication intelligence belongs to backend.

---

# 📦 Final Deliverables

1. docker-compose.kong.yml
2. Kong running and proxying IAM
3. Backend accessible only via Kong
4. Rate limiting active
5. Documentation: /docs/kong-setup.md

---

# 🧠 Backend Refactor Required?

Minimal.

Only possible adjustment:

* Proper handling of X-Forwarded-For

No session logic changes required.

---

# 🔮 Future Scope (Not Now)

These are not part of current task:

* mTLS between Kong and IAM
* JWT validation at gateway
* Distributed rate limiting (Redis policy)
* WAF plugin
* Bot detection
* Circuit breaker
* Global authentication offloading

These require architectural discussion before implementation.

---