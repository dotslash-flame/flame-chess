import { chromium } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import http from 'http';
import { spawn } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SCREENSHOT_DIR = path.resolve(__dirname, 'screenshots');
const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

// Ensure screenshots output directory exists
if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

// Function to check if local server is listening
function isServerRunning(url) {
  return new Promise((resolve) => {
    const req = http.get(`${url}/healthz`, (res) => {
      resolve(res.statusCode === 200);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(1000, () => {
      req.destroy();
      resolve(false);
    });
  });
}

// Function to start server if not running
async function ensureServerRunning() {
  const running = await isServerRunning(BASE_URL);
  if (running) {
    console.log(`[Harness] Server is already running on ${BASE_URL}`);
    return null;
  }

  console.log(`[Harness] Starting local dev server...`);
  const serverProc = spawn('node', [path.resolve(__dirname, 'dev-server.js')], {
    stdio: 'inherit',
    cwd: __dirname
  });

  // Poll until ready
  for (let i = 0; i < 30; i++) {
    await new Promise(r => setTimeout(r, 200));
    if (await isServerRunning(BASE_URL)) {
      console.log(`[Harness] Local dev server ready on ${BASE_URL}`);
      return serverProc;
    }
  }

  throw new Error(`[Harness] Dev server failed to start on ${BASE_URL}`);
}

async function captureScreenshots() {
  let serverProc = null;
  try {
    serverProc = await ensureServerRunning();

    console.log('[Harness] Launching browser...');
    const browser = await chromium.launch({ headless: true });
    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
      deviceScaleFactor: 2
    });

    const page = await context.newPage();

    console.log(`[Harness] Navigating to ${BASE_URL}...`);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(500);

    // Helper to toggle theme cleanly
    const setTheme = async (theme) => {
      await page.evaluate((t) => {
        if (typeof window.applyTheme === 'function') {
          window.applyTheme(t);
        } else {
          document.documentElement.setAttribute('data-theme', t);
          try { localStorage.setItem('flamechess_theme', t); } catch(_) {}
        }
      }, theme);
      await page.waitForTimeout(300);
    };

    // --- 1. Login / Landing View Screenshots ---
    console.log('[Harness] Capturing Login view screenshots...');
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('landing');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const l = document.getElementById('view-landing');
        if (l) l.classList.remove('hidden');
      }
    });
    await page.waitForTimeout(300);

    await setTheme('light');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'login_light.png'), fullPage: true });

    await setTheme('dark');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'login_dark.png'), fullPage: true });

    // --- Perform Dev Login ---
    console.log('[Harness] Performing Dev Login...');
    await page.evaluate(async () => {
      try {
        const resp = await fetch('/auth/dev-login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: 'name=Alice'
        });
        const data = await resp.json();
        if (typeof window.api === 'function') {
          window.state = window.state || {};
          window.state.me = data.identity || { display_name: 'Alice', id: 'dev-alice-123' };
        }
      } catch (e) {
        console.error('Dev login error:', e);
      }
    });

    // --- 2. Lobby / Dashboard View Screenshots ---
    console.log('[Harness] Capturing Dashboard / Lobby view screenshots...');
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
    await page.waitForTimeout(300);

    await setTheme('light');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'dashboard_light.png'), fullPage: true });

    await setTheme('dark');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'dashboard_dark.png'), fullPage: true });

    // --- 3. Gameplay View Screenshots ---
    console.log('[Harness] Capturing Gameplay view screenshots...');
    await page.evaluate(() => {
      if (typeof window.showView === 'function') window.showView('game');
      else {
        document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
        const g = document.getElementById('view-game');
        if (g) g.classList.remove('hidden');
      }
      if (typeof window.renderBoard === 'function') window.renderBoard();
    });
    await page.waitForTimeout(300);

    await setTheme('light');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'gameplay_light.png'), fullPage: true });

    await setTheme('dark');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'gameplay_dark.png'), fullPage: true });

    // Component screenshot: Board
    const boardEl = await page.$('#board') || await page.$('.boardWrap') || await page.$('#view-game');
    if (boardEl) {
      await setTheme('light');
      await boardEl.screenshot({ path: path.join(SCREENSHOT_DIR, 'board_light.png') });
      await setTheme('dark');
      await boardEl.screenshot({ path: path.join(SCREENSHOT_DIR, 'board_dark.png') });
    }

    // --- 4. Game Over Modal Screenshots ---
    console.log('[Harness] Capturing Game Over modal screenshots...');
    await page.evaluate(() => {
      const root = document.getElementById('overlayRoot');
      if (root) {
        root.innerHTML = `
          <div class="ovl" id="gameOverModal">
            <div class="panel modal-card" style="width:420px;text-align:center;position:relative;">
              <div class="modal-accent-bar"></div>
              <div class="outcome-banner win">
                <div class="outcome-title">VICTORY</div>
              </div>
              <div class="victory-status-text">White Wins by Checkmate</div>
              <div class="opp-subtext" style="margin-bottom:8px;">Alice (1200) vs Bob (1180)</div>
              <div class="elo-change-pill ok">+15 ELO (1215)</div>
              <div style="margin-top:20px;display:flex;gap:10px;justify-content:center;" id="rematchArea">
                <button class="btn primary" id="rematchBtn" style="flex:1;padding:10px 16px;">Rematch ↻</button>
                <button class="btn" id="closeGameOverBtn" style="flex:1;padding:10px 16px;">Close</button>
              </div>
            </div>
          </div>
        `;
      }
    });
    await page.waitForTimeout(300);

    await setTheme('light');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'game_over_light.png'), fullPage: true });

    await setTheme('dark');
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, 'game_over_dark.png'), fullPage: true });

    await browser.close();
    console.log('[Harness] All screenshots captured successfully in e2e/screenshots/!');
  } catch (err) {
    console.error('[Harness] Screenshot capture error:', err);
    process.exitCode = 1;
  } finally {
    if (serverProc) {
      console.log('[Harness] Stopping local server process...');
      serverProc.kill();
    }
  }
}

captureScreenshots();
