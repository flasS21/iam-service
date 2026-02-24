# Kong Integration Guide

## Role & Responsibility

You are responsible for:

- Adding Kong API Gateway as edge layer
- Configuring reverse proxy to IAM backend
- Setting up routing
- Implementing rate limiting
- Configuring TLS termination
- Later: mTLS between Kong and backend

You must NOT:

- Modify backend authentication logic
- Change session implementation
- Modify middleware
- Change database schema

Backend must remain gateway-agnostic.

---

# Branching Rules

Protected branches:
- main
- develop

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

---

# Phase 1 Objective

* Add Kong container to docker-compose
* Proxy traffic to IAM backend
* Remove direct backend port exposure

No backend code changes allowed.

---

# Phase 2 Objective

* Add rate limiting plugin
* Add access logging

---

# Phase 3 Objective [WILL DISCUSS IN DETAILED LATER]

* Enable TLS termination
* Configure internal mTLS

---

# Important

All merges must go through Pull Request.
Never push directly to develop or main.
