import { test, expect } from '@playwright/test';

test.describe('Tier 3: Gameplay & Chessboard Infrastructure', () => {
  test.beforeEach(async ({ page, request }) => {
    await request.post('/auth/dev-login', { form: { name: 'Alice' } });
    await page.goto('/');
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('game');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const g = document.getElementById('view-game');
        if (g) g.classList.remove('hidden');
      }
      if (typeof window.renderBoard === 'function') window.renderBoard();
    });
  });

  test('gameplay view container renders properly', async ({ page }) => {
    const gameView = page.locator('#view-game');
    await expect(gameView).toBeVisible();
  });

  test('chessboard element is present in DOM', async ({ page }) => {
    const board = page.locator('#board, .board');
    await expect(board.first()).toBeVisible();
  });

  test('in-game controls (resign & draw buttons) exist', async ({ page }) => {
    const resignBtn = page.locator('#resignBtn');
    const drawBtn = page.locator('#drawBtn');
    await expect(resignBtn).toBeVisible();
    await expect(drawBtn).toBeVisible();
  });
});
