.PHONY: help up down backend web test build

help:
	@echo "MotoSpec Project Automation"
	@echo "============================"
	@echo "  make up       - Start PostgreSQL & Redis services via Docker Compose"
	@echo "  make down     - Stop Docker Compose services"
	@echo "  make backend  - Run Go backend server (port 8080)"
	@echo "  make web      - Run frontend Vite PWA dev server (port 5173)"
	@echo "  make test     - Run all backend unit tests and frontend build"
	@echo "  make build    - Compile backend binary and bundle frontend dist"

up:
	docker compose up -d

down:
	docker compose down

backend:
	cd backend && go run cmd/api/main.go

web:
	cd web && npm run dev

test:
	cd backend && go test -v ./...
	cd web && npm run build

build:
	cd backend && go build -o bin/motospec-api ./cmd/api
	cd web && npm run build
