## 2026-08-03T19:49:16Z

<USER_REQUEST>
You are Worker M3 (Design System & UI Shell Developer) for Flame Chess UI redesign.
Your working directory is /Users/hriday/Documents/Flame Chess/.agents/worker_m3.

MANDATORY INTEGRITY WARNING: DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A Forensic Auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Your task:
1. Read /Users/hriday/Documents/Flame Chess/.agents/explorer_m1_3/mockup_audit.md and /Users/hriday/Documents/Flame Chess/.agents/explorer_m1_2/frontend_analysis.md.
2. Update web/index.html to establish the new design system:
   - Root CSS custom properties for Dark theme ([data-theme="dark"]): dark slate/gold/orange palette matching mockups.
   - Root CSS custom properties for Light theme ([data-theme="light"]): clean warm cream/white/blue/orange palette matching mockups.
   - Smooth theme transition, font definitions (Inter / Plus Jakarta Sans / Outfit / JetBrains Mono fallback).
   - Theme Toggle Switch in top header (☀️/🌙) with localStorage persistence ('flamechess_theme').
3. Modernize top navigation bar header:
   - Brand logo & title ("FlameChess" / "Flame University Chess").
   - View navigation buttons (Play/Lobby, Leaderboard, History).
   - Online player badge (`online: N`).
   - Sound toggle button.
   - User profile dropdown ("Hriday Goyal v").
   - Sun/Moon Theme Toggle switch button.
4. Keep the Go backend code (cmd/, internal/) 100% UNTOUCHED.
5. Verify build/serve with `go test ./...` and ensure web/index.html compiles and loads cleanly.
6. Record progress in /Users/hriday/Documents/Flame Chess/.agents/worker_m3/progress.md and handoff report in /Users/hriday/Documents/Flame Chess/.agents/worker_m3/handoff.md.
7. Send message to parent upon completion with handoff path.
</USER_REQUEST>
