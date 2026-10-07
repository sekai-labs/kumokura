# Kumokura (雲蔵)

> High-performance, cross-cloud S3 storage manager in Go with CLI, TUI, and Desktop GUI interfaces.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Features

- **Unified Hexagonal Architecture**: CLI, TUI, and Desktop interfaces share an identical, high-performance Go domain core.
- **Multi-Cloud S3 Compatibility**: Seamlessly manage AWS S3, MinIO, Cloudflare R2, Wasabi, Backblaze B2, Ceph, and DigitalOcean Spaces.
- **Zero-Plaintext Security**: S3 credentials are encrypted in the native OS Keyring (Keychain, Windows Credential Manager, Secret Service) with AES-256-GCM headless fallback.
- **High-Throughput Transfers**: Tiered buffer pools (`sync.Pool`) eliminate memory bloat during multi-part gigabyte uploads.
- **Directory Synchronization Engine**: Fast 3-way diff comparison, timestamp skew tolerance, dry-run manifests, and mirror deletion safety.
- **Tri-Interface Support**:
  - **CLI (`kumokura`)**: Scriptable, human-friendly tables or pure JSON output.
  - **TUI (`kumokura-tui` / `kumokura tui`)**: Dual-pane keyboard-driven file manager built with Bubble Tea.
  - **Desktop GUI (`kumokura-desktop`)**: Full native desktop app built with Fyne v2 (`fyne.io/fyne/v2`), hardware accelerated, consuming <60MB RAM with zero webview or browser dependencies.

---

## Installation & Quickstart

### Prerequisites (Linux GUI)

Building the desktop GUI (`kumokura-desktop`) on Linux requires OpenGL/X11 and Wayland development headers:

```bash
# Debian / Ubuntu
sudo apt-get update
sudo apt-get install -y libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev libxkbcommon-dev
```
*(Note: `libwayland-dev` and `libxkbcommon-dev` are required for Wayland builds using `-tags wayland`.)*

### Building from Source

```bash
git clone https://github.com/sekai-labs/kumokura.git
cd kumokura

# Build all binaries via Makefile
make build

# Or individually:
go build -o bin/kumokura ./cmd/kumokura
go build -o bin/kumokura-tui ./cmd/kumokura-tui
go build -tags wayland -o bin/kumokura-desktop ./cmd/kumokura-desktop
```

### Adding Your First Storage Account

```bash
# Add AWS Account
kumokura account add --name aws-prod --type AWS --region us-east-1 \
  --access-key AKIAIOSFODNN7EXAMPLE --secret-key wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Add Local MinIO Instance
kumokura account add --name local-minio --type MinIO --endpoint http://127.0.0.1:9000 \
  --path-style --access-key minioadmin --secret-key minioadmin
```

### Quick Usage Examples

```bash
# List all buckets
kumokura bucket list

# Upload a directory with mirror sync
kumokura sync ./dataset my-bucket/dataset/ --delete

# Launch the interactive terminal UI
kumokura-tui

# Launch the desktop graphical interface
kumokura-desktop
```

---

## Architecture Overview

Kumokura is organized around Domain-Driven Design (DDD):
- `cmd/`: Entry points for CLI, TUI, and Desktop GUI.
- `internal/platform/`: SQLite WAL database connection pool, config, TLS transport.
- `internal/security/`: OS Keyring adapter and AES-256-GCM file vault.
- `internal/accounts/`: Storage accounts, credentials, and authentication invariants.
- `internal/providers/`: S3 capability matrix and custom endpoint resolution.
- `internal/buckets/`: S3 bucket lifecycle, versioning, encryption, and tags.
- `internal/objects/`: Streaming object upload/download, pagination, presigned URLs.
- `internal/transfers/`: Multi-part transfer manager, tiered buffers, SQLite checkpoints.
- `internal/synchronization/`: Tree diff planner, conflict policies, glob pattern matching.
- `internal/presentation/`: User interfaces (CLI via Cobra, TUI via Bubble Tea, Desktop GUI via Fyne v2).

See the [Kumokura Wiki](https://github.com/sekai-labs/kumokura/wiki) for full architectural details.

---

## Documentation

Full project documentation is hosted on the GitHub Wiki:
- [Home & Table of Contents](https://github.com/sekai-labs/kumokura/wiki)
- [Status & Verification Report](https://github.com/sekai-labs/kumokura/wiki/Project-Status)
- [Architecture & Design Guide](https://github.com/sekai-labs/kumokura/wiki/Architecture)
- [CLI Reference Manual](https://github.com/sekai-labs/kumokura/wiki/CLI-Reference)
- [TUI User Guide](https://github.com/sekai-labs/kumokura/wiki/TUI-Guide)
- [Desktop GUI Manual](https://github.com/sekai-labs/kumokura/wiki/Desktop-GUI-Guide)
- [Provider Compatibility Matrix](https://github.com/sekai-labs/kumokura/wiki/Provider-Compatibility)
- [Security & Keyring Architecture](https://github.com/sekai-labs/kumokura/wiki/Security-and-Keyring)
- [High-Performance Transfer Engine](https://github.com/sekai-labs/kumokura/wiki/Transfer-Engine)
- [Directory Synchronization Engine](https://github.com/sekai-labs/kumokura/wiki/Directory-Synchronization)
---

## License

[MIT License](LICENSE) © 2026 Sekai Labs
