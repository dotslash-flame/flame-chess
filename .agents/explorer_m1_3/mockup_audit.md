# Flame Chess UI Design System & Mockup Audit

**Auditor:** Explorer 3 (Mockup & UI Design System Audit Specialist)  
**Target Folder:** `/Users/hriday/Downloads/stitch 2`  
**Date:** August 4, 2026  

---

## Executive Summary

A comprehensive audit of all **17 design mockups** in `/Users/hriday/Downloads/stitch 2` was conducted. The mockups cover four core screen categories of the Flame Chess application: **Login Screens**, **Dashboard / Lobby**, **Gameplay Screen**, and **Game Over Modal**.

The design system demonstrates a cohesive, modern UI architecture with distinct dark and light mode implementations, as well as three subtle aesthetic direction variants (Warm Gold/Amber Dark, Warm Cream/Bronze Light, and Vibrant Flame Orange). This report details the complete visual component inventory, page breakdowns, theme switch mechanics, typography rules, component specs, and a full CSS design token system for implementation.

---

## 1. Catalog & Analysis of All 17 Mockups

| # | Directory Name | Key Screen | Theme / Palette | Distinct Features & Branding |
|---|---|---|---|---|
| 1 | `flamechess_login_dark_mode_1` | Login | Dark Charcoal + Warm Gold | Top header navbar with Flame icon logo, `online: 1`, audio toggle icon, user profile dropdown. Centered dark card with flame icon, sans-serif white title, Google login button. |
| 2 | `flamechess_login_dark_mode_2` | Login | Dark Charcoal + Gold Outline | Top header navbar with gold serif text title ("FlameChess"), profile dropdown. Card has thin gold border outline + serif gold title. |
| 3 | `flamechess_login_light_mode_1` | Login | Warm Cream (`#FAF8F5`) | Clean login view without header navbar. White card with soft drop shadow, orange flame icon + dark sans-serif title, Google button, "Sign up" link. |
| 4 | `flamechess_login_light_mode_2` | Login | Warm Cream (`#FAF8F5`) | Clean login view. White card featuring bold dark serif title ("FlameChess"), Google button, "Sign up" link. |
| 5 | `flamechess_login_light_mode_3` | Login | Warm Cream (`#FAF8F5`) | Clean login view. White card featuring a line-art Knight chess piece icon above dark serif title. |
| 6 | `flamechess_dashboard_dark_mode_1` | Dashboard / Lobby | Dark Charcoal + Warm Gold | 2-column layout. Left: Play card grid (Bullet 1+0, Blitz 3+2, Blitz 5+0 active with gold border, Rapid 10+0), custom time inputs, gold "Find match" CTA, "Play a friend" link. Right: 5 stacked cards (Your Blitz rating 787, Online challenge, Top Blitz leaderboard, Watch live, Recent games). |
| 7 | `flamechess_dashboard_dark_mode_2` | Dashboard / Lobby | Dark Charcoal + Gold | Identical to dark_mode_1, but navbar uses Flame University crest emblem (open book inside shield) + "Flame University Chess" header title. |
| 8 | `flamechess_dashboard_light_mode_1` | Dashboard / Lobby | Warm Cream + Bronze/Brown | Light mode lobby. White cards with drop shadows. Active nav item has gold bottom underline indicator. Header includes explicit Sun/Moon Theme Toggle `(☀️/🌙)`. Active time card in bronze brown (`#8C6239`). Win/Loss indicators in green (`W+20`) and red (`L-18`). |
| 9 | `flamechess_dashboard_light_mode_2` | Dashboard / Lobby | Warm Cream + Bronze/Brown | Light mode lobby. Identical layout to light_mode_1, but navbar uses circular seal monogram ("FC" inside laurel wreath) + serif text "FlameChess". Includes Theme Toggle switch. |
| 10 | `flamechess_gameplay_dark_mode` | Gameplay | Dark Charcoal + Slate Blue/Gray Board | Central 8x8 chessboard with Slate Blue/Gray squares (`#4C6063` / `#91A398`). Player info top & bottom with glowing gold timer (`4:41`). Right panel: MOVES card (move table, navigation controls `|< < > >| live`, `Resign` & `Offer draw` buttons) and CHAT card (message list & input). |
| 11 | `flamechess_gameplay_light_mode` | Gameplay | Warm Cream + Classic Green Board | Classic Green & Cream chessboard (`#769656` / `#E8EDD0`) with external rank/file coordinates. Player info cards with dark digital timers (`5:00` / `4:41`). Right panel: MOVES card with explicit `1px #E5E7EB` border annotation, move list, tan navigation buttons, bronze `Offer draw` CTA (`#8C6239`). CHAT card. |
| 12 | `game_over_modal_dark_mode` | Game Over Modal | Dark Backdrop + Gold Accent | Dimmed dark gameplay backdrop. Centered dark modal with gold top border line, King icon, "You lost" title, "vs Dot Slash · resign" subtext, primary gold "Rematch ↻" CTA, secondary row ("New game ▶", "⚡ Review", "Back to lobby"). |
| 13 | `game_over_modal_light_mode` | Game Over Modal | Light Backdrop + Wooden Frame | Dimmed light gameplay backdrop with wooden chessboard frame (`#E6CEAC`). White modal card with close `X` icon, bold serif "You lost" title, dark navy "Rematch ↻" CTA (`#0F172A`), secondary buttons ("New game ▶", "Review"). Tabbed sidebar panel (Moves / Chat tabs). |
| 14 | `image.png_1` | Dashboard / Lobby | Dark Charcoal + Flame Orange | Dark mode dashboard variation with vibrant Flame Orange theme (`#FF9500`). Active nav pill. Selected time control card with glowing orange border. Orange "Find match ▶" CTA and orange rating numbers (`787`, `819`, `812`). |
| 15 | `image.png_2` | Login | Dark Charcoal + Flame Orange | Dark mode login with Flame Orange theme. Card features 3D Fire emoji (`🔥`), bold white title, and solid glowing orange Google login button (`#FF8C38`). |
| 16 | `image.png_3` | Gameplay | Dark Charcoal + Flame Orange | Dark mode gameplay with Flame Orange timer badge (`#FF9500`) and Classic Green & Cream board (`#70A260` / `#E8EDD0`). Dark right panel cards (`#22201E`). |
| 17 | `image.png_4` | Game Over Modal | Dark Backdrop + Flame Orange | Dark mode Game Over modal with Flame Orange theme (`#FF8C38`). Top orange border glow line. Primary "Rematch ↻" and secondary "New game ▶" in solid bright orange fill. |

---

## 2. Theme Requirements & Visual Aesthetics

### Aesthetic Directions Discovered
1. **Warm Gold / Amber Dark Mode (Classic FlameChess)**
   - Dark charcoal background (`#0F0F10` / `#121214`) with warm gold/amber accents (`#D4AF37` / `#C5A059`).
   - Card backgrounds: `#1A1A1E` with subtle 1px gold borders or muted borders (`#2D2D32`).
   - Timers & CTAs: Warm gold fill (`#D5A754`) with dark text.

2. **Warm Cream / Bronze-Brown Light Mode (Warm Minimalist)**
   - Soft off-white / warm cream background (`#FAF8F5` / `#F7F5F0`).
   - Cards: Crisp white (`#FFFFFF`) with soft drop shadows (`0 4px 12px rgba(0,0,0,0.05)`).
   - Accents & Primary CTAs: Rich bronze brown (`#8C6239` / `#966D3A`) and dark navy (`#0F172A`).
   - Chessboard options: Classic Green/Cream (`#769656` / `#E8EDD0`) or Light Wood frame (`#E6CEAC`).

3. **High-Energy Flame Orange (Vibrant Theme)**
   - Deep charcoal/black (`#161514` / `#1C1A18`).
   - Accents: Vibrant flame orange (`#FF9500` / `#FF8C38` / `#FF7A00`).
   - Timers, highlights, and primary CTA buttons glow in solid flame orange.

### Dark Mode vs. Light Mode Structural Differences
- **Header Navbar:**
  - Dark Mode: Integrated dark bar (`#121214`), light text links, gold/orange flame icon or university crest logo.
  - Light Mode: Cream navbar (`#FAF8F5`), dark text links, active tab indicated by a gold/bronze bottom underline indicator or pill background. Top-right includes a dedicated **Sun/Moon Theme Toggle switch `(☀️/🌙)`**.
- **Cards & Surfaces:**
  - Dark Mode: Dark gray container cards (`#1A1A1E` / `#22201E`) with faint 1px borders (`#2E2E34`).
  - Light Mode: White cards (`#FFFFFF`) with subtle 1px border (`#E5E7EB`) and soft elevated shadows.
- **Chessboard Presentation:**
  - Dark Mode: Slate Blue/Gray squares (`#4C6063` / `#91A398`) or Green/Cream squares embedded flush in a dark container with internal coordinate labels.
  - Light Mode: Green/Cream squares with external rank/file labels (1-8 left, a-h bottom) or optional natural wooden border surround (`#E6CEAC`).

### Theme Toggle Placement & Mechanics
- **Placement:** Top-right corner of the main navigation header bar, positioned immediately to the right of the User Profile pill/avatar menu.
- **Visual Design:** Toggle switch component featuring a Sun icon `☀️` on the left and a Moon icon `🌙` on the right with a rounded sliding knob.
- **Behavior:** Toggling seamlessly switches all CSS design token variables between Dark Mode values and Light Mode values.

---

## 3. Typography System

The design system relies on a clean, accessible typographic hierarchy combining modern sans-serif fonts for UI precision with an optional serif font for brand elegance:

1. **Primary Sans-Serif Font (UI & Core Content)**
   - *Font Family:* `Inter`, `SF Pro Display`, `-apple-system`, `sans-serif`.
   - *Usage:* Dashboard titles, buttons, card content, move list, chat, form inputs, ratings.
   - *Weights:* Medium (500), SemiBold (600), Bold (700).

2. **Secondary Serif Font (Branding & Titles)**
   - *Font Family:* `Playfair Display`, `Cormorant Garamond`, `Georgia`, `serif`.
   - *Usage:* Elegant logo variants (`FlameChess`), university headers ("Flame University Chess"), and Game Over modal headings ("You lost").
   - *Weights:* SemiBold (600), Bold (700).

3. **Digital / Monospace Font (Timers & Coordinates)**
   - *Font Family:* `JetBrains Mono`, `Roboto Mono`, `tabular-nums`.
   - *Usage:* Game clocks (`5:00`, `4:41`), move numbers, rating statistics (`787`).
   - *Weights:* Bold (700) with fixed numeric widths for clock stability.

---

## 4. Component Inventory

### A. Navigation & Header
- **Logo Options:**
  - *Option A (Default):* Orange/Gold Flame Icon + Sans-Serif "FlameChess".
  - *Option B (Serif Elegant):* Gold Serif "FlameChess" text.
  - *Option C (University Crest):* Shield/Book Emblem + "Flame University Chess".
  - *Option D (Laurel Monogram):* Circular "FC" Laurel Wreath Seal + Serif "FlameChess".
- **Nav Links:** "Play" (active state indicator), "Leaderboard", "History".
- **Status & Controls:** Online count badge (`online: 1` / `online: 2`), Audio toggle icon, User Profile dropdown menu (avatar + name), Theme Toggle switch `(☀️/🌙)`.

### B. Login Screen Component
- **Card Container:** Centered card (width: ~420px, rounded corners: 16px).
- **Branding Header:** Choice of Flame icon, Gold serif title, Knight line art, or 3D Fire emoji (`🔥`).
- **Subtitle:** "Play rated chess with Flame University."
- **Primary CTA:** "Login with Google" button with Google 'G' logo.
- **Footer:** "Don't have an account? Sign up." link.

### C. Dashboard / Lobby Components
- **Time Control Cards Grid:** 4 preset cards (Bullet 1+0, Blitz 3+2, Blitz 5+0, Rapid 10+0) with icons, time control text, and user rating. Selected state highlights card with active border and tinted background.
- **Custom Time Controls:** Label "custom" with `base` and `inc` number input fields.
- **Matchmaking CTAs:** Primary "Find match ▶" button (solid filled) and Secondary "Play a friend (link)" button (bordered outline).
- **Sidebar Panels:**
  - *Your Rating Card:* Displays current Blitz rating (`787`) and total games played.
  - *Online Challenge:* Status of direct challenge lobby.
  - *Top Blitz Leaderboard:* Ranked list (1-5) of top players and ratings.
  - *Watch Live Games:* List of ongoing featured matches.
  - *Recent Games:* Match history with outcome tag (`W+20` in green, `L-18` in red/maroon).

### D. Gameplay Screen Components
- **Chessboard:** 8x8 grid with rank (1-8) and file (a-h) coordinates. Piece set styled in high-contrast vector outlines.
- **Player Info Bars:** Top (Opponent) and Bottom (User) bars showing avatar, player name, rating/format badge (`787 BLITZ`), and digital countdown timer.
- **Active Timer Display:** Highlighted countdown box with glowing gold or vibrant orange accent when active.

### E. Side Panels (Moves & Chat)
- **Moves Panel:**
  - Table showing move number, White move, Black move, and current move highlight.
  - Step navigation toolbar: `|<` (first), `<` (prev), `>` (next), `>|` (last), `live` indicator badge.
  - In-game action buttons: `Resign` button (warning/outline style) and `Offer draw` button.
  - Optional Tabbed layout in light mode: "Moves" tab and "Chat" tab in a single card.
- **Chat Panel:**
  - Header: "CHAT".
  - Scrollable message stream with username and rating badges.
  - Text input field ("Say something...") and `Send` button.

### F. Game Over Modal Component
- **Backdrop:** Dimmed and blurred gameplay screen.
- **Modal Box:** Centered dialog container with rounded corners (16px) and optional top glowing accent line.
- **Result Header:** Outcome icon (King piece symbol), result title ("You lost" / "You won" / "Draw"), and subtext detail ("vs Dot Slash · resign").
- **Action Buttons:**
  - Primary CTA: Full-width "Rematch ↻" button (solid fill).
  - Secondary Row: "New game ▶", "⚡ Review", and "Back to lobby".

---

## 5. CSS Design Tokens Specification

The following CSS variables map all design tokens required to support Dark Mode, Light Mode, and the Vibrant Orange theme variant seamlessly:

```css
/* ==========================================================================
   Flame Chess UI Design System - CSS Design Tokens
   ========================================================================== */

:root {
  /* Common Spacing & Radii */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
  --radius-full: 9999px;
  
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-serif: 'Playfair Display', 'Cormorant Garamond', Georgia, serif;
  --font-mono: 'JetBrains Mono', 'Roboto Mono', monospace;

  /* Outcome Colors */
  --color-win: #22c55e;
  --color-win-bg: rgba(34, 197, 94, 0.1);
  --color-loss: #ef4444;
  --color-loss-bg: rgba(239, 68, 68, 0.1);
  --color-draw: #94a3b8;
}

/* --------------------------------------------------------------------------
   1. Dark Mode Tokens (Warm Gold Accent) - Default Theme
   -------------------------------------------------------------------------- */
[data-theme="dark"] {
  /* Backgrounds & Surfaces */
  --bg-app: #0f0f10;
  --bg-surface: #1a1a1e;
  --bg-surface-hover: #242429;
  --bg-surface-active: #2c2c33;
  --bg-card: #161619;
  --bg-modal-backdrop: rgba(0, 0, 0, 0.75);

  /* Typography Colors */
  --text-primary: #f3f4f6;
  --text-secondary: #9ca3af;
  --text-muted: #6b7280;
  --text-inverse: #0f0f10;

  /* Borders & Dividers */
  --border-color: #2b2b32;
  --border-color-hover: #3f3f48;
  --border-focus: #d4af37;

  /* Brand & Accents (Warm Gold) */
  --accent-primary: #d5a754;
  --accent-primary-hover: #e2b866;
  --accent-primary-text: #121214;
  --accent-glow: rgba(213, 167, 84, 0.25);
  --accent-border: #8c6d33;

  /* Chessboard Tokens (Slate Blue/Gray Theme) */
  --board-square-light: #91a398;
  --board-square-dark: #4c6063;
  --board-square-highlight: rgba(213, 167, 84, 0.5);
  --board-coords-text: #6b7280;

  /* Timers */
  --timer-bg-active: #d5a754;
  --timer-text-active: #121214;
  --timer-bg-inactive: #222026;
  --timer-text-inactive: #9ca3af;
}

/* --------------------------------------------------------------------------
   2. Light Mode Tokens (Warm Cream & Bronze Theme)
   -------------------------------------------------------------------------- */
[data-theme="light"] {
  /* Backgrounds & Surfaces */
  --bg-app: #faf8f5;
  --bg-surface: #ffffff;
  --bg-surface-hover: #f3efe8;
  --bg-surface-active: #e9e3d8;
  --bg-card: #ffffff;
  --bg-modal-backdrop: rgba(15, 23, 42, 0.4);

  /* Typography Colors */
  --text-primary: #1c1917;
  --text-secondary: #57534e;
  --text-muted: #a8a29e;
  --text-inverse: #ffffff;

  /* Borders & Dividers */
  --border-color: #e7e5e4;
  --border-color-hover: #d6d3d1;
  --border-focus: #8c6239;

  /* Brand & Accents (Bronze Brown / Dark Navy) */
  --accent-primary: #8c6239;
  --accent-primary-hover: #734f2d;
  --accent-primary-text: #ffffff;
  --accent-glow: rgba(140, 98, 57, 0.2);
  --accent-border: #b08252;

  /* Chessboard Tokens (Classic Green & Cream Theme) */
  --board-square-light: #e8edd0;
  --board-square-dark: #769656;
  --board-square-highlight: rgba(245, 246, 130, 0.8);
  --board-coords-text: #78716c;
  --board-frame-wood: #e6ceac;

  /* Timers */
  --timer-bg-active: #0f172a;
  --timer-text-active: #ffffff;
  --timer-bg-inactive: #f1f5f9;
  --timer-text-inactive: #475569;
}

/* --------------------------------------------------------------------------
   3. Flame Orange Variant Tokens (High Energy Theme)
   -------------------------------------------------------------------------- */
[data-theme="dark-orange"] {
  --bg-app: #161514;
  --bg-surface: #22201e;
  --bg-surface-hover: #2d2a27;
  --bg-card: #1c1a18;

  --text-primary: #f9fafb;
  --text-secondary: #9ca3af;
  
  --border-color: #332f2b;

  --accent-primary: #ff9500;
  --accent-primary-hover: #ffaa33;
  --accent-primary-text: #161514;
  --accent-glow: rgba(255, 149, 0, 0.3);
  --accent-border: #ff8c38;

  --board-square-light: #e8edd0;
  --board-square-dark: #70a260;
  --board-square-highlight: rgba(255, 149, 0, 0.6);

  --timer-bg-active: #ff9500;
  --timer-text-active: #161514;
}
```

---

## 6. Recommendations for Implementation

1. **Standardize Navigation Header:** Adopt a unified top navbar layout containing logo, page links ("Play", "Leaderboard", "History"), online user counter badge, sound toggle button, user profile dropdown, and the explicit **Sun/Moon Theme Toggle Switch `(☀️/🌙)`**.
2. **Consolidate Logo Branding:** Use Option A (Flame Icon + Sans-Serif "FlameChess") as the primary web application logo, while offering Option C (Flame University Crest) for official university events or tournament views.
3. **Implement Design Tokens via CSS Variables:** Use the CSS variables cataloged above in `index.css` or Tailwind CSS config to enable seamless switching between dark mode, light mode, and orange accent mode.
4. **Cohesive Chessboard Styling:** Implement toggleable chessboard themes in settings (Slate Blue/Gray vs Classic Green/Cream vs Wooden Board Frame).
