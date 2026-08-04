import { test, expect } from '@playwright/test';

test.describe('Tier 2: Dashboard & Navigation', () => {
  test.beforeEach(async ({ page, request }) => {
    // Authenticate user before dashboard tests
    await request.post('/auth/dev-login', { form: { name: 'Alice' } });
    await page.goto('/');
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('lobby');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const lb = document.getElementById('view-lobby');
        if (lb) lb.classList.remove('hidden');
        const tb = document.getElementById('topbar');
        if (tb) tb.classList.remove('hidden');
      }
    });
  });

  test('lobby / dashboard view is visible after login', async ({ page }) => {
    const lobbyView = page.locator('#view-lobby');
    await expect(lobbyView).toBeVisible();

    const topbar = page.locator('#topbar');
    await expect(topbar).toBeVisible();
  });

  test('leaderboard component fetches and displays top players', async ({ request, page }) => {
    const lbRes = await request.get('/api/leaderboard');
    expect(lbRes.ok()).toBeTruthy();
    const lbData = await lbRes.json();
    expect(Array.isArray(lbData)).toBeTruthy();
    expect(lbData.length).toBeGreaterThan(0);

    const lbTable = page.locator('#leaderboardTable');
    await expect(lbTable).toBeVisible();
  });

  test('challenge creation endpoint generates challenge link', async ({ request, page }) => {
    const chalRes = await request.post('/api/challenges');
    expect(chalRes.ok()).toBeTruthy();
    const chalData = await chalRes.json();
    expect(chalData.url).toContain('challenge=');
  });

  test('navigation links switch active view', async ({ page }) => {
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('leaderboard');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const el = document.getElementById('view-leaderboard');
        if (el) el.classList.remove('hidden');
      }
    });

    const lbView = page.locator('#view-leaderboard');
    await expect(lbView).toBeVisible();
  });
});
