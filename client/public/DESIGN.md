---
name: Obsidian Telemetry
colors:
  surface: '#0b1326'
  surface-dim: '#0b1326'
  surface-bright: '#31394d'
  surface-container-lowest: '#060e20'
  surface-container-low: '#131b2e'
  surface-container: '#171f33'
  surface-container-high: '#222a3d'
  surface-container-highest: '#2d3449'
  on-surface: '#dae2fd'
  on-surface-variant: '#bbcabf'
  inverse-surface: '#dae2fd'
  inverse-on-surface: '#283044'
  outline: '#86948a'
  outline-variant: '#3c4a42'
  surface-tint: '#4edea3'
  primary: '#4edea3'
  on-primary: '#003824'
  primary-container: '#10b981'
  on-primary-container: '#00422b'
  inverse-primary: '#006c49'
  secondary: '#4cd7f6'
  on-secondary: '#003640'
  secondary-container: '#03b5d3'
  on-secondary-container: '#00424e'
  tertiary: '#ffb95f'
  on-tertiary: '#472a00'
  tertiary-container: '#e29100'
  on-tertiary-container: '#523200'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#6ffbbe'
  primary-fixed-dim: '#4edea3'
  on-primary-fixed: '#002113'
  on-primary-fixed-variant: '#005236'
  secondary-fixed: '#acedff'
  secondary-fixed-dim: '#4cd7f6'
  on-secondary-fixed: '#001f26'
  on-secondary-fixed-variant: '#004e5c'
  tertiary-fixed: '#ffddb8'
  tertiary-fixed-dim: '#ffb95f'
  on-tertiary-fixed: '#2a1700'
  on-tertiary-fixed-variant: '#653e00'
  background: '#0b1326'
  on-background: '#dae2fd'
  surface-variant: '#2d3449'
typography:
  display-lg:
    fontFamily: Geist
    fontSize: 40px
    fontWeight: '600'
    lineHeight: 48px
    letterSpacing: -0.03em
  headline-lg:
    fontFamily: Geist
    fontSize: 30px
    fontWeight: '600'
    lineHeight: 38px
    letterSpacing: -0.025em
  headline-lg-mobile:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Geist
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.02em
  title-sm:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: -0.005em
  body-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: 0em
  label-md:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: -0.01em
  label-sm:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.02em
  code-xs:
    fontFamily: JetBrains Mono
    fontSize: 10px
    fontWeight: '400'
    lineHeight: 14px
    letterSpacing: 0.04em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-lg: 1.5rem
  margin: 1.5rem
  margin-sm: 1rem
  margin-lg: 2.5rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 0.75rem
  space-lg: 1.25rem
  space-xl: 2rem
---

## Brand & Style
The design system establishes a mission-critical, enterprise infrastructure aesthetic designed for Site Reliability Engineers, Platform Architects, and Database Administrators. The interface must project absolute reliability, technical precision, and operational calm under high-severity conditions. 

The aesthetic fuses hyper-dense telemetry displays with a sophisticated, modernist dark architecture. It balances technical utility with meticulous craftsmanship—utilizing obsidian-level dark canvas depths, hair-line micro-borders, translucent floating contextual control docks, and luminous phosphorescent signal accents. Typography delivers razor-sharp legibility across dense data tables, execution logs, and live stream metrics. The overall psychological feel is authoritative, tactical, and unmistakably engineered.

## Colors
The palette is built strictly for dark-mode telemetry and dense system administration. 

- **Primary Canvas & Surfaces (`neutral_color_hex: #0F172A`):** The foundational substrate is a deep obsidian slate. Surfaces ascend through tonal layering: Root canvas (`#050811`), base container surfaces (`#0B1120`), elevated panels/cards (`#0F172A`), and interactive float/overlay layers (`#1E293B`).
- **Primary Accent (`primary_color_hex: #10B981`):** Luminous Emerald serves as the primary system driver, denoting healthy operation, confirmed recovery points, operational nodes, and affirmative interaction states. Highlight tints run from `#34D399` to structural deep emerald `#064E3B`.
- **Secondary Accent (`secondary_color_hex: #06B6D4`):** Electric Cyan indicates active, non-blocking asynchronous orchestrations—live replication streams, snapshot serialization, and dynamic in-flight backup jobs.
- **Tertiary Accent (`tertiary_color_hex: #F59E0B`):** Amber signals latency degradation, imminent quota breaches, retention window expiries, and cautionary snapshot states.
- **Semantic Crimson (`#F43F5E`):** Reserved exclusively for unrecoverable errors, backup failures, replication stall events, and destructive actions.
- **Micro-Borders & Structural Gridlines:** Implemented with 8% to 14% white alphas (`rgba(255, 255, 255, 0.08)` to `rgba(255, 255, 255, 0.14)`), preventing high-contrast visual noise while preserving sharp boundaries between clustered data points.

## Typography
The typography strategy enforces strict hierarchical differentiation between display contexts, standard interface copy, and machine metadata:

- **Geist:** Deployed across system headers, high-level cluster metrics, and navigation landmarks. Its tight geometric aperture and condensed tracking deliver a high-tech, modern infrastructure signature.
- **Inter:** The backbone for transactional views, settings panels, table records, and form configurations. Tuned for maximum legibility at 12px and 14px sizes across dense data layouts.
- **JetBrains Mono:** The primary operational voice for technical tokens, cron syntax, snapshot IDs (`snp_9f82c0b`), IP addresses, byte sizes, checksum hashes, and terminal output. 
- **Tabular Figures:** Monospaced and numerical contexts must enforce tabular figures (`tnum`) to eliminate layout jitter during high-frequency real-time throughput metrics.

## Layout & Spacing
The layout model employs a fluid 12-column grid balanced by compact, high-density margins. Space is treated as a functional resource: compact enough to present multi-cluster backup statuses above the fold, yet spaced to prevent cognitive fatigue during incident triage.

- **Desktop (1280px+):** 12-column fluid grid, 24px margins (`margin`), 16px to 24px gutters (`gutter-lg`). Sidebar docks remain persistent at fixed technical widths (240px collapsed, 280px expanded), with the orchestrator canvas expanding fluidly.
- **Tablet (768px - 1279px):** 8-column layout with 16px gutters and margins. Collapsible telemetry sidecars collapse into sliding overlay sheets.
- **Mobile (Below 768px):** 4-column layout with 16px margins (`margin-sm`) and 12px gutters. Data tables transition to stacked cluster-card summaries, and execution logs enforce lateral scroll with preserved monospaced integrity.
- **Component Rhythm:** Interior padding uses micro increments—`space-xs` (4px) and `space-sm` (8px) for badge internals, input padding, and table cells; `space-md` (12px) and `space-lg` (20px) for container shells and module headers.

## Elevation & Depth
Elevation is achieved through dark surface luminosity layering and glassmorphism rather than heavy drop shadows:

- **Surface Tiers:**
  - *Tier 0 (Root Bed):* Pure obsidian black `#030712`.
  - *Tier 1 (Panels & Shelves):* `#0B1120` with a 1px hairline border of `rgba(255, 255, 255, 0.06)`.
  - *Tier 2 (Interactive Cards & Tables):* `#0F172A` with a 1px border of `rgba(255, 255, 255, 0.09)`.
  - *Tier 3 (Floating Docks & Command Modals):* Backdrop blur (`16px`), background `rgba(15, 23, 42, 0.82)`, and subtle top edge highlights via inner linear borders (`rgba(255, 255, 255, 0.12)`).
- **Depth Lighting:** Shadows are tinted obsidian with high diffusion and minimal y-offset to avoid float disjointedness: `0 8px 32px -4px rgba(0, 0, 0, 0.65)`.
- **Active State Glows:** Triggered elements (such as running backup states or selected orchestrators) utilize radial focus blooms: `0 0 16px rgba(16, 185, 129, 0.18)`.

## Shapes
A "Soft" roundedness (`roundedness: 1` / 4px base radius) is strictly applied across all controls to maintain an engineered, precision-tooled feel. High-curvature components are avoided to preserve screen space and maintain an architectural aesthetic.

- **Base Controls (Inputs, Buttons, Badges):** 4px (`0.25rem`) border radius.
- **Panels & Container Cards:** 8px (`0.5rem`) border radius (`rounded-lg`).
- **Floating Modals & Flyouts:** 12px (`0.75rem`) border radius (`rounded-xl`).
- **Telemetry Indicators:** Status dots and live node pulses are circular (full radius), providing an organic contrast to the structured, angular grid.

## Components

### Buttons & Interactive Controls
- **Primary:** Emerald background (`#10B981`) with dark text (`#022C22`), 4px radius, 32px height in standard density. Micro-top-highlight border `rgba(255, 255, 255, 0.25)` inset. Hover activates luminosity lift to `#34D399`.
- **Secondary / Ghost:** Subsurface slate background (`rgba(255, 255, 255, 0.04)`), 1px border (`rgba(255, 255, 255, 0.08)`), white text (`#F8FAFC`). Hover shifts border to `rgba(255, 255, 255, 0.2)` with background `rgba(255, 255, 255, 0.08)`.
- **Danger Action:** Obsidian fill with subtle crimson border (`#F43F5E`), hover fills background with crimson at 20% opacity and text `#FDA4AF`.

### Status Indicators & Signal Chips
- **Chips:** Monospace text in `label-sm` font. 20px fixed height. Border opacity matched to semantic signal:
  - *Healthy / Completed:* Background `rgba(16, 185, 129, 0.12)`, text `#34D399`, border `rgba(16, 185, 129, 0.25)`.
  - *In-Flight / Streaming:* Background `rgba(6, 182, 212, 0.12)`, text `#38BDF8`, border `rgba(6, 182, 212, 0.25)`. Accompanied by a spinning pulse icon.
  - *Warning:* Background `rgba(245, 158, 11, 0.12)`, text `#FBBF24`, border `rgba(245, 158, 11, 0.25)`.
  - *Failed / Degraded:* Background `rgba(244, 63, 94, 0.12)`, text `#FB7185`, border `rgba(244, 63, 94, 0.25)`.

### Input Fields & Technical Dropdowns
- **Height & Style:** Compact 32px height, slate bed (`#070D19`), 1px outline (`rgba(255, 255, 255, 0.1)`). Text color is `#F1F5F9`.
- **Focus State:** 1px border transition to `#10B981` accompanied by a localized 2px emerald glow (`rgba(16, 185, 129, 0.25)`).
- **Technical Inputs (Cron / Connection Strings):** Auto-rendered with `JetBrains Mono` at `label-md` size with integrated copy-to-clipboard micro actions.

### Telemetry Tables & Data Lists
- **Structure:** Zero outer table margin; flush containment with 1px slate divider borders.
- **Row Architecture:** 36px compact row height. Alternate row backgrounds are transparent, utilizing micro-border bottoms (`rgba(255, 255, 255, 0.04)`). Row hover triggers unified `#131D31` fill transition.
- **Monospace Alignment:** Snapshot ID hashes, execution runtimes, compression ratios, and cron notations align to strict right or left boundaries using tabular numbers.

### Cards & Container Panels
- Surface color `#0B1120`, interior padding `space-md` (12px), perimeter hairline border (`rgba(255, 255, 255, 0.07)`). Header sections include bottom separation lines with trailing action links and live telemetry badges.

### Domain-Specific Components
- **Cron Orchestrator Builder:** A visual 5-part segmented control coupled with an inline humanized cron translator ("Every day at 03:00 UTC").
- **Live Stream Log Viewer:** Monospace terminal component featuring a dark `#030712` inner container, sticky log level signals (`[INFO]`, `[WARN]`, `[FAIL]`), virtualized scrolling, and micro-pause/resume telemetry controls.
- **Storage Tier Allocation Bar:** A segmented horizontal bar showing distributed capacity across Hot S3, Cold Glacier, and Encrypted Local blocks with color-coded volumetric fills.