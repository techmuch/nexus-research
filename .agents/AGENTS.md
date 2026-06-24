# Nexus Research Agents Rules

This file outlines the workspace-specific rules, style guidelines, and behavioral constraints for all AI agents operating within the NEXUS Research codebase. Agents must prioritize these rules during code generation, debugging, and task planning.

## 1. Architectural Alignment
- **NEXUS Style**: Always adhere to the architectural conventions and values defined in the `nexus-style` skill. Prioritize single-binary deployments, multitenant architecture, and low operational friction.
- **TUI First**: Maintain a premium Terminal User Interface (TUI) feel. Use `bubbletea` for the main loop, `huh` for interactive forms, and `lipgloss` for styling.

## 2. Testing Constraints
- **TUI Workflows**: All TUI workflow and state transition tests MUST be written using Charm's `teatest` framework. Do not use plain string-matching unit tests for TUI views.
- **Asynchronous Flakiness Mitigation**: Be aware of asynchronous transition flakes when testing complex `huh` forms with `teatest`. Ensure proper `WaitFor` synchronization is used, and insert small sleeps (`time.Sleep`) between rapid key presses (`tm.Send`) if necessary to allow the render loop to catch up. If a test remains stubbornly flaky due to PTY timing, it is acceptable to skip it (`t.Skip`) with a documented explanation to keep the CI build green.

## 3. Database & State Integration
- **Shared Test Database**: Tests run sequentially but share a global `:memory:` SQLite connection via `db.DB`. When writing integration tests, ensure data state is cleanly isolated or reset between test runs (e.g., using a `cleanupDB()` function) to prevent cascading failures.
- **Enterprise & Dual-Database Support**: The system must support both SQLite (for Lite standalone deployments) and PostgreSQL (for Enterprise deployments). All schemas and SQL queries must be compatible with both (e.g., avoiding database-specific syntax where possible or abstracting it).
- **Multi-Tenant Design**: All database queries, schema designs, and backup/restore mechanisms must consider a multi-tenant architecture. Tenant isolation is a strict requirement for enterprise utilization.
- **Data Persistence & Backups**: Ensure that every sensitive system mutation writes to the audit logs. Backups MUST be managed via native database tools (SQLite `VACUUM INTO` / backup APIs or PostgreSQL `pg_dump`/`pg_restore`) targeting a rolling `backups/` subfolder. Do NOT use external third-party tools like Kopia.

## 4. UI/UX Aesthetics
- **Visual Formatting**: Terminal output should be responsive to terminal resize events (`tea.WindowSizeMsg`). Always wrap long text and use cohesive color palettes.
- **Resilient UI**: The application should provide interactive configurations and graceful error warnings instead of panicking or failing immediately on misconfigurations.

## 5. Build System & Execution
- **Build System Enforcement**: Never run `go build` directly. Always use `./build.sh`. Because the Go binary embeds the frontend, the frontend must be compiled into `frontend/dist/` first, which the build script handles automatically.
- **Frontend-Backend Syncing**: When testing changes that require both the Go backend and the React frontend, remember that changes to the `frontend/` directory require a full rebuild (`./build.sh -f` or `make build-frontend`) before running the Go server, unless explicitly using a Vite dev server (`npm run dev`) configured to proxy API requests.

## 6. Audit Logging Mandate
- **State Mutations**: Any time a new API endpoint, TUI command, or database function mutates system state (e.g., creating a project, changing a password), you MUST invoke `db.LogAuditAction(...)` to maintain the immutable system trail required by `nexus-style`.
