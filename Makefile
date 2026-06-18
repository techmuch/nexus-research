.PHONY: all build-frontend build-backend build clean run test-backend test-frontend-e2e test-all

all: build

build-frontend:
	@echo "Building frontend..."
	cd frontend && npm run build

build-backend:
	@echo "Building Go backend..."
	go build -o bin/nexus-research main.go

build: build-frontend build-backend
	@echo "Build complete! Binary is at bin/nexus-research"

run: build-frontend
	@echo "Running backend server..."
	go run main.go serve

test-backend:
	@echo "Running Go backend unit tests with coverage..."
	go test -coverprofile=coverage.out ./cmd/... ./server/...
	@go tool cover -func=coverage.out | grep total

test-frontend-e2e:
	@echo "Running Playwright E2E tests..."
	cd frontend && npm run test:e2e

test-all: test-backend test-frontend-e2e

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf frontend/dist/
