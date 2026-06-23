---
name: nexus-frontend
description: Guide on building and modifying web components using the @nexus/nexus-shell wrapper and interacting with the backend Go API.
---

# Nexus Frontend Guidelines

This skill defines the patterns for building the frontend of NEXUS Research applications using Vite, React, and TypeScript.

## 1. Application Shell Integration
- **Wrapper Component**: All major views should be integrated inside the `@nexus/nexus-shell` component. This provides standard navigation, theming, and layout structures consistent with the NEXUS ecosystem.
- **View Context**: When creating or navigating pages, ensure the frontend tracking mechanism (e.g., `usePageContext` or similar context providers) is updated. This context string (e.g. `"Viewing Dashboard"`) must be passed along to AI endpoints so the system understands the user's current intent.

## 2. Go:Embed & Compilation Loop
- **Static Assets**: The backend serves the compiled Vite output (`dist/`) directly using Go's `//go:embed`.
- **Rebuilding**: When you make changes to React components, always run `./build.sh` (or `./build.sh -f` for frontend-only) to compile the assets. Testing the backend executable without this step will serve the older cached UI.

## 3. Communication with the Backend
- **REST APIs**: Interface with `/api/v1/` routes provided by the Go server.
- **Streaming (SSE)**: For AI chat integration or live updates, utilize Server-Sent Events from endpoints mapped in the backend. Use the native `EventSource` API or your preferred SSE wrapper.

## 4. UI/UX Values
- The web experience must match the premium feel of the TUI: dark modes, glassmorphism, fluid micro-animations, and minimal, clean typography.
