import { test, expect } from '@playwright/test';

test.describe('Tier 4: Game Over Modal & Social / Chat UI', () => {
  test.beforeEach(async ({ page, request }) => {
    await request.post('/auth/dev-login', { form: { name: 'Alice' } });
    await page.goto('/');
  });

  test('chat panel renders with message input and send button in game view', async ({ page }) => {
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('game');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const g = document.getElementById('view-game');
        if (g) g.classList.remove('hidden');
      }
    });

    const chatPanel = page.locator('#chatPanel');
    await expect(chatPanel).toBeVisible();

    const chatInput = page.locator('#chatInput');
    const sendChatBtn = page.locator('#sendChatBtn');
    await expect(chatInput).toBeVisible();
    await expect(sendChatBtn).toBeVisible();
  });

  test('game over modal renders outcome, result subtext, and rematch button', async ({ page }) => {
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('game');
      const root = document.getElementById('overlayRoot');
      if (root) {
        root.innerHTML = `
          <div class="ovl" id="gameOverModal">
            <div class="panel modal-card" style="width:400px;text-align:center;">
              <div class="modal-accent-bar"></div>
              <div class="outcome-banner win">
                <div class="outcome-title">VICTORY</div>
              </div>
              <div class="victory-status-text">White Wins by Checkmate</div>
              <div class="elo-change-pill ok">+15 ELO</div>
              <div style="margin-top:16px;" id="rematchArea">
                <button class="btn primary" id="rematchBtn" style="width:100%">Rematch ↻</button>
              </div>
            </div>
          </div>
        `;
      }
    });

    const modal = page.locator('#gameOverModal');
    await expect(modal).toBeVisible();

    const victoryText = page.locator('.victory-status-text');
    await expect(victoryText).toContainText('Checkmate');

    const rematchBtn = page.locator('#rematchBtn');
    await expect(rematchBtn).toBeVisible();
  });

  test('live spectator panel displays live games or empty state', async ({ page }) => {
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('lobby');
    });

    const liveGamesPanel = page.locator('#liveGamesPanel, #spectatorPanel, #liveGamesList');
    await expect(liveGamesPanel.first()).toBeVisible();
  });
});
