# Kumokura Desktop GUI Guide

## Overview

The Kumokura Desktop GUI is a native, lightweight graphical workstation built on **Fyne v2** (`fyne.io/fyne/v2`). It consumes less than 60MB RAM on idle while providing a compact, developer-focused desktop storage environment with zero webview overhead.

## Prerequisites & Build (Linux)

Building the desktop GUI on Linux requires OpenGL, X11, and Wayland development packages:

```bash
# Debian / Ubuntu
sudo apt-get update
sudo apt-get install -y libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev libxkbcommon-dev
```
*(Note: `libwayland-dev` and `libxkbcommon-dev` provide Wayland client headers such as `wayland-client-core.h` when compiling with `-tags wayland`.)*

## Launching

```bash
make build
./bin/kumokura-desktop
```

## User Interface Layout

### 1. Top Header
- **Breadcrumb Navigation**: Displays current bucket and prefix hierarchy. Click any segment to jump back up the directory tree.
- **Search & Filter**: Real-time fuzzy filter for displayed object keys and prefixes.
- **Theme Toggle**: Switch between sleek Dark Mode (Slate 950) and Crisp Light Mode.

### 2. Left Sidebar
- **Accounts Tree**: Quickly switch between connected S3 providers (AWS, MinIO, Cloudflare R2, Wasabi, etc.).
- **Bucket List**: Displays all buckets associated with the selected account.
- **Transfers Badge**: Shows total running and queued file transfers.

### 3. Main Object Explorer
- **High-Density Table**: Displays Name, Size (formatted with binary IEC units), Storage Class, and Last Modified timestamp.
- **Double Click Navigation**: Double click any directory row to descend into virtual folder prefixes.
- **Action Toolbar**: Download, copy presigned links, and delete items.

### 4. Collapsible Details Drawer (Right)
- Inspect object metadata, ETag, exact byte size, and full key path.
- One-click presigned URL generator with clipboard copy helper.

### 5. Real-Time Transfer Drawer (Bottom)
- Live transfer speed gauges in MiB/s.
- Progress bars and transfer control buttons (Pause, Resume, Cancel).
