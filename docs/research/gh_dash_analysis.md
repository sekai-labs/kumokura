# gh-dash UI/UX Deep Dive & Terminal Design Architecture

## 1. Overview of `gh-dash` as a Benchmark

`gh-dash` (created by dlvhdr) represents the gold standard for developer terminal user interfaces (TUIs). It transformed GitHub workflow management from disjointed web tabs and slow CLI queries into an instantaneous, keyboard-driven dashboard.

Kumokura's TUI modal interface (`kumokura tui`) directly adopts the visual grammar, spatial layout rules, focus hierarchy, and responsive behaviors of `gh-dash`, adapted for cloud object storage management.

---

## 2. Visual Hierarchy & Design System Breakdown

### 2.1 Color Palette & Semantic Tokens
`gh-dash` achieves high visual polish by avoiding the garish, saturated primary colors typical of legacy terminal utilities (bright red, harsh yellow, neon blue). Instead, it adopts modern subdued slate and pastel accents:

- **Surface & Backgrounds**:
  - Base Terminal Background: Terminal default or `#0F172A` (Slate 900)
  - Card / Panel Elevated Background: `#1E293B` (Slate 800)
  - Active Row Highlight: `#334155` (Slate 700) with subtle foreground contrast
- **Borders**:
  - Inactive Panel Border: `#374151` (Gray 700) or `#475569` (Slate 600)
  - Active / Focused Panel Border: `#38BDF8` (Sky 400) or `#06B6D4` (Cyan 500)
  - Modal Border: `#818CF8` (Indigo 400) or `#A855F7` (Purple 500)
- **Semantic Status Badges**:
  - Success / Synced / Active: `#4ADE80` (Green 400)
  - Running / In Progress / Uploading: `#38BDF8` (Sky 400)
  - Warning / Modifying / Transitioning: `#FBBF24` (Amber 400)
  - Error / Failed / Missing: `#F87171` (Red 400)
  - Neutral / Read-Only / Inactive: `#94A3B8` (Slate 400)

### 2.2 Border Geometry
- Exclusively uses **Rounded Borders** (`lipgloss.RoundedBorder()`) for outer card boundaries and modals.
- Hard ASCII corners (`+--+` or double-line `╔══╗`) are strictly avoided.
- Inactive panels have low-contrast subtle lines; the active panel "pops" immediately with the cyan/sky accent.

---

## 3. Spatial Layout & Responsive Viewport Adaptations

### 3.1 Three-Zone Layout Anatomy

```
+-----------------------------------------------------------------------------------------+
| [1 Buckets]  [2 Objects]  [3 Transfers]  [4 Sync]   |  Profile: wasabi-prod (us-east-1) | <- Header & Tab Bar
+------------------------------------+----------------------------------------------------+
| ACTIVE PANE: BUCKETS (6)           | OBJECT EXPLORER: s3://assets-production/           |
| > assets-production      1.4 TB    | Key                         Size     Last Modified |
|   backups-daily          842 GB    | > images/                   [DIR]    -             |
|   client-uploads         12.8 GB   |   config.json               14 KB    2025-05-12    |
|   logs-archive           4.2 TB    |   release-v2.tar.gz         420 MB   2025-05-14    |
|   temp-scratch           120 MB    |                                                    |
|                                    |                                                    |
+------------------------------------+----------------------------------------------------+
| [Tab] Switch Pane  [j/k] Navigate  [/] Filter  [u] Upload  [d] Download  [?] Help       | <- Status Footer
+-----------------------------------------------------------------------------------------+
```

1. **Header & Context Bar (Height: 2–3 rows)**:
   - Left: Main functional tab pills (`[1 Buckets] [2 Objects] [3 Transfers] [4 Sync]`).
   - Center/Right: Active profile identifier, current S3 region, online status, and transfer rate.
2. **Body Workspace (Height: Viewport - 4 rows)**:
   - Configurable split panels (e.g. Master-Detail 35% / 65% split, or full-width data table).
   - Card headers contain dynamic entity counters (e.g., `BUCKETS (6)`, `TRANSFERS (2 Active, 14 Completed)`).
3. **Status Bar & Keymap Footer (Height: 1–2 rows)**:
   - High-density, single-line or two-line footer.
   - Shows currently available shortcut actions using bracketed keys: `[j/k] Navigate  [/] Search  [u] Upload  [d] Download  [space] Select  [?] Help`.
   - Displays real-time message toasts and transfer notifications.

---

### 3.2 Responsive Degradation Rules

Terminal windows vary widely in size. Kumokura follows rigorous degradation rules based on terminal dimensions received via `tea.WindowSizeMsg`:

| Viewport Category | Dimensions | Layout Strategy |
|---|---|---|
| **Ultra-Wide** | $\ge 160 \times 45$ | 3-Column Split: Buckets (20%), Object Tree (50%), Object Detail / Metadata Sidebar (30%). Full transfer gauges visible. |
| **Standard Desktop** | $120 \times 35$ to $159 \times 44$ | 2-Column Split: Master Navigation (35%), Detail Workspace (65%). Sidebar collapses to toggle drawer (`[i] Info`). |
| **Compact Terminal** | $100 \times 28$ to $119 \times 34$ | 2-Column with compressed columns: File size/dates truncated; relative timestamps (`2d ago`) instead of ISO-8601. |
| **Minimum Required** | $80 \times 24$ to $99 \times 27$ | Single Column Tabbed View: Sidebar and auxiliary panels hidden. Tabs switch full-screen views. Margins reduced to 0. |
| **Too Small** | $< 80 \times 24$ | Safe Fallback Screen: Centered card displaying *"Terminal window too small: Kumokura requires minimum 80x24 (Current: WxH)"*. Zero panics. |

---

## 4. Keyboard Navigation & Focus Routing Engine

### 4.1 Hierarchical Focus State Machine
In a multi-pane TUI, global key captures create severe usability bugs (e.g., typing 'd' in a search bar accidentally deleting a bucket). Kumokura routes `tea.KeyMsg` through a strict hierarchical state machine:

```
[tea.KeyMsg arrives]
         |
         v
+-------------------------+
| Is Modal Overlay Open?  | --- YES ---> Dispatch strictly to Modal Component (Escape cancels, Enter confirms)
+-------------------------+
         | NO
         v
+-------------------------+
| Is Search / Filter Bar  | --- YES ---> Dispatch to bubbles/textinput (Type text, Enter locks filter, Esc clears)
| currently Focused?      |
+-------------------------+
         | NO
         v
+-------------------------+
| Route to Active Focused | -----------> Active Pane handles local navigation (j/k, Up/Down, Enter, Space)
| Panel (Buckets/Objects) |
+-------------------------+
         | Unhandled key
         v
+-------------------------+
| Check Global Keymap     | -----------> Global Shortcuts (Tab, Shift-Tab, 1..4, ?, q, Ctrl+C)
+-------------------------+
```

### 4.2 Standard Keymap Matrix

| Keybinding | Scope | Function |
|---|---|---|
| `Tab` / `Shift-Tab` | Global | Cycle active panel focus clockwise / counter-clockwise |
| `1`, `2`, `3`, `4` | Global | Direct jump to Tab: 1=Buckets, 2=Objects, 3=Transfers, 4=Sync |
| `j` / `k` or `Down` / `Up` | Panel | Navigate selected row cursor down / up |
| `g` / `G` | Panel | Jump to first item / jump to last item |
| `Ctrl+d` / `Ctrl+u` | Panel | Half-page down / half-page up |
| `Enter` | Panel | Drill down / open bucket / expand directory / view details |
| `Backspace` / `h` | Objects | Navigate to parent directory (`../`) |
| `/` | Panel | Open live fuzzy filter search input for current list |
| `Escape` | Global | Dismiss modal, exit filter mode, return focus to main pane |
| `u` | Objects | Trigger Upload file/directory prompt |
| `d` | Objects | Trigger Download selected object(s) prompt |
| `Delete` / `x` | Objects | Trigger Delete object confirmation dialog |
| `m` | Objects | Move / Server-Side Copy selected object |
| `p` | Objects | Generate Presigned URL dialog |
| `i` | Objects | Toggle Object Metadata & Tags Inspector Drawer |
| `r` | Global | Force refresh active view from cloud API |
| `?` | Global | Toggle Help Overlay Modal |
| `q` | Global | Quit TUI (safely pausing background transfers) |

---

## 5. Modal Dialogs & Search Overlays

### 5.1 Centered Modal Positioning Math
Modals in `gh-dash` are rendered as floating overlays centered over the darkened background workspace. Lip Gloss handles this via explicit coordinate calculation:

```
ModalWidth  = Min(80, WindowWidth - 8)
ModalHeight = Min(20, WindowHeight - 6)

TopPadding  = (WindowHeight - ModalHeight) / 2
LeftPadding = (WindowWidth - ModalWidth) / 2
```

Kumokura uses a full-screen canvas composite to render the dimmed background view beneath, with the bordered modal rendered on top with a distinct border color (`#818CF8`).

### 5.2 Confirmation Modal Safety
Destructive operations (e.g. deleting a bucket with versioned objects, recursive folder deletion) trigger confirmation dialogs requiring explicit keyboard confirmation:
- Standard deletion: `[y]es / [n]o` selector.
- High-risk deletion (e.g., deleting non-empty production bucket): User must type the exact bucket name or press `Ctrl+Enter` after confirmation.

---

## 6. Implementation Disciplines (Derived from `golang-tui-design`)

1. **`View()` Purity**:
   `View()` must never trigger cloud API calls, read local files, mutate row selections, or allocate network sockets. It is a strictly pure mapping of `Model -> string`.
2. **String Width Discipline**:
   All string truncations and layout alignments use `lipgloss.Width()` and `ansi.Truncate()` to guarantee that UTF-8 multi-byte characters and ANSI styling escape codes do not corrupt line lengths or break panel borders.
3. **Async Cmd Pipeline**:
   Listing S3 buckets or scanning large directories runs strictly inside `tea.Cmd` goroutines, returning typed messages (`BucketsLoadedMsg`, `ObjectsLoadedMsg`, `TransferProgressMsg`) to the `Update()` loop. The UI never freezes or stutters during network I/O.
