---
name: VaultGuard
colors:
  surface: '#f9f9ff'
  surface-dim: '#d3daef'
  surface-bright: '#f9f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f1f3ff'
  surface-container: '#e9edff'
  surface-container-high: '#e1e8fd'
  surface-container-highest: '#dce2f7'
  on-surface: '#141b2b'
  on-surface-variant: '#434655'
  inverse-surface: '#293040'
  inverse-on-surface: '#edf0ff'
  outline: '#737686'
  outline-variant: '#c3c6d7'
  surface-tint: '#0053db'
  primary: '#004ac6'
  on-primary: '#ffffff'
  primary-container: '#2563eb'
  on-primary-container: '#eeefff'
  inverse-primary: '#b4c5ff'
  secondary: '#006591'
  on-secondary: '#ffffff'
  secondary-container: '#39b8fd'
  on-secondary-container: '#004666'
  tertiary: '#3e3fcc'
  on-tertiary: '#ffffff'
  tertiary-container: '#585be6'
  on-tertiary-container: '#f1eeff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dbe1ff'
  primary-fixed-dim: '#b4c5ff'
  on-primary-fixed: '#00174b'
  on-primary-fixed-variant: '#003ea8'
  secondary-fixed: '#c9e6ff'
  secondary-fixed-dim: '#89ceff'
  on-secondary-fixed: '#001e2f'
  on-secondary-fixed-variant: '#004c6e'
  tertiary-fixed: '#e1e0ff'
  tertiary-fixed-dim: '#c0c1ff'
  on-tertiary-fixed: '#07006c'
  on-tertiary-fixed-variant: '#2f2ebe'
  background: '#f9f9ff'
  on-background: '#141b2b'
  surface-variant: '#dce2f7'
typography:
  title-lg:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.015em
  section-header:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: 0.04em
  body-default:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: 0em
  body-medium:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 20px
    letterSpacing: 0em
  label-primary:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: -0.005em
  caption-muted:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
    letterSpacing: 0em
  caption-medium:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.01em
  code-sm:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0em
  code-default:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: -0.01em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  margin: 1.5rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 0.75rem
  space-lg: 1rem
  space-xl: 1.5rem
---

## Brand & Style

This design system establishes an ultra-precise, high-density operational cockpit engineered for IT system administrators, site reliability engineers, and enterprise infrastructure teams. The core design language is Corporate Modern with technical utilitarian rigor—prioritizing absolute clarity, high information throughput, and unambiguous visual state communication over decorative embellishment.

Every view communicates stability, promptness, and technical integrity. Visual weight is strictly functional: structural framing remains quiet and architectural, while status-driven accents draw instantaneous attention to anomalies, system health, and backup fidelity. The emotional tone is calm under pressure, rigorously organized, and hyper-reliable.

## Colors

The palette leverages a crisp, neutral foundation with an explicit semantic hierarchy calibrated for mission-critical IT monitoring.

### Neutral Foundation
- **Canvas Background**: `#F7F8FA` — Light, cool-tinted gray providing comfortable contrast against pure white cards.
- **Surface**: `#FFFFFF` — Primary application plane for cards, panels, floating rails, and modal overlays.
- **Borders & Dividers**: `#E5E7EB` — Crisp structural boundaries. Hover and interactive borders shift to `#D1D5DB` or active `#2563EB`.
- **Subtle Fill / Hover Surface**: `#F3F4F6` — Used for table row hover states, secondary button backgrounds, and code snippet backdrops.
- **Text Primary**: `#111827` — High-legibility slate charcoal for headlines, primary values, and table data.
- **Text Secondary**: `#6B7280` — Mid-tone gray for column headers, metadata, and field descriptions.
- **Text Muted**: `#9CA3AF` — Subdued slate for inactive items, placeholders, and timestamps.

### Brand & Interaction Colors
- **Primary Accent**: `#2563EB` — Action blue for primary CTAs, active navigation items, selected radio indicators, and active switch toggles.
- **Secondary Accent**: `#0EA5E9` — Sky blue reserved for active sync states, linear progress indicators, and running data transfers.
- **Info Accent**: `#6366F1` — Indigo for configuration guidance, token highlights, and staging environment indicators. Background tint: `#EEF2FF`.

### Status & System Health
- **Success / Healthy**: `#16A34A` text and glyphs with `#DCFCE7` badge container. Denotes verified snapshots, operational agents, and green SLAs.
- **Warning / Attention**: `#D97706` text and glyphs with `#FEF3C7` badge container. Flags high memory usage, pending retention trims, or non-fatal checksum drift.
- **Danger / Failed**: `#DC2626` text and glyphs with `#FEE2E2` badge container. Designates failed jobs, disconnected cluster nodes, and SLA breaches.

## Typography

Typography balances administrative utility and visual density. The primary family is **Inter**, configured with tight tracking across numerical readouts and table columns for maximum glanceability.

- **Title Large** (`20px`, Semi-Bold, -0.015em tracking): Primary page titles and command drawer headers.
- **Section Headers** (`14px`, Semi-Bold, 0.04em tracking, Uppercase): Module categorization, column groups, and metric card titles.
- **Body & Data Rows** (`14px`, Regular / Medium): Standard log records, configuration parameters, and primary table cells.
- **Form & Status Labels** (`13px`, Medium): Form input labels, context chip descriptors, and inline controls.
- **Muted Supporting Copy** (`12px`, Regular): Timestamps, relative sync dates, checksum hashes, and secondary telemetry metadata.
- **Monospace Telemetry** (`12px` & `13px`, **JetBrains Mono**): Used strictly for immutable paths (`/mnt/vol01/backup`), SHA-256 signatures, cron strings (`0 2 * * *`), IP addresses, and command-line API tokens.

## Layout & Spacing

The layout is built for high information density on wide multi-monitor IT desks while providing a resilient two-column dashboard structure.

### Structure & Grids
- **Canvas Margins**: Fixed `1.5rem` (`24px`) padding on the main viewport interior.
- **Command Center Two-Column Split**:
  - Primary Operational Column (`70%` width or `8 cols` of a 12-col grid): Hosts the top-level System Health strip, the 4-up backup metric cards, and the interactive Recent Runs table.
  - Secondary Stream Column (`30%` width or `4 cols`): Houses the real-time node activity stream, storage cluster breakdown, and pending snapshot queue.
- **Component Padding Scale**:
  - `space-xs` (`4px`): Micro badge padding, status dot gaps, inline metadata icon gaps.
  - `space-sm` (`8px`): Table cell vertical padding, compact button padding, segmented pill gaps.
  - `space-md` (`12px`): Input field internal padding, dropdown menu item gaps.
  - `space-lg` (`16px`): Standard card body padding, metric tile interior padding.
  - `space-xl` (`24px`): Panel header padding, modal dialog framing.

### Viewport Breakpoints
- **Desktop (1280px+)**: Two-column layout with pinned floating left dock and contextual top-right user pill.
- **Tablet (768px - 1279px)**: Activity feed collapses into a sequential stacked panel beneath the primary table. Dock collapses to an icon-only edge anchor.
- **Mobile (< 768px)**: Dock transitions to a fixed bottom command pill; tables shift into card-based summary rows.

## Elevation & Depth

Visual hierarchy uses flat, architectural separation reinforced with precision hairline borders and whisper-level shadows. Avoid heavy blurs or deep drops.

- **Level 0 (Canvas Base)**: `#F7F8FA` flat base plane.
- **Level 1 (Card & Module Surface)**: Pure white `#FFFFFF` surface with `1px solid #E5E7EB` hairline border and ambient grounding shadow: `0 1px 3px 0 rgba(0, 0, 0, 0.05)`.
- **Level 2 (Floating Edge Elements & Dropdowns)**: Left dock navigation, top-right context chip, and context popovers use `#FFFFFF` with `1px solid #E5E7EB` and an elevated shadow: `0 4px 6px -1px rgba(0, 0, 0, 0.07), 0 2px 4px -2px rgba(0, 0, 0, 0.05)`.
- **Level 3 (Modal Sheets & Confirmation Trays)**: Centered overlay sheets with `0 20px 25px -5px rgba(0, 0, 0, 0.08), 0 8px 10px -6px rgba(0, 0, 0, 0.04)` over a `#111827` backdrop blurred with `backdrop-filter: blur(2px); opacity: 0.3`.

## Shapes

The design system employs a hybrid geometric strategy: **tight structural corners (Soft, 4px - 6px)** for analytical containers and cards to preserve grid rigidity, juxtaposed against **infinite pill radiuses (999px)** for interactive triggers, status indicators, and floating context controls.

- **Cards, Panels, Inputs, Tables**: `4px` to `6px` radius (`roundedness: 1`), keeping grid lines sharp and maximizing screen real estate for tabular metrics.
- **Floating Dock Navigation**: `999px` fully rounded pill container with nested rounded pill icon triggers.
- **Status Badges & Chips**: `999px` pill radius to distinguish runtime state tokens from rectangular content wrappers.
- **Action Buttons**: `6px` radius for primary/secondary actions; circular `999px` for icon-only utility toggles.

## Components

### Floating Edge Dock Navigation
- **Placement**: Fixed vertical floating pill anchored to the left viewport edge with `16px` offset.
- **Style**: Pure white `#FFFFFF` background, `1px solid #E5E7EB`, `0 4px 12px rgba(0,0,0,0.06)` shadow.
- **Items**: 40px × 40px icon containers with `8px` corner radius. Inactive icons are `#6B7280` on transparent; active icons are `#2563EB` backed by `#EFF6FF` soft fill with a 3px vertical indicator strip.

### Floating Context Chip
- **Placement**: Pinned to the top-right corner of the viewport (`16px` margin), decoupled from card grids.
- **Structure**: Pill-shaped container (`height: 36px`), `#FFFFFF`, `1px solid #E5E7EB`, `0 2px 4px rgba(0,0,0,0.04)`.
- **Content**: Left: 24px circular avatar or organization glyph. Center: 13px medium Org Name (`#111827`). Right: downward chevron (`#6B7280`). Interactive hover shifts border to `#D1D5DB`.

### Status Badges
- **Dimensions**: Explicit `height: 22px`, `padding: 0 8px`, `border-radius: 999px`.
- **Typography**: `12px`, font weight `500` (Medium).
- **Variants**:
  - *Healthy*: `#DCFCE7` background, `#16A34A` text, leading 6px pulsing dot `#16A34A`.
  - *Warning*: `#FEF3C7` background, `#D97706` text, solid 6px dot `#D97706`.
  - *Failed*: `#FEE2E2` background, `#DC2626` text, exclamation glyph or solid 6px dot `#DC2626`.
  - *Info/Queued*: `#EEF2FF` background, `#6366F1` text.

### Operational Metric Cards & Panels
- **Structure**: `#FFFFFF` background, `1px solid #E5E7EB`, `border-radius: 6px`, `box-shadow: 0 1px 3px rgba(0,0,0,0.05)`, padding `16px`.
- **Header**: `14px` Semi-Bold uppercase section header (`#6B7280`) paired with top-right micro-action or sparkline.
- **Metrics**: `28px` bold numeric primary text (`#111827`) with supporting `12px` caption muted delta indicator (`+2.4% vs last cycle`).

### Data Tables (Recent Runs & Backups)
- **Header**: `#F9FAFB` surface, `height: 36px`, borders top and bottom `1px solid #E5E7EB`, text `12px` medium uppercase `#6B7280`.
- **Rows**: `#FFFFFF` surface, alternating transition to `#F9FAFB` on hover. `height: 48px`, border bottom `1px solid #E5E7EB`.
- **Cells**: Job target name in `14px` medium (`#111827`), execution path or snapshot ID in `12px` monospaced **JetBrains Mono** (`#6B7280`), right-aligned duration and status badge.

### Form Inputs & Filters
- **Input Fields**: Height `36px`, padding `0 12px`, `border: 1px solid #E5E7EB`, `border-radius: 6px`, text `14px` Regular. Focus state: `border-color: #2563EB; box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.15)`.
- **Checkboxes**: `16px × 16px`, `border-radius: 4px`, `border: 1px solid #D1D5DB`. Checked state: background `#2563EB` with white checkmark glyph.

### Buttons
- **Primary**: Background `#2563EB`, text `#FFFFFF`, height `36px`, padding `0 14px`, `border-radius: 6px`, font `13px` medium. Hover: `#1D4ED8`. Active: `#1E40AF`.
- **Secondary**: Background `#FFFFFF`, text `#111827`, border `1px solid #E5E7EB`. Hover: `#F3F4F6`, border `#D1D5DB`.
- **Destructive**: Background `#FFFFFF`, text `#DC2626`, border `1px solid #FCA5A5`. Hover: `#FEE2E2`.