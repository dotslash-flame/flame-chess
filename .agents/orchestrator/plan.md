# Plan: Flame Chess UI Redesign & Verification

## Objective
Implement a modern, high-fidelity UI for Flame Chess supporting both Light and Dark modes with a theme toggle based on `/Users/hriday/Downloads/stitch 2` mockups, without altering the Go backend. Provide Playwright verification scripts, automated tests, and mockup screenshot comparisons.

## Milestones Overview

### Milestone 1: Exploration & Codebase Analysis (In Progress)
- Dispatch 3 `teamwork_preview_explorer` agents in parallel:
  - Explorer 1: Inspect Go backend architecture, HTTP & WebSocket endpoints (`cmd`, `internal`, `API.md`, `ARCHITECTURE.md`).
  - Explorer 2: Inspect existing frontend (`web/index.html`, assets, static serving mechanism).
  - Explorer 3: Audit design mockups in `/Users/hriday/Downloads/stitch 2` (catalog components, light/dark mode design details, color tokens, page layouts).

### Milestone 2: Test Infrastructure & Baseline Verification Setup
- Build Playwright automated test suite structure for visual screenshot comparison and functional testing (Tiers 1-4).
- Setup background local dev server runner and verify test environment readiness.

### Milestone 3: Design System & Core Shell (Theme Toggle & Layout)
- Establish CSS custom properties / styling tokens for Light and Dark themes matching design mockups.
- Build theme toggle persistence and global layout shell (Navbar, Sidebar, Theme Toggle).

### Milestone 4: Login & Authentication Pages
- Implement modern Login / Auth screens matching mockup specifications (Light & Dark variants).
- Preserve Go backend auth API integration (`/auth/...`, session cookies).

### Milestone 5: Dashboard & Matchmaking Lobby
- Implement modern Dashboard, Quick Match controls, Active Games list, Leaderboards/Ratings.
- Integrate existing WebSocket hub & REST endpoints.

### Milestone 6: Gameplay & Interactive Chessboard
- Implement modern Game Room page with interactive chessboard, timers, move history, player profiles, turn indicators.
- Wire to WebSocket game actor.

### Milestone 7: Game Over Modal, Chat & Spectating Panels
- Implement Game Over modal (Victory / Defeat / Draw) per design mockups.
- Implement in-game chat and spectator UI.

### Milestone 8: End-to-End Verification, Playwright Screenshots & Audit
- Launch local dev server.
- Execute Playwright screenshot scripts in both Light and Dark modes.
- Run Forensic Auditor and Reviewers to verify mockup match, functionality, and backend immutability.
- Claim victory.
