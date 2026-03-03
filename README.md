# ParkBot

Automated parking permit purchasing bot for ISU with a native desktop GUI. Built with Go, Fyne, and go-rod.

## Features

- Native dark-themed GUI (not a web app)
- Automated Chrome-based permit purchasing via go-rod
- One-click run with configurable permit, vehicle, and billing details
- Lock file prevents accidental double-purchases
- Keyboard shortcuts: `Cmd+S` save, `Cmd+R` run, `Cmd+.` stop, `Cmd+L` clear log (Ctrl on Windows/Linux)

## Requirements

- **Go 1.22+**
- **Google Chrome** or **Chromium** installed
- Platform-specific dependencies:
  - **macOS**: Xcode command-line tools (`xcode-select --install`)
  - **Linux**: OpenGL and X11/Wayland dev libraries (`sudo apt install libgl1-mesa-dev xorg-dev` on Ubuntu)
  - **Windows**: GCC via [MinGW-w64](https://www.mingw-w64.org/) or [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

## Install & Run

```bash
git clone https://github.com/jackulau/ParkBot.git
cd ParkBot
go run .
```

Or build a binary:

```bash
go build -o parkbot .
./parkbot
```

## Configuration

ParkBot stores its config at a platform-specific location:

| OS | Config path |
|----|-------------|
| macOS | `~/Library/Application Support/ParkBot/config.yaml` |
| Windows | `%APPDATA%\ParkBot\config.yaml` |
| Linux | `~/.config/ParkBot/config.yaml` |

You can also pass a custom config path as the first argument:

```bash
./parkbot /path/to/config.yaml
```

### Config format

```yaml
permit_keyword: "COMMUTER"
vehicle_keyword: ""        # empty = first vehicle
address_keyword: ""        # empty = first address
email: "you@example.com"
one_time: true
chrome_profile: ""         # empty = auto-detect
billing:
  card_number: "4111111111111111"
  expiry: "12/28"
  cvv: "123"
  name: "Your Name"
  zip: "50010"
```

All fields can also be edited directly in the GUI. Click **Save Config** to persist.

## Chrome Profile Paths

ParkBot auto-detects your Chrome profile. Default locations:

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/Google/Chrome/Default` |
| Windows | `%LOCALAPPDATA%\Google\Chrome\User Data\Default` |
| Linux | `~/.config/google-chrome/Default` (also checks Chromium, Snap, Flatpak) |

## Cross-Platform Build

```bash
# macOS (native)
go build -o parkbot .

# Linux (requires cross-compiler)
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -o parkbot-linux .

# Windows (requires MinGW)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o parkbot.exe .
```

## How It Works

1. Launches Chrome using your existing profile (preserves login session)
2. Navigates to the ISU parking portal
3. Selects the specified permit, vehicle, and address
4. Fills billing information and submits
5. Writes a lock file to prevent re-purchasing

## Troubleshooting

**"Chrome must be quit first"** -- Close all Chrome windows before running ParkBot. Chrome's debug port requires exclusive access to the profile.

**Lock file exists** -- A previous purchase was completed. Click **Remove Lock** in the GUI or delete the lock file manually to run again.

**Linux: blank window** -- Install OpenGL libraries: `sudo apt install libgl1-mesa-dev xorg-dev`

**Windows: text looks blurry** -- ParkBot adjusts text sizing for ClearType automatically. Ensure your display scaling is set to 100% or 125% for best results.

## License

MIT
