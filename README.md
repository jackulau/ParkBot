# ParkBot

Automated parking permit purchasing bot with a native desktop GUI.

ParkBot uses Chrome browser automation (via [go-rod](https://github.com/go-rod/rod)) to navigate your university parking portal and purchase a permit with your saved configuration. The desktop GUI is built with [Fyne](https://fyne.io/).

## Quick Start

1. Download or build the binary for your platform (see below).
2. Run the binary. The GUI will open.
3. Fill in your permit keyword, vehicle, billing info, and Chrome profile path.
4. Click **SAVE CONFIG** to persist your settings to `config.yaml`.
5. Click **RUN** to start the bot.

## Building from Source

### Prerequisites

- **Go 1.22+** — [https://go.dev/dl/](https://go.dev/dl/)
- **C compiler** — Fyne requires CGO for platform-native graphics:
  - **macOS**: Xcode Command Line Tools (`xcode-select --install`)
  - **Linux**: `gcc`, `pkg-config`, and OpenGL/X11 dev libraries
  - **Windows**: MinGW-w64 or MSYS2

### Build for your platform

```bash
# Simple build for your current OS
go build -o parkbot .

# Or use the build script for all platforms
chmod +x build.sh
./build.sh
```

### Build script usage

```bash
./build.sh                  # Build all available targets
./build.sh darwin            # macOS only (Intel + Apple Silicon)
./build.sh linux             # Linux only (amd64 + arm64)
./build.sh windows           # Windows only (amd64)
./build.sh clean             # Remove ./dist/ directory
```

Output binaries are placed in `./dist/` with the naming convention:
```
parkbot-{os}-{arch}[.exe]
```

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `OUTDIR` | `./dist` | Output directory for binaries |
| `LDFLAGS` | `-s -w` | Go linker flags (default strips debug info) |
| `VERSION` | `dev` | Version label for the build |
| `CC_LINUX_AMD64` | auto-detect | C compiler for linux/amd64 cross-compilation |
| `CC_LINUX_ARM64` | auto-detect | C compiler for linux/arm64 cross-compilation |
| `CC_WINDOWS_AMD64` | auto-detect | C compiler for windows/amd64 cross-compilation |

### Cross-compilation

Fyne requires CGO, so cross-compiling needs a C cross-compiler for each target platform.

#### From macOS, build for Linux

```bash
# Install musl-cross (provides x86_64-linux-musl-gcc)
brew install filosottile/musl-cross/musl-cross

# Build
CC_LINUX_AMD64=x86_64-linux-musl-gcc ./build.sh linux
```

#### From macOS, build for Windows

```bash
# Install MinGW-w64
brew install mingw-w64

# Build
./build.sh windows
```

#### From Linux, build for Windows

```bash
# Debian/Ubuntu
sudo apt install gcc-mingw-w64-x86-64

# Build
./build.sh windows
```

## Runtime Dependencies

### All platforms

- **Google Chrome** or **Chromium** — The bot uses Chrome DevTools Protocol (via go-rod) to automate the parking portal. Chrome must be installed and accessible.
- **Internet connection** — Required to access the parking portal.

### macOS

- macOS 10.15 (Catalina) or later
- No additional runtime dependencies beyond Chrome
- Chrome profile path: `~/Library/Application Support/Google/Chrome/Default`

### Linux

- X11 or Wayland display server
- OpenGL drivers (Mesa or proprietary)
- System libraries: `libgl1-mesa-dev`, `xorg-dev` (these are typically pre-installed on desktop Linux)
- Chrome profile path: `~/.config/google-chrome/Default`

### Windows

- Windows 10 or later
- No additional runtime dependencies beyond Chrome
- Chrome profile path: `%LOCALAPPDATA%\Google\Chrome\User Data\Default`

## Configuration

ParkBot reads configuration from `config.yaml` in the current directory (or a path specified as the first command-line argument).

```yaml
permit_keyword: COMMUTER        # Text to match in the permit grid
vehicle_keyword: HONDA           # Text to match in the vehicle grid
address_keyword: MAIN            # Text to match in the address grid
email: user@example.com          # Receipt email
one_time: true                   # Write lock file after purchase
chrome_profile: ""               # Leave empty for OS default

billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
  name: "Your Name"
  zip: "50010"
```

### Lock file

When `one_time: true`, the bot writes a `purchased.lock` file after a successful purchase. This prevents accidental double-purchases. Delete the lock file (or use the GUI button) to run again.

## Testing

```bash
go test -v ./...
```

## Project Structure

| File | Description |
|---|---|
| `main.go` | Entry point, loads config, launches GUI |
| `gui.go` | Fyne desktop GUI with custom dark theme |
| `config.go` | YAML config loading/saving, platform-specific Chrome path detection |
| `bot.go` | Browser automation logic using go-rod |
| `build.sh` | Cross-platform build script |
| `build_test.go` | Build and config tests |
