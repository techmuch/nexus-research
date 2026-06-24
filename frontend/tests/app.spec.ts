import { test, expect } from '@playwright/test';

test.describe('NEXUS Research Station Login & Auth E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Mock layout and profile requests to avoid syntax errors from Vite index.html fallback
    await page.route('**/api/layout', async (route) => {
      await route.fulfill({ json: {} });
    });
    await page.route('**/api/profile', async (route) => {
      await route.fulfill({ json: { full_name: 'Admin User', title: 'System Administrator', email: 'admin@example.com', theme: 'system' } });
    });
  });

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
    await expect(page.locator('h1:has-text("NEXUS")')).toBeVisible();
    await expect(page.locator('button:has-text("SIGN IN")')).toBeVisible();

    // Fill in credentials and submit
    await page.fill('input[placeholder="user@example.com"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'wrongpassword');
    await page.click('button:has-text("SIGN IN")');

    // Verify error message is displayed
    await expect(page.locator('text=invalid username or password')).toBeVisible();
  });

  test('should log in successfully and render the workspace', async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.error(`[PAGE CONSOLE ERROR]: ${msg.text()}`);
      }
    });
    page.on('pageerror', err => {
      console.error(`[PAGE UNHANDLED EXCEPTION]: ${err.message}`);
    });

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
    await page.fill('input[placeholder="user@example.com"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'adminpassword');
    await page.click('button:has-text("SIGN IN")');

    // Verify we bypass the login screen and show the workbench header
    await expect(page.locator('h1:has-text("Nexus Research")').first()).toBeVisible();
  });
});

test.describe('NEXUS Research Station Dialogue Mapper E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.error(`[PAGE CONSOLE ERROR]: ${msg.text()}`);
      }
    });
    page.on('pageerror', err => {
      console.error(`[PAGE UNHANDLED EXCEPTION]: ${err.message}`);
    });

    // Mock layout and profile requests
    await page.route('**/api/layout', async (route) => {
      await route.fulfill({ json: {} });
    });
    await page.route('**/api/profile', async (route) => {
      await route.fulfill({ json: { full_name: 'Admin User', title: 'System Administrator', email: 'admin@example.com', theme: 'system' } });
    });

    // Mock projects and files requests
    await page.route('**/api/projects', async (route) => {
      await route.fulfill({
        json: [
          {
            id: 'proj-1',
            name: 'Default Project',
            role: 'owner',
            created_at: '2026-06-23T23:18:27Z'
          }
        ]
      });
    });
    await page.route('**/api/files', async (route) => {
      await route.fulfill({
        json: [
          {
            id: 'file-1',
            project_id: 'proj-1',
            parent_id: null,
            name: 'Workspace Map.map',
            type: 'file',
            content: '{"nodes": [], "edges": []}'
          }
        ]
      });
    });

    // Mock check auth to return authenticated immediately
    await page.route('**/api/auth/check', async (route) => {
      await route.fulfill({ json: { authenticated: true, username: 'admin' } });
    });

    await page.goto('/');
  });

  test('should load the workspace with correct title and header', async ({ page }) => {
    await expect(page.locator('h1:has-text("Nexus Research")').first()).toBeVisible();
  });

  test('should display the dialogue mapper panels (library, canvas, inspector)', async ({ page }) => {
    // If Workspace Map.map is not visible, click Explorer to expand the sidebar
    const fileLocator = page.locator('text=Workspace Map.map');
    if (!await fileLocator.isVisible()) {
      await page.getByRole('button', { name: 'Explorer' }).first().click();
    }
    await expect(fileLocator).toBeVisible();

    // Open the file "Workspace Map.map" by double clicking it in the file explorer sidebar
    await fileLocator.dblclick();

    // Verify Node Library is loaded
    await expect(page.locator('text=IBIS NODE LIBRARY')).toBeVisible();
    
    // Verify React Flow canvas is loaded
    await expect(page.locator('.react-flow')).toBeVisible();
    
    // Verify Argument Inspector is loaded
    await expect(page.locator('text=Argument Inspector')).toBeVisible();
  });
});
