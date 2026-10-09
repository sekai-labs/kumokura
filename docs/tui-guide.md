# Kumokura TUI User Guide

## Overview

The Kumokura Terminal User Interface (`kumokura-tui` or `kumokura tui`) provides a high-density, dual-pane file management experience inspired by `mc` (Midnight Commander) and `gh-dash`.

## Launching

```bash
# Launch directly
kumokura-tui

# Or via the unified CLI binary
kumokura tui
```

## Navigation & Keybindings

### Global Controls
- `Tab`: Switch focus between Left Pane and Right Pane.
- `q` / `Ctrl+C`: Exit the TUI application.
- `?`: Toggle the in-app help modal.

### Pane Navigation
- `j` / `Down Arrow`: Move cursor down by one item.
- `k` / `Up Arrow`: Move cursor up by one item.
- `g`: Jump to the top of the list.
- `G`: Jump to the bottom of the list.
- `Enter`: Open selected bucket or enter folder prefix.
- `Backspace` / `u`: Navigate up to parent prefix or bucket root.

### Object & Transfer Operations
- `d`: Download selected object to local directory.
- `u`: Upload file from local filesystem to active prefix.
- `x` / `Delete`: Delete selected object (prompts confirmation).
- `t`: Open transfers status overlay.
- `s`: Open sync planner dialog.
