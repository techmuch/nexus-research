import { test, expect } from '@playwright/test';

test.describe('NEXUS Research Station Workbench E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the local Vite dev server
    await page.goto('/');
  });

  test('should load the workspace with correct title and header', async ({ page }) => {
    // Verify that the header title of the workbench is visible
    await expect(page.locator('.font-bold.text-lg.text-primary:has-text("NEXUS RESEARCH STATION")')).toBeVisible();
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
