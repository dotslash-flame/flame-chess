import { test, expect } from '@playwright/test';

test.describe('Tier 1: Authentication & Theme Infrastructure', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('health check endpoint responds with status ok', async ({ request }) => {
    const response = await request.get('/healthz');
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.status).toBe('ok');
  });

  test('landing page loads correctly', async ({ page }) => {
    await expect(page).toHaveTitle(/Flame Chess/i);
    const landingView = page.locator('#view-landing');
    await expect(landingView).toBeVisible();
  });

  test('theme toggle switches between light and dark mode', async ({ page }) => {
    const html = page.locator('html');

    // Default theme should be dark or as configured
    const initialTheme = await html.getAttribute('data-theme');
    expect(initialTheme).toBeTruthy();

    // Toggle theme via theme toggle button or helper
    const themeBtn = page.locator('#themeToggleBtn');
    if (await themeBtn.isVisible()) {
      await themeBtn.click();
    } else {
      await page.evaluate(() => window.toggleTheme && window.toggleTheme());
    }

    const toggledTheme = await html.getAttribute('data-theme');
    expect(toggledTheme).not.toBe(initialTheme);
  });

  test('dev login authentication sets session cookie and loads user profile', async ({ page, request }) => {
    // Execute dev-login endpoint
    const loginRes = await request.post('/auth/dev-login', {
      form: { name: 'Alice' }
    });
    expect(loginRes.ok()).toBeTruthy();
    const loginData = await loginRes.json();
    expect(loginData.identity.name).toBe('Alice');

    // Verify /api/me works with session
    const meRes = await request.get('/api/me');
    expect(meRes.ok()).toBeTruthy();
    const meData = await meRes.json();
    expect(meData.user.display_name).toBe('Alice');
  });
});
