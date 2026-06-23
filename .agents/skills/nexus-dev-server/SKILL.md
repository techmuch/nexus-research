---
name: nexus-dev-server
description: Guidelines for rebuilding the Nexus Research Station and managing the development server instance.
---

# Nexus Dev Server Skill

This skill provides step-by-step instructions for building the Nexus Research Station application and running a local development instance. Use this skill when you need to rebuild the project or manage the local server instance to test and inspect changes.

## 1. Quick Rebuild & Run Command

To compile and launch the server in a single command line:
```bash
./build.sh && ./bin/nexus-research serve --port 8080 --db nexus.db
```

## 2. Rebuilding Options

Always use the `./build.sh` script to build the project. Avoid using `go build` directly, because the Go backend embeds the frontend assets from `frontend/dist/` at compile time.

- **Full Build (Default)**: Builds the frontend assets, then compiles the Go binary.
  ```bash
  ./build.sh
  ```
- **Backend Only**: Compiles only the Go backend binary. Use this when only Go files have changed and `frontend/dist/` is already up to date.
  ```bash
  ./build.sh -b
  ```
- **Frontend Only**: Compiles only the React/TypeScript frontend.
  ```bash
  ./build.sh -f
  ```
- **Clean Build**: Cleans any existing artifacts (`bin/`, `frontend/dist/`, `frontend/node_modules/`) and rebuilds everything from scratch.
  ```bash
  ./build.sh -c
  ```
- **With Tests**: Rebuilds the project and runs the backend test suite.
  ```bash
  ./build.sh -t
  ```

## 3. Starting the Server

The application server can be started using the compiled binary. By default, it uses `nexus.db` for the database and serves on port `8080`.

- **Start server with defaults**:
  ```bash
  ./bin/nexus-research serve
  ```
- **Start server with custom port and database**:
  ```bash
  ./bin/nexus-research serve --port 9090 --db dev_nexus.db
  ```

## 4. Stopping the Server

To stop the running instance of the development server, choose one of the following methods:

### Method A: Keyboard Interrupt (If running in foreground)
If you started the server in your active terminal, stop it by pressing:
`Ctrl + C`

### Method B: Kill the Process (If running in background)
If the server is running in the background or you need to force-terminate it, run:
```bash
pkill -f "nexus-research serve"
```
Or find the process ID (PID) and kill it:
```bash
# 1. Find the PID of the running server
pgrep -f "nexus-research serve"

# 2. Kill the process (replace <PID> with the output of the pgrep command)
kill <PID>
```

### Method C: Stale Port Cleanup
If the server did not shut down cleanly and port `8080` is still occupied, free the port:
```bash
kill -9 $(lsof -t -i:8080)
```

## 5. Development Restart Cycle

To verify recent changes, execute the following cycle in your terminal:

1. Stop the active instance:
   ```bash
   pkill -f "nexus-research serve"
   ```
2. Recompile and start the server:
   ```bash
   ./build.sh && ./bin/nexus-research serve
   ```
