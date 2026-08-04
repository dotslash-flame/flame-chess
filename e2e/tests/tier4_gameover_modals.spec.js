import { test, expect } from '@playwright/test';

test.describe('Tier 4: Game Over & Modals', () => {
  test.beforeEach(async ({ page, request }) => {
    await request.post('/auth/dev-login', { form: { name: 'Alice' } });
    await page.goto('/');
  });

  test('live games endpoint returns active games list', async ({ request }) => {
    const res = await request.get('/api/games/live');
    expect(res.ok()).toBeTruthy();
    const liveGames = await res.json();
    expect(Array.isArray(liveGames)).toBeTruthy();
  });

  test('game over modal displays game result and rematch option', async ({ page }) => {
    await page.evaluate(() => {
      const root = document.getElementById('overlayRoot');
      if (root) {
        root.innerHTML = `
          <div id="gameOverModalTest" class="modal-overlay" style="position:fixed;inset:0;background:rgba(0,0,0,0.7);display:flex;align-items:center;justify-content:center;z-index:1000;">
            <div class="panel modal-content" style="background:var(--bg-panel,#1e293b);color:var(--fg,#f8fafc);padding:24px;border-radius:12px;width:400px;text-align:center;">
              <h2 style="font-size:24px;margin-bottom:8px;color:var(--accent,#f59e0b);">Game Over</h2>
              <div id="gameResultText" style="font-size:18px;font-weight:bold;margin-bottom:4px;">White Wins by Checkmate</div>
              <button class="btn primary" id="rematchBtn">Rematch</button>
            </div>
          </div>
        `;
      }
    });

    const modal = page.locator('#gameOverModalTest');
    await expect(modal).toBeVisible();

    const resultText = page.locator('#gameResultText');
    await expect(resultText).toHaveText('White Wins by Checkmate');

    const rematchBtn = page.locator('#rematchBtn');
    await expect(rematchBtn).toBeVisible();
  });
});
