---
name: nexus-server
description: Standards and boilerplate for extending the Go backend, REST API routes, Server-Sent Events, and ensuring security constraints.
---

# Nexus Server Guidelines

This skill provides the required conventions when adding or modifying functionality in the backend HTTP server and business logic layers of NEXUS Research applications.

## 1. REST API Routing
- **Location**: API routes are defined in the `server/` package.
- **Namespacing**: Ensure all endpoints are properly versioned and namespaced (e.g., `/api/v1/...`).
- **Response Format**: Return consistent JSON payloads. Structure responses securely, stripping out any hashed passwords or internal database IDs where not necessary.

## 2. Server-Sent Events (SSE)
- **Streaming Paradigm**: When building AI chat or live data streaming functionality, utilize `text/event-stream`.
- **Implementation**: The HTTP handler should use `http.Flusher` to flush chunks to the client continuously. Handle client disconnects gracefully by listening to `r.Context().Done()`.

## 3. Security and Context
- **Tenant Scope**: Always scope queries and modifications to the current `tenant_id` or user authorization context.
- **Audit Logging**: Any destructive action or sensitive mutation (creates, deletes, renames, password resets, access sharing) **must** invoke `db.LogAuditAction(username, action, resourceType, resourceID, details)`.

## 4. Application Logic
- Keep the `server` layer thin. Put heavy data orchestration inside the `db/` package or specialized service packages, allowing the `server` handlers to focus solely on HTTP request/response serialization.
