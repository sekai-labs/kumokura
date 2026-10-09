# ADR 0004: Native Desktop GUI Implementation with Fyne v2

- **Status**: Accepted
- **Date**: 2026-10-07
- **Authors**: Kumokura Core Engineering Team

---

## 1. Context & Problem Statement

While terminal interfaces satisfy DevOps engineers, many creative professionals, researchers, and enterprise users demand a native desktop graphical interface. Desktop requirements include:
- Native OS drag-and-drop file transfers.
- Interactive multi-bucket exploration and hierarchical directory trees.
- Visual transfer manager with real-time speed graphs and progress rings.
- System tray integration and native OS desktop notifications.
- Cross-platform support across Linux (Wayland & X11), macOS (Intel & Apple Silicon), and Windows 10/11.

Traditional desktop solutions present trade-offs:
- **Electron**: Heavy binary size (>150MB), excessive idle memory consumption (200MB–600MB RAM), and complex node/native binding issues.
- **Pure Go Toolkits (Fyne)**: Custom widget rendering lacks modern web design ergonomics, rich typography, and mature component ecosystems.
- **Java/Cocoa Bridges (Cyberduck)**: High memory usage and complex cross-platform maintenance.

Kumokura requires a desktop solution that is lightweight, performant, cross-platform, and cleanly integrates with our Hexagonal Go core.

---

## 2. Decision
We select **Fyne v2** (`fyne.io/fyne/v2`) for the Kumokura desktop GUI.

```
+-----------------------------------------------------------------------------------------+
|                                 FYNE V2 DESKTOP GUI APP                                 |
|                                                                                         |
|   +---------------------------------------------------------------------------------+   |
|   |                     NATIVE WIDGET HIERARCHY & COMPOSITION                       |   |
|   |   Widgets: NavigationSplit, BucketList, ObjectTable, DetailsCard, TransferDrawer|   |
|   |   Containers: BorderLayout, HSplit, VSplit, AppToolbar, AccountSelect           |   |
|   +---------------------------------------------------------------------------------+   |
|                                            |                                            |
|                                   Direct Go Invocation                                  |
|                                            v                                            |
|   +---------------------------------------------------------------------------------+   |
|   |                       DESKTOP PRESENTATION ADAPTER                              |   |
|   |   Package: internal/presentation/desktop                                        |   |
|   |   Struct: DesktopApp (wires UI widgets directly to AppContainer services)       |   |
|   +---------------------------------------------------------------------------------+   |
|                                            |                                            |
|                                            v                                            |
|   +---------------------------------------------------------------------------------+   |
|   |                       APPLICATION & DOMAIN CORE (Hexagonal)                     |   |
|   |   AccountService, BucketService, TransferEngine, SyncService                    |   |
|   +---------------------------------------------------------------------------------+   |
+-----------------------------------------------------------------------------------------+
```

### 2.1 Native Adapter Architecture (`internal/presentation/desktop/`)
- A dedicated Go adapter struct (`DesktopApp`) manages the Fyne application lifecycle and window presentation.
- Presentation directly binds application services (`AppContainer`) to standard and custom Fyne widgets.
- Eliminates all webview, Chromium, Node.js, and IPC serialization layers.
- Transfers and async operations update UI components safely on the main thread via event channels or synchronous mutex guards.

### 2.2 Native Desktop Capabilities
- **Zero External Runtime**: Compiles to a single standalone Go binary without needing webview libraries, Node, or npm.
- **Hardware Accelerated**: Renders native UI controls via OpenGL, Metal, and Wayland/X11.
- **Small Binary & Minimal Memory**: Idle memory consumption stays strictly under 60MB RAM.
- **Dialog & Clipboard Integration**: Uses native Fyne dialogs and clipboard for seamless OS desktop integration.

## 3. Consequences

### Positive:
- **No Webview Dependencies**: Removes `webkit2gtk`, WebView2, and WKWebView system dependencies and version incompatibilities.
- **Pure Go Ecosystem**: Build process requires only `go build` / `make build` with zero npm / Vite / TypeScript toolchain dependencies.
- **Strict Hexagonal Alignment**: Desktop GUI is purely an inbound driving adapter under `internal/presentation/desktop/`.
- **Fast Startup & Low RAM**: Instant window rendering and low runtime memory footprint.

### Trade-offs:
- **CGO Requirements for OpenGL/Wayland**: Compilation requires standard C compiler and graphics headers (`libgl`, `wayland`/`x11` headers on Linux).
