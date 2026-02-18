# ===============================
# Docker Compose Makefile
# ===============================

# ---- Config ----
DC = docker compose
BASE_FILE = -f docker-compose.yml
KC_FILE = -f docker-compose.keycloak.yml

# Default target
.DEFAULT_GOAL := help

# ===============================
# Help
# ===============================

help:
	@echo ""
	@echo "Docker Compose Commands"
	@echo "------------------------"
	@echo "make up                -> Start main stack"
	@echo "make up-d              -> Start main stack (detached)"
	@echo "make up-build          -> Build & start main stack"
	@echo "make up-build-d        -> Build & start main stack (detached)"
	@echo ""
	@echo "make kc-up             -> Start Keycloak stack"
	@echo "make kc-up-d           -> Start Keycloak stack (detached)"
	@echo "make kc-up-build       -> Build & start Keycloak stack"
	@echo "make kc-up-build-d     -> Build & start Keycloak stack (detached)"
	@echo ""
	@echo "make down              -> Stop main stack"
	@echo "make kc-down           -> Stop Keycloak stack"
	@echo ""
	@echo "make restart           -> Restart main stack"
	@echo "make kc-restart        -> Restart Keycloak stack"
	@echo ""
	@echo "make logs              -> Logs (main stack)"
	@echo "make kc-logs           -> Logs (Keycloak stack)"
	@echo ""
	@echo "make ps                -> Show containers"
	@echo ""

# ===============================
# Main Stack
# ===============================

up:
	$(DC) $(BASE_FILE) up

up-d:
	$(DC) $(BASE_FILE) up -d

up-build:
	$(DC) $(BASE_FILE) up --build

up-build-d:
	$(DC) $(BASE_FILE) up --build -d

down:
	$(DC) $(BASE_FILE) down

restart:
	$(DC) $(BASE_FILE) down
	$(DC) $(BASE_FILE) up -d

logs:
	$(DC) $(BASE_FILE) logs -f

ps:
	$(DC) $(BASE_FILE) ps

# ===============================
# Keycloak Stack
# ===============================

kc-up:
	$(DC) $(KC_FILE) up

kc-up-d:
	$(DC) $(KC_FILE) up -d

kc-up-build:
	$(DC) $(KC_FILE) up --build

kc-up-build-d:
	$(DC) $(KC_FILE) up --build -d

kc-down:
	$(DC) $(KC_FILE) down

kc-restart:
	$(DC) $(KC_FILE) down
	$(DC) $(KC_FILE) up -d

kc-logs:
	$(DC) $(KC_FILE) logs -f
