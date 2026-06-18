---
name: nexus-style
description: Build, maintain, and automate single-binary, TUI-enabled, AI-augmented applications using the architectural conventions and values established in the NEXUS project.
---

# NEXUS Application Development Guidelines

This skill defines the core design patterns, architectural conventions, and automated development loop established in the NEXUS project. Use this skill when modifying, extending, or creating new features in NEXUS-like systems.

---

## 1. Core Architecture & Values

NEXUS-style applications prioritize **self-contained deployment**, **low operational friction**, and **AI-augmented development**.

### 1.1 Single-Binary Deployment & Multitenancy
- **Multitenant Architecture**: Start from a strict multitenant foundation. All data schemas, API routes, and background processes must isolate tenant context by default using a robust `tenant_id` scoping strategy.
- **Backend**: Concurrency-friendly backend (e.g., Go) serving REST APIs and static content.
- **Frontend**: Single Page Application (React/Vite) compiled and embedded directly into the binary at compile-time (using language features like `go:embed`).
- **Database**: PostgreSQL for robust handling of concurrent writes, multi-tenant partitioning, and historical analytics, or local SQLite for stand-alone mode.
- **Command Line**: Unified CLI structure using standard CLI libraries (e.g., Cobra).

### 1.2 Frontend Shell Integration
- **Shell Wrapper**: Utilize the official `@nexus/nexus-shell` npm library to wrap frontend components and govern core application layouts.
- **Consistency**: Ensure all newly developed UI views conform to the operational standards and component life-cycles managed by `nexus-shell`.

### 1.3 Identity, Access Control, & Audit Logs
- **User Management**: Core user provisioning, profile management, and authentication life-cycles must be implemented as a foundational layer.
- **Permission Controls**: Implement Role-Based Access Control (RBAC) or Attribute-Based Access Control (ABAC) enforced at both the API gateway and the database level.
- **Action Logging**: Every system mutation, sensitive read, administrative change, and user authentication event must be logged to an immutable, timestamped audit trail.

### 1.4 Native Usage & Behavior Analytics
- **Telemetry Ingestion**: Implement native, built-in analytics engines to capture and aggregate platform usage patterns directly within the platform.
- **View & Frequency Tracking**: Identify which specific application views and workflows are accessed, alongside their operational frequency and velocity.
- **User-Attributed Context**: Map interactions to the active user context where permitted, tracking individual workflow paths across sessions.
- **Demographic & Environment Capture**: Capture system-accessible metadata including geographical location (via IP lookups), browser/device environments, local language preferences, and tenant organization metrics to build rich behavioral profiles.

---

## 2. Interactive TUI & CLI Development

Administrating a NEXUS application must feel premium. Make CLI experiences visual, interactive, and resilient.

### 2.1 Guided Configurations (`Huh`)
- Use interactive form engines (like `huh` in Go) for CLI setups (`nexus config init`).
- **Resilient Validation**: Connection failures (to database or AI endpoints) should show clear warning alerts but **must not block progress**. Allow users to continue editing and save the draft config.
- **Silent Mode**: Support a `--silent` flag to skip interactive prompts and immediately write default configuration parameters.

### 2.2 Live State Monitoring & Operations (`Bubble Tea` / `Lip Gloss`)
- Use rich visual formatting (`lipgloss`) with cohesive palettes.
- Terminal output should dynamically detect terminal capabilities, gracefully falling back to plain ASCII and suggesting a terminal upgrade if a basic terminal is detected.
- **System Maintenance TUI**: Provide a streamlined, interactive terminal user interface for administrative tasks, specifically managing database/system **backups** and **restores** with real-time progress indicators.

---

## 3. Context-Aware LLM & Streaming Integration

Integrate AI not just as a static API, but as a real-time, context-aware companion.

### 3.1 Context Tracking
- Track user sitemap transitions and active views in frontend context (e.g., `usePageContext`).
- Feed the current **View Context** (e.g. `"Viewing Task #3: Fix login bug, Status: draft"`) directly into backend AI requests.
- The chat handler must inject this context into the system prompt so the assistant responds dynamically to what the user is looking at.

### 3.2 Streaming & History
- Leverage Server-Sent Events (SSE) `text/event-stream` for low-latency streaming responses.
- Store turn-by-turn chat history in database tables and retrieve the last ~10 turns (20 messages) to maintain sliding-window conversational memory.

---

## 4. AI-Augmented Development Loop (The Developer Ecosystem)

NEXUS automates its own development through a closed-loop task execution system.

```mermaid
graph TD
    A[requirements.md] -- 1. req export / task sync --> B(Database tasks Table)
    B -- 2. Web UI selection / is_selected=true --> C(Task Queue)
    C -- 3. run `make develop` --> D[develop.sh]
    D -- 4. Prompts LLM with tasks + reqs --> E[Gemini CLI]
    E -- 5. Code changes / Updates docs --> F[Git Commit & Push]
    F -- 6. req import / task sync --> B