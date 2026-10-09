# ADR 0003: TUI gh-dash Inspired Architecture

- **Status**: Accepted
- **Date**: 2026-10-07
- **Authors**: Kumokura Core Engineering Team

---

## 1. Context & Problem Statement

Command-line utilities often require users to memorize dozens of flags and sub-commands, while traditional desktop GUIs disrupt keyboard-centric terminal workflows. Terminal power users, SREs, and DevOps engineers demand an interactive, keyboard-driven interface that lives directly inside their tmux or shell environment.

However, poorly designed TUIs suffer from:
- Jarring visual flickering and unstyled ANSI artifacts.
- Unresponsive layouts that break or crash on non-standard terminal dimensions (< 100 columns).
- Ambiguous focus states where typed keys trigger unintended side effects.
- Direct network calls inside render loops freezing the terminal interface.

Kumokura requires an interactive TUI inspired by the industry benchmark `gh-dash`, delivering clean visual hierarchy, predictable keyboard navigation, and responsive degradation.

---

## 2. Decision

We implement the Kumokura TUI using **Bubble Tea v2**, **Lip Gloss**, and **Bubbles**, adhering strictly to the architectural standards defined in the `golang-tui-design` skill.

### 2.1 The Elm Architecture (TEA) Separation
The TUI is implemented as a pure unidirectional state machine:
- **`Model`**: Retains domain snapshot state (active buckets, visible object keys, active transfers), view dimensions, focus index, and active modal overlays.
- **`Init`**: Returns initial asynchronous `tea.Cmd` triggers (e.g. loading active profile credentials and bucket lists).
- **`Update`**: Handles incoming messages (`tea.KeyMsg`, `tea.WindowSizeMsg`, `DataLoadedMsg`, `TransferProgressMsg`) and transitions the model state deterministically.
- **`View`**: A 100% pure, side-effect-free function returning the styled terminal string representation. No network calls, disk reads, or model mutations occur inside `View()`.

### 2.2 Visual Grammar & Palette (gh-dash Benchmark)
- **Palette**: Dark slate backgrounds (`#0F172A`, `#1E293B`), active sky-cyan borders (`#38BDF8`), muted inactive borders (`#374151`), and semantic status indicators (green `#4ADE80`, amber `#FBBF24`, red `#F87171`).
- **Rounded Borders**: All panel cards and modal windows use `lipgloss.RoundedBorder()`.
- **Selected Row Highlight**: Subtle background tinting (`#334155`) with high-contrast text rather than harsh reverse-video blinding.

### 2.3 Hierarchical Focus Routing State Machine
Input events (`tea.KeyMsg`) route through a deterministic 4-stage hierarchy:
1. **Modal Overlay**: If a modal dialog (help, confirmation, upload prompt) is open, all input is captured exclusively by the modal.
2. **Search / Filter Input**: If the live fuzzy filter (`/`) is active, input routes to `bubbles/textinput`.
3. **Active Panel**: Otherwise, navigation commands (`j`, `k`, `g`, `G`, `Enter`, `Space`) route to the currently focused panel.
4. **Global Keymap**: Unhandled keys route to global shortcuts (`Tab`, `1..4`, `q`, `?`).

### 2.4 Responsive Degradation Strategy
The TUI adjusts dynamically to terminal window resize messages (`tea.WindowSizeMsg`):
- **$\ge 160 \times 45$**: 3-panel split (Buckets 20%, Object Explorer 50%, Metadata/Preview Drawer 30%).
- **$120 \times 35$ to $159 \times 44$**: 2-panel master-detail split (Buckets 35%, Objects 65%). Inspector drawer available via toggle (`i`).
- **$80 \times 24$ to $119 \times 34$**: Single full-screen panel with tab switching (`1`, `2`, `3`, `4`). Padding and margins compressed to 0.
- **$< 80 \times 24$**: Graceful warning card: *"Terminal window too small: Kumokura requires minimum 80x24"*. Zero panics or crashes.

### 2.5 ANSI Width & Truncation Discipline
All visual widths are computed using `lipgloss.Width()`, and string truncations are performed via `ansi.Truncate()` / `lipgloss.NewStyle().MaxWidth()`. Raw Go byte-slicing (`len()` or `s[:n]`) is strictly prohibited on formatted terminal strings.

---

## 3. Consequences

### Positive:
- **World-Class Ergonomics**: Terminal power users can navigate, preview, download, and manage S3 storage with sub-second keyboard responses.
- **Zero Terminal Freezes**: Asynchronous `tea.Cmd` execution ensures smooth 60fps rendering even during high-throughput file transfers.
- **High Visual Appeal**: Elegant gh-dash aesthetic positions Kumokura as a modern, premier developer tool.

### Negative / Trade-offs:
- **Viewport State Management**: Maintaining multi-breakpoint layouts and cursor boundaries across dynamic terminal resizing requires disciplined testing.
