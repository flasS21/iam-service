# Kong Integration Guide (IAM Service)

## Objective

Add Kong as the **edge gateway** in front of IAM backend.

Kong is responsible for **traffic control**, not authentication logic.

Backend remains the single source of truth for:

* Session validation
* CSRF validation
* Idle timeout
* Absolute expiry
* Session version enforcement

Kong must not duplicate or replace backend security logic.

---

# Role & Responsibility

You are responsible for:

* Adding Kong API Gateway as edge layer
* Configuring reverse proxy to IAM backend
* Defining routes
* Implementing rate limiting
* Configuring TLS termination (later phase)
* Preparing structure for future mTLS

You must NOT:

* Modify backend authentication logic
* Change session implementation
* Modify middleware
* Implement CSRF at Kong
* Change database schema
* Add session validation logic in gateway

Backend must remain gateway-agnostic.

---

# Architecture Responsibility Split

## Kong (Edge Layer)

Handles:

* TLS termination
* Rate limiting
* Request logging
* IP filtering (if needed)
* CORS (when frontend is added)
* Reverse proxy routing

Does NOT handle:

* Session validation
* CSRF validation
* Redis access
* User authentication logic

---

## Backend (IAM Service)

Handles:

* Session validation
* CSRF validation
* Idle sliding window
* Absolute expiry
* Session version invalidation
* Logout and logout-all

All authentication intelligence remains in backend.

---

# Branching Rules

Protected branches:

* main
* develop

You cannot push directly to them.

You must work in feature branches.

---

# Setup Instructions

## 1. Clone Repository

```bash
git clone https://github.com/flasS21/iam-service.git
cd iam-service
```

## 2. Switch to develop

```bash
git checkout develop
git pull origin develop
```

## 3. Create Kong Feature Branch

```bash
git checkout -b feature/kong-integration
git push -u origin feature/kong-integration
```

---

# Workflow

1. Make changes
2. Commit changes

```bash
git add .
git commit -m "feat: add kong reverse proxy setup"
git push
```

3. Open Pull Request

Branch:

```
feature/kong-integration → develop
```

4. Wait for approval before merge

Do not merge your own PR.

---

# Phase 1 Objective — Basic Gateway

* Add Kong container
* Connect to existing Docker network
* Configure reverse proxy to IAM backend
* Remove public backend port exposure
* Ensure only Kong is exposed publicly

Validation:

* [http://localhost:8000/api/ping](http://localhost:8000/api/ping) works
* Direct backend port is not accessible from host

No backend code changes allowed.

---

# Phase 2 Objective — Traffic Protection

* Add rate limiting plugin
* Enable proxy access logs
* Ensure proper client IP forwarding

Do NOT implement CSRF at gateway.

CSRF remains backend responsibility.

---

# Phase 3 Objective — Transport Security

* Enable TLS termination at Kong
* Prepare internal structure for mTLS (Kong ↔ backend)
* Do not implement mTLS until approved

---

# Security Rules

* Admin API must not be publicly exposed
* Do not trust all IPs (no 0.0.0.0/0 trusted IP configuration)
* Do not add authentication plugins that conflict with backend session system
* Do not move session validation into Kong

---

# Non-Goals

Kong is not:

* Identity provider
* Session manager
* CSRF engine
* Business logic layer

It is strictly an edge traffic controller.

---