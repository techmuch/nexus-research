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
    await expect(page.locator('h1:has-text("Nexus Dialogue Mapper")')).toBeVisible();
  });
});

test.describe('NEXUS Research Station Dialogue Mapper E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Mock check auth to return authenticated immediately
    await page.route('**/api/auth/check', async (route) => {
      await route.fulfill({ json: { authenticated: true, username: 'admin' } });
    });

    await page.goto('/');
  });

  test('should load the workspace with correct title and header', async ({ page }) => {
    await expect(page.locator('h1:has-text("Nexus Dialogue Mapper")')).toBeVisible();
  });

  test('should display the dialogue mapper panels (library, canvas, inspector)', async ({ page }) => {
    // Verify Node Library is loaded
    await expect(page.locator('text=IBIS NODE LIBRARY')).toBeVisible();
    
    // Verify React Flow canvas is loaded
    await expect(page.locator('.react-flow')).toBeVisible();
    
    // Verify Argument Inspector is loaded
    await expect(page.locator('text=Argument Inspector')).toBeVisible();
  });
});
