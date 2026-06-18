# 🚀 NEXUS Research Station

NEXUS Research Station is an autonomous multi-agent orchestration and analysis workbench. It combines a concurrency-friendly, single-binary Go backend with a professional-grade, multi-panel React SPA dashboard powered by the [nexus-shell](https://github.com/techmuch/nexus-shell) library.

---

## 🏗 System Architecture

The application is architected for zero-dependency deployment:
* **Backend**: Written in Go, exposing high-performance REST APIs and CLI commands via Cobra.
* **Frontend**: A React/TypeScript Single Page Application built on the `nexus-shell` workspace engine, compiled into static assets.
* **Compilation**: The static assets from `/frontend/dist` are embedded directly into the Go binary at compile-time using Go's native `go:embed` functionality.

---

## 🚦 Getting Started

### Prerequisites
* **Go**: v1.21 or later
* **Node.js**: v18 or later (with `npm`)

### Installation & Quick Start

1. **Clone the repository**:
   ```bash
   git clone https://github.com/techmuch/nexus-research.git
   cd nexus-research
   ```

2. **Install frontend dependencies**:
   ```bash
   cd frontend
   npm install
   cd ..
   ```

3. **Build the application**:
   Use the unified `Makefile` to build both the frontend and backend in one command:
   ```bash
   make build
   ```
   This will output the compiled executable to `bin/nexus-research`.

4. **Run the server**:
   ```bash
   ./bin/nexus-research serve --port 8080
   ```
   Open `http://localhost:8080` in your web browser.

---

## 🛠 Command Line Interface (CLI)

The application binary provides a clean CLI powered by Cobra:

```bash
# Start the web server
./bin/nexus-research serve --port <PORT_NUMBER>

# Show help options
./bin/nexus-research --help
```

---

## 🔌 API Endpoints

The Go backend exposes the following status endpoints:

### `GET /api/status`
Returns the status, uptime, and database connection state of the research station.
* **Response**:
  ```json
  {
    "status": "ok",
    "uptime": "1h2m3s",
    "version": "0.1.0",
    "db_connected": true
  }
  ```

---

## 🧪 Testing and Coverage

We maintain high test coverage (>90%) for the Go backend and verify UI workflows via Playwright E2E tests.

### Run All Tests
```bash
make test-all
```

### Backend Unit Tests & Coverage
Runs Go unit tests and outputs a coverage report:
```bash
# Run unit tests
make test-backend

# View HTML coverage report
go tool cover -html=coverage.out
```

### Frontend E2E Tests
Runs Playwright end-to-end tests for the React application:
```bash
# Install Playwright browsers (first-time only)
cd frontend && npx playwright install && cd ..

# Run E2E tests
make test-frontend-e2e
```

---

## 📖 Embedded Documentation & GitHub Pages

The same documentation available in the repo is also embedded directly in the application UI:
1. Start the server and navigate to `http://localhost:8080`.
2. Click **Help -> Open Documentation** in the main menu to open the Docs workstation.

A live preview of the documentation and interactive shell is automatically deployed to GitHub Pages at:
👉 **[NEXUS Research Station Live](https://techmuch.github.io/nexus-research/)**
