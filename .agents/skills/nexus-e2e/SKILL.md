---
name: nexus-e2e
description: Instructions for writing end-to-end tests for the web application using Playwright.
---

# Nexus E2E Testing Guidelines

This skill guides agents on how to safely structure, execute, and debug end-to-end browser integration tests for NEXUS Research applications using Playwright.

## 1. Test Environment Setup
- **Executing Tests**: All E2E tests should be triggered via `npm run test:e2e` inside the `frontend/` directory.
- **Backend Dependency**: Playwright requires the compiled Go backend to be running to serve the embedded frontend and API routes. The test script or your test orchestration process must start the Go server before Playwright runs, or configure Playwright's `webServer` option to boot the binary.

## 2. Writing Playwright Tests
- **Location**: Store tests in the `frontend/tests/` directory (or wherever configured in `playwright.config.ts`).
- **Selectors**: Use semantic, accessibility-focused selectors like `getByRole`, `getByLabel`, or `getByText` whenever possible, reflecting the best practices of modern web guidance.
- **Authentication**: When testing guarded routes, structure your tests to either mock the auth endpoints or perform a programmatic login sequence via a shared testing utility before navigating to the target page.

## 3. Dealing with Streaming/SSE
- Playwright tests involving the AI chat or other SSE components must correctly wait for specific network events or DOM mutations rather than relying on static timeouts, since streaming responses yield data asynchronously.
