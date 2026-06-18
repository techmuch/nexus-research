import { test, expect } from '@playwright/test';

test.describe('NEXUS Research Station Login & Auth E2E Tests', () => {
  test('should display the login screen initially and handle validation errors', async ({ page }) => {
    // Mock check auth to return not authenticated
    await page.route('**/api/auth/check', async (route) => {
      await route.fulfill({ json: { authenticated: false } });
    });

    // Mock login endpoint to fail
    await page.route('**/api/login', async (route) => {
      await route.fulfill({ 
        status: 401,
        json: { error: 'invalid username or password' } 
      });
    });

    await page.goto('/');

    // Verify login card elements
    await expect(page.locator('text=NEXUS RESEARCH STATION')).toBeVisible();
    await expect(page.locator('button:has-text("ESTABLISH LINK")')).toBeVisible();

    // Fill in credentials and submit
    await page.fill('input[placeholder="e.g. admin"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'wrongpassword');
    await page.click('button:has-text("ESTABLISH LINK")');

    // Verify error message is displayed
    await expect(page.locator('text=invalid username or password')).toBeVisible();
  });

  test('should log in successfully and render the workspace', async ({ page }) => {
    // Mock check auth to return not authenticated initially
    await page.route('**/api/auth/check', async (route) => {
      await route.fulfill({ json: { authenticated: false } });
    });

    // Mock login endpoint to succeed
    await page.route('**/api/login', async (route) => {
      await route.fulfill({ json: { status: 'ok', username: 'admin' } });
    });

    await page.goto('/');

    // Fill in credentials and submit
    await page.fill('input[placeholder="e.g. admin"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'adminpassword');
    await page.click('button:has-text("ESTABLISH LINK")');

    // Verify we bypass the login screen and show the workbench header
    await expect(page.locator('.font-bold.text-lg.text-primary:has-text("NEXUS RESEARCH STATION - TERMINAL ADMIN")')).toBeVisible();
  });
});

test.describe('NEXUS Research Station Workbench E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Mock check auth to return authenticated immediately
    await page.route('**/api/auth/check', async (route) => {
      await route.fulfill({ json: { authenticated: true, username: 'admin' } });
    });

    // Mock status endpoint to return a mock payload
    await page.route('**/api/status', async (route) => {
      await route.fulfill({ 
        json: { status: 'ok', uptime: '1m', version: '0.1.0', db_connected: true } 
      });
    });

    await page.goto('/');
  });

  test('should load the workspace with correct title and header', async ({ page }) => {
    await expect(page.locator('.font-bold.text-lg.text-primary:has-text("NEXUS RESEARCH STATION - TERMINAL ADMIN")')).toBeVisible();
  });

  test('should display the Dashboard and default active panels', async ({ page }) => {
    // Click on the Dashboard tab button to make it active (since Documentation opens last)
    await page.locator('.flexlayout__tab_button').filter({ hasText: 'Dashboard' }).first().click();

    // Verify that the Dashboard components are loaded
    await expect(page.locator('text=NEXUS Research Station').first()).toBeVisible();
    await expect(page.locator('text=Autonomous Multi-Agent Orchestration').first()).toBeVisible();
    await expect(page.locator('text=Go API Connection')).toBeVisible();
  });

  test('should render and navigate embedded documentation', async ({ page }) => {
    // Verify that the Documentation tab is present and contains content
    await expect(page.locator('text=Getting Started with NEXUS Research Station')).toBeVisible();

    // Find and click on the "Technical Architecture" navigation button in the documentation sidebar
    const archButton = page.locator('button:has-text("Technical Architecture")');
    await expect(archButton).toBeVisible();
    await archButton.click();

    // Verify that the documentation content shifts to Technical Architecture
    await expect(page.locator('h1').filter({ hasText: 'Technical Architecture' })).toBeVisible();
    await expect(page.locator('text=Registry-Driven Frontend')).toBeVisible();
  });
});
