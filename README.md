# ParkBot

Automated parking permit purchase bot for Iowa State University's TickeTrak portal. Features a native desktop GUI built with Fyne and Chrome browser automation via go-rod.

---

## Features

- **Native GUI**: Dark-themed desktop application with form fields, activity log, and status indicator
- **Browser Automation**: Uses your existing Chrome profile (preserves login sessions)
- **Config File**: YAML-based configuration for all settings
- **Lock File Protection**: Prevents accidental double-purchases
- **Color-Coded Logging**: Real-time activity log with color-coded messages
- **Cross-Platform**: Runs on macOS, Windows, and Linux

---

## Requirements

### All Platforms

- **Go 1.22+** (for building from source)
- **Google Chrome** installed
- **Active ISU account** logged into the TickeTrak portal in Chrome

### macOS

- macOS 12 (Monterey) or later recommended
- Xcode Command Line Tools: `xcode-select --install`
- Chrome location: `/Applications/Google Chrome.app`

### Windows

- Windows 10 or later
- GCC compiler for CGo (required by Fyne):
  - Install [MSYS2](https://www.msys2.org/) and run: `pacman -S mingw-w64-x86_64-gcc`
  - Or install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
- Chrome location: typically `C:\Program Files\Google\Chrome\Application\chrome.exe`

### Linux

- X11 or Wayland display server
- OpenGL development libraries:
  - **Ubuntu/Debian**: `sudo apt install xorg-dev libgl1-mesa-dev`
  - **Fedora**: `sudo dnf install libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel`
  - **Arch**: `sudo pacman -S xorg-server mesa`
- Chrome or Chromium installed

---

## Installation

### From Source

```bash
git clone https://github.com/jacklau/parkbot.git
cd parkbot
go build -o parkbot .
```

### Run Directly

```bash
go run .
```

---

## Usage

### 1. First-Time Setup

1. Open Google Chrome and log into the [ISU TickeTrak Portal](https://ctitt-iastate.cticloudhost.com/TickeTrak.WebPortal/sso/Home/Index)
2. **Fully quit Chrome** (Cmd+Q on macOS, or close all windows on Windows/Linux)
3. Run ParkBot

### 2. Configuration

Fill in the GUI form fields:

| Field            | Required | Description                                          |
| ---------------- | -------- | ---------------------------------------------------- |
| Permit keyword   | Yes      | Text to match in the permit list (e.g., "COMMUTER")  |
| Vehicle keyword  | No       | Text to match your vehicle (empty = first vehicle)    |
| Address keyword  | No       | Text to match your address (empty = first address)    |
| Email            | No       | Receipt email address                                 |
| Card number      | Yes      | Credit card number                                    |
| Expiry           | Yes      | Card expiration in MM/YY format                       |
| CVV              | Yes      | Card security code                                    |
| Name on card     | No       | Cardholder name                                       |
| Billing ZIP      | No       | Billing ZIP code                                      |
| One-time lock    | No       | Write lock file after purchase to prevent double-buy  |
| Chrome profile   | No       | Path to Chrome profile (auto-detected if empty)       |

Click **SAVE CONFIG** to save settings to `config.yaml` for future use.

### 3. Running the Bot

1. Click **RUN** to start the bot
2. Watch the Activity Log for progress
3. The status indicator changes to "RUNNING" (green)
4. Chrome will open automatically and navigate through the purchase flow
5. On success, "Purchase confirmed!" appears in the log
6. Click **STOP** at any time to cancel

### 4. Config File

You can also create `config.yaml` manually:

```yaml
permit_keyword: COMMUTER
vehicle_keyword: HONDA
address_keyword: ""
email: your@email.com
one_time: true
chrome_profile: ""
billing:
  card_number: "4111111111111111"
  expiry: "12/25"
  cvv: "123"
  name: "John Doe"
  zip: "50010"
```

Pass a custom config path as an argument:

```bash
./parkbot /path/to/my-config.yaml
```

---

## Chrome Profile Paths

ParkBot auto-detects the default Chrome profile. If you use a non-default profile, set the `chrome_profile` field to the full path:

| Platform | Default Path                                                        |
| -------- | ------------------------------------------------------------------- |
| macOS    | `~/Library/Application Support/Google/Chrome/Default`               |
| Windows  | `C:\Users\<user>\AppData\Local\Google\Chrome\User Data\Default`    |
| Linux    | `~/.config/google-chrome/Default`                                   |

For named profiles, replace `Default` with `Profile 1`, `Profile 2`, etc.

---

## Lock File

When the "one-time lock" option is enabled, ParkBot creates `purchased.lock` in the working directory after a successful purchase. This prevents the bot from running again and accidentally buying a second permit.

- **To run again**: Click the "REMOVE LOCK" button in the GUI, or manually delete `purchased.lock`
- **Lock file location**: Same directory where you run the `parkbot` binary

---

## Troubleshooting

### "Launching Chrome" error

Chrome must be **fully quit** before ParkBot can use your profile. On macOS, use Cmd+Q (not just closing the window). Your login session is saved in the Chrome profile and will be reused automatically.

### "No permit/vehicle/address grid found"

The portal uses AJAX to load content. ParkBot waits up to 15 seconds for grids to appear. If the portal structure has changed, the grid detection selectors may need updating.

### GUI does not display

- **Linux**: Ensure OpenGL libraries are installed (see Requirements above)
- **Windows**: Ensure a GCC compiler is available for CGo
- **macOS**: Ensure Xcode Command Line Tools are installed

### Bot stops mid-flow

Check the Activity Log for the specific error. Common causes:
- Network timeout (retry by clicking RUN again)
- Portal UI changed (selectors may need updating)
- Maximum permits reached for the vehicle

---

## Known Issues

1. **Chrome must be quit first**: The bot cannot connect to Chrome that is already running without a debug port
2. **Lock file is relative**: `purchased.lock` is created in the current working directory
3. **Single permit per run**: The bot handles one permit purchase per execution
4. **Payment processor assumptions**: Billing form fill relies on common HTML field naming conventions
5. **macOS linker warning**: `ld: warning: ignoring duplicate libraries: '-lobjc'` is cosmetic and harmless

---

## Development

### Running Tests

```bash
go test -v ./...
```

### Building for All Platforms

```bash
# macOS (native)
go build -o parkbot .

# Windows (cross-compile from macOS/Linux)
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o parkbot.exe .

# Linux (cross-compile from macOS)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-gnu-gcc go build -o parkbot-linux .
```

Note: Cross-compilation requires the appropriate C cross-compiler because Fyne uses CGo.

### Project Structure

```
parkbot/
  main.go       - Entry point, loads config
  gui.go        - Fyne GUI (theme, layout, event handlers)
  config.go     - Config loading, saving, validation, platform detection
  bot.go        - Chrome automation logic (go-rod)
  config_test.go - Unit tests for config and validation
  config.yaml   - User configuration (gitignored)
  purchased.lock - Lock file after purchase (gitignored)
```

---

## License

Private project. Not for redistribution.
