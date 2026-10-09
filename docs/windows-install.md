# Kumokura Windows Installation Guide

Kumokura provides three interfaces on Windows:
- **`kumokura.exe`** (CLI): Command-line interface for scripts and terminal operations.
- **`kumokura-tui.exe`** (TUI): Full-screen dual-pane keyboard-driven terminal file manager.
- **`kumokura-desktop.exe`** (GUI): Native desktop graphical interface with Fyne v2.

---

## Method 1: Automated PowerShell Installer (Recommended)

You can install Kumokura using our automated PowerShell installation script. It fetches the latest GitHub release, unpacks it to `%LOCALAPPDATA%\Programs\Kumokura`, adds the folder to your user `PATH`, and creates Start Menu shortcuts.

### One-liner Install via PowerShell

Open PowerShell and execute:

```powershell
irm https://raw.githubusercontent.com/sekai-labs/kumokura/main/packaging/windows/install.ps1 | iex
```

### Or Download and Run Locally

```powershell
# Download the installer script
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/sekai-labs/kumokura/main/packaging/windows/install.ps1" -OutFile "install.ps1"

# Run the installer
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

#### Custom Options
The installer script accepts optional parameters:
```powershell
# Specify a specific version tag and custom install directory
.\install.ps1 -Version "v0.1.0" -InstallDir "C:\Tools\Kumokura"

# Skip Start Menu shortcuts
.\install.ps1 -NoShortcut
```

---

## Method 2: Manual ZIP Installation

1. Go to the [Kumokura Releases page](https://github.com/sekai-labs/kumokura/releases).
2. Download `kumokura_<version>_windows_amd64.zip`.
3. Extract the archive into your preferred directory, for example:
   ```
   C:\Users\<YourUsername>\AppData\Local\Programs\Kumokura\
   ```
   The archive contains:
   - `kumokura.exe`
   - `kumokura-tui.exe`
   - `kumokura-desktop.exe`
4. Add the installation folder to your User Environment `PATH`:
   - Press <kbd>Win</kbd> + <kbd>R</kbd>, type `sysdm.cpl` and press Enter.
   - Go to the **Advanced** tab -> click **Environment Variables**.
   - Under **User variables**, select `Path` and click **Edit**.
   - Click **New** and add the path to the extracted folder (e.g., `%LOCALAPPDATA%\Programs\Kumokura`).
   - Click **OK** to save.
5. Open a new Command Prompt or PowerShell window and verify:
   ```powershell
   kumokura version
   ```

---

## Method 3: Package Managers (Scoop / Winget)

### Scoop (Custom Bucket)
If you use [Scoop](https://scoop.sh/):
```powershell
scoop install https://raw.githubusercontent.com/sekai-labs/kumokura/main/packaging/windows/kumokura.json
```

---

## Uninstallation

To remove Kumokura installed via the automated script:

```powershell
irm https://raw.githubusercontent.com/sekai-labs/kumokura/main/packaging/windows/install.ps1 | iex -ArgumentList "-Uninstall"
```

Or manually:
1. Delete the `%LOCALAPPDATA%\Programs\Kumokura` folder.
2. Remove `%LOCALAPPDATA%\Programs\Kumokura` from your user `PATH` environment variable.
3. Remove the shortcut from `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Kumokura.lnk`.
