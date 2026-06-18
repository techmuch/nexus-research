.PHONY: all build-frontend build-backend build clean run

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

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf frontend/dist/
