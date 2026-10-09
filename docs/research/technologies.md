# Technology Evaluation & Architectural Selection

## 1. Cloud & S3 Storage SDKs

### 1.1 AWS SDK for Go v2 (`github.com/aws/aws-sdk-go-v2`)
- **Architecture**: Modular Go SDK divided into granular packages (`aws`, `config`, `service/s3`, `feature/s3/manager`).
- **Strengths**:
  - Official canonical implementation of AWS Signature Version 4 (SigV4) and SigV4a (multi-region).
  - Strongly typed API request/response structures with context propagation throughout all call chains.
  - Native Transfer Manager (`feature/s3/manager`) providing out-of-the-box multipart upload/download abstractions with configurable concurrency and chunk sizing.
  - Highly decoupled middleware stack: allows injecting custom endpoint resolvers, retry policies, HTTP transport round-trippers, header injectors, and telemetry hooks without modifying core business logic.
  - Active maintenance, zero external C-dependencies, strict Go module hygiene.
- **Weaknesses**:
  - Pointer-heavy structs (`aws.String()`, `aws.Int64()`) require defensive nil-checking.
  - Default configurations favor standard AWS endpoints; third-party S3-compatible providers (Cloudflare R2, MinIO, Wasabi) require explicit endpoint resolution overrides and forced path-style flags (`UsePathStyle: true`).
- **Verdict for Kumokura**: **Selected as the Core S3 Driver Adapter**. Its middleware architecture makes it trivial to implement custom provider profiles (Wasabi, MinIO, R2, Ceph) while retaining full SigV4 compliance and low-level HTTP transport customization.

### 1.2 MinIO Go Client SDK (`github.com/minio/minio-go/v7`)
- **Architecture**: Monolithic high-level client library tailored for MinIO and AWS S3 object storage.
- **Strengths**:
  - Simplified API: high-level methods like `FPutObject`, `FGetObject`, and `Core` primitives.
  - Built-in automatic multipart chunk size calculation based on source file size.
  - Native support for MinIO server admin APIs (user management, bucket notifications, replication policies).
- **Weaknesses**:
  - Tightly coupled HTTP client and retry architecture; harder to intercept low-level HTTP sockets for custom adaptive rate limiting compared to AWS SDK v2 middleware.
  - Less modular dependency graph compared to AWS SDK v2.
  - AGPL/Apache licensing considerations across specific MinIO repository modules.
- **Verdict for Kumokura**: **Not selected as primary core driver**. However, we maintain compatibility with MinIO endpoints by leveraging AWS SDK v2 with custom endpoint overrides and path-style addressing.

---

## 2. Desktop GUI Frameworks

### 2.1 Wails v2 (`github.com/wailsapp/wails/v2`) vs Wails v3
- **Architecture**: Go backend compiled to native binary with embedded WebKit/Chromium webview (WebKit2GTK on Linux, WKWebView on macOS, WebView2 on Windows). IPC bridge translates Go struct methods to auto-generated TypeScript bindings.
- **Wails v2 Evaluation**:
  - *Maturity*: Production-proven, rock-solid stability across macOS, Windows, and modern Linux distributions.
  - *Ecosystem*: Battle-tested build tooling (`wails build`), native asset bundling, cross-platform window management, native menus, and dialogs.
  - *Footprint*: Minimal executable size (~15MB compressed), low idle RAM usage (~40-80MB), native look and feel.
- **Wails v3 Status**:
  - *Evaluation*: Wails v3 represents a major architectural overhaul with multi-window capabilities, native menu bar apps, and independent worker processes. However, as of late 2024 / early 2025, v3 is still in active alpha/beta transition. API interfaces are subject to breaking changes.
- **Verdict for Kumokura**: **Replaced by native Fyne v2** to eliminate external webview/node dependencies, avoid webview version disparities across Linux distros, and deliver a 100% pure Go native experience.

### 2.2 Fyne (`fyne.io/fyne/v2`)
- **Architecture**: Pure Go native GUI toolkit rendered entirely via OpenGL / Metal / DirectX hardware graphics pipeline.
- **Strengths**:
  - Single language stack (100% Go, no HTML/CSS/JS toolchains).
  - Cross-platform consistency across desktop and mobile.
  - Lightweight single binary output.
- **Weaknesses**:
  - Non-native UI aesthetics: custom widget rendering does not match OS native controls or modern web design systems (Tailwind/Radix/Lucide).
  - Limited typography rendering and styling flexibility for complex, high-density data grids, interactive multipart progress gauges, and rich object inspection sidebars.
  - Complex custom component authoring compared to component-rich web ecosystems.
- **Verdict for Kumokura**: **Selected**. Pure Go architecture, zero webview dependencies, hardware-accelerated OpenGL/Wayland rendering, and direct integration with domain services.

---

## 3. Web & Desktop Frontend Stack

### 3.1 Vue 3 + TypeScript + Vite + Tailwind CSS
- **Architecture**: Reactive single-page application hosted inside the Wails Webview.
- **Strengths**:
  - *Vue 3 Composition API & `<script setup>`*: Explicit, clean separation of reactive state, computed values, and lifecycle hooks. Excellent ergonomics for complex stateful interfaces like multi-tab object browsers.
  - *TypeScript*: End-to-end type safety when interacting with Go backend structs through Wails auto-generated models (`frontend/wailsjs/go/models.ts`).
  - *Tailwind CSS*: Rapid creation of high-density, accessible UI components with dark mode support.
  - *Pinia*: Centralized reactive stores for active accounts, bucket navigation stacks, transfer progress queues, and system settings.
  - *Vite*: Sub-second HMR (Hot Module Replacement) during desktop app development.
- **Verdict for Kumokura**: **Selected as the official GUI frontend technology**.

---

## 4. CLI Frameworks

### 4.1 Cobra (`github.com/spf13/cobra`) + Viper (`github.com/spf13/viper`)
- **Architecture**: Standard Go CLI framework powering Kubernetes (`kubectl`), GitHub CLI (`gh`), Hugo, and Docker CLI.
- **Strengths**:
  - POSIX-compliant flag parsing (pflag), nested sub-commands (`kumokura bucket create`, `kumokura object sync`).
  - Shell autocompletion generation (Bash, Zsh, Fish, PowerShell).
  - Man-page and Markdown documentation generation.
  - Viper integration for layered configuration resolution (flags > env vars > config file > defaults).
- **Weaknesses**:
  - Global mutable state in default examples if not carefully engineered.
- **Verdict for Kumokura**: **Selected for CLI orchestration**. In accordance with our Hexagonal Architecture, Cobra commands will instantiate and call the Application Service layer without embedding business logic into command action handlers.

---

## 5. Terminal User Interface (TUI) Frameworks

### 5.1 Bubble Tea v2 (`github.com/charmbracelet/bubbletea/v2`), Bubbles, and Lip Gloss
- **Architecture**: Implementation of The Elm Architecture (TEA) in Go: Model, Update, View, and asynchronous Cmd messages.
- **Strengths**:
  - Pure functional state transitions: predictable state updates without concurrent race conditions in rendering.
  - Bubble Tea v2 advantages: modernized renderer, enhanced mouse and window management, decoupled layout mathematics, improved ANSI color profiling.
  - Lip Gloss: Declarative CSS-like styling in terminal strings (margins, borders, padding, foreground/background colors, text alignment).
  - Bubbles: Battle-tested sub-components (text inputs, spinners, progress bars, viewports, tables).
- **gh-dash Architectural Alignment & Skill Guidelines**:
  - Adheres strictly to the `golang-tui-design` skill specification:
    - `View()` is a pure, side-effect-free function with zero I/O and zero state mutations.
    - ANSI width calculations utilize `lipgloss.Width()` and `ansi.Truncate()`, never raw Go byte string slicing.
    - Responsive degradation: seamless multi-column to single-column stacking down to `80x24` terminals.
    - Active vs. Inactive panel border contrast (`#38BDF8` active cyan vs `#374151` inactive muted gray).
    - Status footer with discoverable hotkeys and modal overlay capture for search and confirmations.
- **Verdict for Kumokura**: **Selected as the official TUI engine**.

---

## 6. Local Embedded Database Drivers

### 6.1 `modernc.org/sqlite` (Pure Go) vs `github.com/mattn/go-sqlite3` (CGO)
- **Comparison**:
  - `mattn/go-sqlite3`: Standard CGO-based binding to native SQLite C library. Excellent performance, but requires cross-compilation toolchains (musl-cross, MinGW, macOS SDKs) during CI/CD releases.
  - `modernc.org/sqlite`: Pure Go port generated via C-to-Go transpiler (`ccgo`). Requires zero CGO (`CGO_ENABLED=0`), enabling instant cross-compilation across all target architectures (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`).
- **Performance Evaluation**:
  - For Kumokura's use case (storing account profiles, transfer ledger state, multipart checkpoints, and directory sync caches), modernc SQLite achieves >20,000 writes/sec in WAL mode with indexed lookups, comfortably exceeding our transfer queue requirements.
- **Verdict for Kumokura**: **Selected `modernc.org/sqlite`**. Guarantees hassle-free cross-compilation in GoReleaser without compromising throughput or durability. WAL mode (`PRAGMA journal_mode=WAL;`) is enforced for concurrent multi-goroutine access.

---

## 7. Native OS Credential Storage (Keyrings)

### 7.1 `github.com/zalando/go-keyring` vs `github.com/99designs/keyring`
- **Comparison**:
  - `zalando/go-keyring`: Focused library binding to macOS Keychain, Windows Credential Manager, and Linux Secret Service (via DBus). Zero dependencies on external C libraries on Linux/macOS.
  - `99designs/keyring`: Rich multi-backend library supporting Keychain, Windows Cred, Pass, KWallet, File, and encrypted files. Higher complexity and broader surface area.
- **Verdict for Kumokura**: **Selected `zalando/go-keyring`** for primary native keyring access, paired with a custom fallback adapter to an AES-256-GCM encrypted local SQLite database for headless Linux / CI environments where DBus Secret Service is unavailable.

---

## 8. End-to-End Testing & Automation

### 8.1 Playwright for Wails Desktop GUI Testing
- **Architecture**: Headless/headed browser automation engine supporting Chromium / WebKit / Firefox.
- **Application to Wails**:
  - In Wails v2 dev mode (`wails dev`), the frontend runs as an accessible HTTP web application while proxying Go IPC calls via WebSocket.
  - Playwright tests can exercise the entire Vue 3 UI, test file drag-and-drop simulations, bucket list rendering, modal dialogs, and credential entry.
- **Verdict for Kumokura**: **Selected for automated GUI regression and integration testing**.

---

## 9. Release Automation & Cross-Platform Packaging

### 9.1 GoReleaser (`goreleaser`)
- **Capabilities**:
  - Builds optimized binaries across Linux (amd64, arm64), macOS (Intel, Apple Silicon Universal), and Windows (amd64).
  - Generates checksum manifests, signed release notes, and GitHub Releases.
  - Packages native installers: `.deb`, `.rpm`, Homebrew tap formulas, and Windows zip/installer packages.
  - Integrates with Wails CLI for desktop packaging (macOS `.app` / `.dmg`, Windows `.exe` / `.msi`, Linux `.AppImage` / `.deb`).
- **Verdict for Kumokura**: **Selected as the official build and release orchestrator**.

---

## 10. Technology Selection Matrix Summary

| Architectural Component | Selected Technology | Alternative Considered | Primary Justification |
|---|---|---|---|
| **S3 Protocol Adapter** | `aws-sdk-go-v2` | `minio-go/v7` | Modular middleware, canonical SigV4/SigV4a, custom transport control. |
| **Desktop GUI Engine** | Fyne v2 (`fyne.io/fyne/v2`) | Wails v2, Wails v3 | Full native Go GUI, hardware acceleration, zero webview/npm toolchains. |
| **Desktop GUI Frontend** | Pure Native Go Widgets | Vue 3, React | Single binary compilation, unified language stack, low idle footprint. |
| **CLI Engine** | Cobra + Viper | Urfave/cli | Industry standard, robust subcommands, pflag POSIX compliance. |
| **TUI Engine** | Bubble Tea v2 + Lip Gloss | tview, cview | Decoupled Elm Architecture, gh-dash visual parity, pure `View()` rendering. |
| **Local SQLite Driver** | `modernc.org/sqlite` | `mattn/go-sqlite3` | Pure Go, CGO-free cross-compilation across all platforms. |
| **OS Keyring** | `zalando/go-keyring` | `99designs/keyring` | Lightweight, native OS bindings, paired with encrypted DB fallback. |
| **E2E UI Testing** | Playwright | Cypress | Native multi-browser automation, seamless Vite dev-server testing. |
| **Build & Release** | GoReleaser | Custom Makefiles | Automated cross-platform packaging, Homebrew, deb/rpm generation. |
