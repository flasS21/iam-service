# mTLS Setup Guide – Kong to IAM

##  Overview
This document provides step-by-step instructions to set up and test Mutual TLS (mTLS) between Kong Gateway and IAM backend.

---

##  Prerequisites

- Docker & Docker Compose  
- OpenSSL  
- Git  

---

##  Step-by-Step Setup

### 1. Generate Certificates

Create `certs` directory and generate all required certificates:

```bash
# Create certs directory
mkdir -p certs
cd certs
```

---


### 1.1 Create SAN Configuration (`san.conf`)
Place `san.conf` inside the `certs/` folder (same location where certificates are generated)

```ini
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
CN = iam-api

[v3_req]
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = iam-api
DNS.2 = localhost
DNS.3 = iam-service
IP.1 = 127.0.0.1
```

---

### 1.2 Generate Root CA

```bash
openssl req -new -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout ca.key \
  -out ca.crt \
  -subj "/CN=IAM Internal CA"
```

---

### 1.3 Generate IAM Server Certificate

```bash
# Generate private key
openssl genrsa -out iam.key 2048

# Create certificate signing request
openssl req -new -key iam.key -out iam.csr -subj "/CN=iam-api"

# Sign certificate with CA
openssl x509 -req -days 365 -in iam.csr -CA ca.crt -CAkey ca.key \
  -set_serial 02 \
  -out iam.crt \
  -extfile san.conf \
  -extensions v3_req
```

---

### 1.4 Generate Kong Client Certificate

```bash
# Generate private key
openssl genrsa -out kong.key 2048

# Create CSR
openssl req -new -key kong.key -out kong.csr -subj "/CN=kong-client"

# Sign certificate with CA
openssl x509 -req -days 365 -in kong.csr -CA ca.crt -CAkey ca.key \
  -set_serial 01 \
  -out kong.crt
```

---

### 1.5 Verify Certificates

```bash
# Check IAM certificate
openssl x509 -in iam.crt -text -noout | grep -A 2 "Subject:"

# Check Kong certificate
openssl x509 -in kong.crt -text -noout | grep "Subject:"
```

Return to project root:

```bash
cd ..
```

---

## 2. Start Services

```bash
# Start IAM backend and dependencies
docker-compose up -d

# Start Kong
docker-compose -f docker-compose.kong.yml up -d

# Wait for initialization
sleep 10
```

---

##  Testing Commands

###  Test 1: Through Kong (Should Work)

```bash
curl http://localhost:8000/health
```

**Expected:**
```json
{"status":"ok"}
```

**Status Code:** `200 OK`

---

###  Test 2: Direct HTTPS Without Certificate (Should Fail)

```bash
curl -k https://localhost:8443/health
```

**Expected:** TLS handshake error

```text
curl: (56) schannel: failed to read data from server: SEC_E_ILLEGAL_MESSAGE
```

**Status Code:** `000` (connection failed)

 This proves IAM requires a client certificate.

---

###  Test 3: Verify Kong → IAM is HTTPS

```bash
docker logs kong --tail 20 | grep -E "protocol|https"
```

**Expected:**
```json
"service":{"protocol":"https","port":8443}
```

 Proves Kong is using HTTPS upstream.

---

###  Test 4: Verify IAM HTTPS Server

```bash
docker logs iam-service-iam-api-1 --tail 10 | grep "starting HTTPS"
```

**Expected:**
```text
{"level":"INFO","msg":"starting HTTPS server on port 8443"}
```

 Proves IAM is listening for HTTPS connections.

 ###  Test 5: Direct HTTPS WITH Client Certificate (Should Work)

```bash
openssl s_client -connect localhost:8443 \
  -cert certs/kong.crt \
  -key certs/kong.key \
  -CAfile certs/ca.crt
  Expected:

Verify return code: 0 (ok)

Successful TLS handshake

👉 Confirms mTLS is fully working.