package main

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ─── Palette ──────────────────────────────────────────────────────────────────

var (
	palBg      = color.RGBA{R: 0x0F, G: 0x0F, B: 0x0F, A: 0xFF}
	palSurface = color.RGBA{R: 0x1A, G: 0x1A, B: 0x1A, A: 0xFF}
	palBorder  = color.RGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF}
	palAccent  = color.RGBA{R: 0xC8, G: 0x10, B: 0x2E, A: 0xFF}
	palFg      = color.RGBA{R: 0xEE, G: 0xEE, B: 0xEE, A: 0xFF}
	palMuted   = color.RGBA{R: 0x5A, G: 0x5A, B: 0x5A, A: 0xFF}
	palInput   = color.RGBA{R: 0x17, G: 0x17, B: 0x17, A: 0xFF}
	palHover   = color.RGBA{R: 0x24, G: 0x24, B: 0x24, A: 0xFF}
	palSuccess = color.RGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}
	palLogOK   = color.RGBA{R: 0x66, G: 0xBB, B: 0x6A, A: 0xFF}
	palLogErr  = color.RGBA{R: 0xEF, G: 0x53, B: 0x50, A: 0xFF}
	palLogDim  = color.RGBA{R: 0x3A, G: 0x3A, B: 0x3A, A: 0xFF}
	palLogWarn = color.RGBA{R: 0xFF, G: 0xCA, B: 0x28, A: 0xFF}
)

// ─── Theme ───────────────────────────────────────────────────────────────────

const (
	colorLogOK   fyne.ThemeColorName = "log-ok"
	colorLogErr  fyne.ThemeColorName = "log-err"
	colorLogDim  fyne.ThemeColorName = "log-dim"
	colorLogWarn fyne.ThemeColorName = "log-warn"
)

type isuTheme struct{}

func (t isuTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return palBg
	case theme.ColorNameForeground:
		return palFg
	case theme.ColorNamePrimary:
		return palAccent
	case theme.ColorNameFocus:
		return palAccent
	case theme.ColorNameSelection:
		return color.RGBA{R: 0xC8, G: 0x10, B: 0x2E, A: 0x40}
	case theme.ColorNameButton:
		return palSurface
	case theme.ColorNameDisabledButton:
		return color.RGBA{R: 0x13, G: 0x13, B: 0x13, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.RGBA{R: 0x35, G: 0x35, B: 0x35, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return palMuted
	case theme.ColorNameInputBackground:
		return palInput
	case theme.ColorNameInputBorder:
		return palBorder
	case theme.ColorNameHover:
		return palHover
	case theme.ColorNameOverlayBackground:
		return color.RGBA{R: 0x1C, G: 0x1C, B: 0x1C, A: 0xFF}
	case theme.ColorNameMenuBackground:
		return color.RGBA{R: 0x1C, G: 0x1C, B: 0x1C, A: 0xFF}
	case theme.ColorNameHeaderBackground:
		return color.RGBA{R: 0x0A, G: 0x0A, B: 0x0A, A: 0xFF}
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 0x2A, G: 0x2A, B: 0x2A, A: 0xFF}
	case theme.ColorNameSeparator:
		return palBorder
	case theme.ColorNameShadow:
		return color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x80}
	case theme.ColorNameError:
		return palLogErr
	case theme.ColorNameSuccess:
		return palSuccess
	case theme.ColorNameWarning:
		return palLogWarn
	case colorLogOK:
		return palLogOK
	case colorLogErr:
		return palLogErr
	case colorLogDim:
		return palLogDim
	case colorLogWarn:
		return palLogWarn
	}
	return theme.DefaultTheme().Color(n, v)
}

// Font delegates to Fyne's default bundled fonts. Fyne bundles NotoSans and
// NotoMono, so no system font installation is required on any platform. This
// avoids "missing font" issues on minimal Linux installations.
func (t isuTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }
func (t isuTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

// Size returns theme dimension sizes, with minor adjustments on Linux where
// font metrics and widget padding can render differently due to varying DPI
// configurations and display server behavior.
func (t isuTheme) Size(n fyne.ThemeSizeName) float32 {
	base := theme.DefaultTheme().Size(n)
	if runtime.GOOS == "linux" {
		switch n {
		case theme.SizeNamePadding:
			// Slightly more padding to compensate for tighter font metrics
			return base + 1
		case theme.SizeNameInlineIcon:
			// Ensure inline icons align with text on Linux
			return base
		}
	}
	return base
}

// ─── GUI struct ───────────────────────────────────────────────────────────────

type GUI struct {
	app     fyne.App
	win     fyne.Window
	cfgPath string

	// Form fields
	permitE  *widget.Entry
	vehicleE *widget.Entry
	addressE *widget.Entry
	emailE   *widget.Entry
	cardE    *widget.Entry
	expiryE  *widget.Entry
	cvvE     *widget.Entry
	nameE    *widget.Entry
	zipE     *widget.Entry
	oneTimeC *widget.Check
	profileE *widget.Entry

	// Buttons
	runBtn  *widget.Button
	stopBtn *widget.Button

	// Status indicator (canvas.Text for dynamic color)
	statusDot  *canvas.Text
	statusText *canvas.Text

	// Lock banner
	lockBanner fyne.CanvasObject

	// Log
	logRich   *widget.RichText
	logScroll *container.Scroll
	logMu     sync.Mutex

	// Bot state
	running bool
	cancel  context.CancelFunc
	mu      sync.Mutex
}

// ─── Log writer ───────────────────────────────────────────────────────────────

type guiLogWriter struct{ g *GUI }

func (w *guiLogWriter) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	if line == "" {
		return len(p), nil
	}

	colorName := fyne.ThemeColorName(theme.ColorNameForeground)
	low := strings.ToLower(line)
	switch {
	case strings.Contains(low, "[debug]"):
		colorName = colorLogDim
	case strings.Contains(low, "error") || strings.Contains(low, "failed") ||
		strings.Contains(low, "could not") || strings.Contains(low, "panic"):
		colorName = colorLogErr
	case strings.Contains(low, "warning") || strings.Contains(low, "warn") ||
		strings.Contains(low, "maximum"):
		colorName = colorLogWarn
	case strings.Contains(low, "confirmed") || strings.Contains(low, "success") ||
		strings.Contains(low, "saved") || strings.Contains(low, "lock file written"):
		colorName = colorLogOK
	}

	seg := &widget.TextSegment{
		Text: line + "\n",
		Style: widget.RichTextStyle{
			ColorName: colorName,
			Inline:    true,
			TextStyle: fyne.TextStyle{Monospace: true},
			SizeName:  theme.SizeNameCaptionText,
		},
	}

	w.g.logMu.Lock()
	w.g.logRich.Segments = append(w.g.logRich.Segments, seg)
	if len(w.g.logRich.Segments) > 2000 {
		w.g.logRich.Segments = w.g.logRich.Segments[len(w.g.logRich.Segments)-2000:]
	}
	w.g.logMu.Unlock()

	w.g.logRich.Refresh()
	w.g.logScroll.ScrollToBottom()
	return len(p), nil
}

// ─── Linux environment helpers ───────────────────────────────────────────────

// initLinuxEnv sets environment variables needed for correct rendering on Linux.
// It handles the X11 vs Wayland distinction and sets font-related hints to
// ensure consistent text rendering regardless of the user's desktop environment.
func initLinuxEnv() {
	if runtime.GOOS != "linux" {
		return
	}

	// Fyne uses GLFW which supports both X11 and Wayland (via XWayland or native).
	// If FYNE_SCALE is not set, leave DPI scaling to the toolkit defaults.
	// Users can override with FYNE_SCALE=1.0 (or any float) if needed.

	// Ensure DISPLAY is set for X11 sessions; Wayland sessions typically set
	// WAYLAND_DISPLAY. If neither is set, default to :0 for X11 as a fallback.
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		os.Setenv("DISPLAY", ":0")
		log.Println("[gui] Warning: neither DISPLAY nor WAYLAND_DISPLAY set, defaulting DISPLAY=:0")
	}

	// Log which display server protocol is in use for debugging.
	if wd := os.Getenv("WAYLAND_DISPLAY"); wd != "" {
		log.Printf("[gui] Wayland session detected (WAYLAND_DISPLAY=%s)", wd)
		// Fyne GLFW on Wayland may run via XWayland. This is fine for rendering.
		// Native Wayland support depends on Fyne/GLFW version.
	} else if d := os.Getenv("DISPLAY"); d != "" {
		log.Printf("[gui] X11 session detected (DISPLAY=%s)", d)
	}
}

// platformWindowSize returns the initial window size adjusted for the platform.
// On Linux with high DPI, Fyne's canvas scaling handles the size, but we use a
// slightly smaller default to fit more reliably on smaller screens and tiling
// window managers common in Linux desktop environments.
func platformWindowSize() fyne.Size {
	if runtime.GOOS == "linux" {
		return fyne.NewSize(1050, 680)
	}
	return fyne.NewSize(1100, 720)
}

// ─── Entry point ─────────────────────────────────────────────────────────────

func runGUI(cfgPath string, cfg *Config) {
	initLinuxEnv()

	a := app.New()
	a.Settings().SetTheme(isuTheme{})
	g := &GUI{app: a, cfgPath: cfgPath}
	g.win = a.NewWindow("ParkBot")
	g.win.Resize(platformWindowSize())
	g.win.SetMaster()
	g.buildUI(cfg)
	log.SetOutput(io.MultiWriter(os.Stderr, &guiLogWriter{g: g}))
	g.win.ShowAndRun()
}

// ─── Layout assembly ──────────────────────────────────────────────────────────

func (g *GUI) buildUI(cfg *Config) {
	g.initEntries(cfg)
	g.initButtons()

	split := container.NewHSplit(g.buildFormPanel(), g.buildLogPanel())
	split.Offset = 0.38

	g.lockBanner = g.buildLockBanner()
	if _, err := os.Stat(lockFile); err != nil {
		g.lockBanner.Hide()
	}

	top := container.NewVBox(g.buildHeader(), g.lockBanner)
	g.win.SetContent(container.NewBorder(top, nil, nil, nil, split))
}

func (g *GUI) initEntries(cfg *Config) {
	ne := func(ph string) *widget.Entry {
		e := widget.NewEntry()
		e.SetPlaceHolder(ph)
		return e
	}

	g.permitE = ne("e.g. COMMUTER")
	g.vehicleE = ne("leave empty for first vehicle")
	g.addressE = ne("leave empty for first address")
	g.emailE = ne("receipt@email.com")
	g.cardE = ne("card number")
	g.expiryE = ne("MM/YY")
	g.cvvE = widget.NewPasswordEntry()
	g.cvvE.SetPlaceHolder("CVV")
	g.nameE = ne("name on card")
	g.zipE = ne("billing ZIP")
	g.oneTimeC = widget.NewCheck("Write lock file after purchase (prevents double-buy)", nil)
	g.profileE = ne(defaultChromeProfile())

	g.permitE.SetText(cfg.PermitKeyword)
	g.vehicleE.SetText(cfg.VehicleKeyword)
	g.addressE.SetText(cfg.AddressKeyword)
	g.emailE.SetText(cfg.Email)
	g.cardE.SetText(cfg.Billing.CardNumber)
	g.expiryE.SetText(cfg.Billing.Expiry)
	g.cvvE.SetText(cfg.Billing.CVV)
	g.nameE.SetText(cfg.Billing.Name)
	g.zipE.SetText(cfg.Billing.Zip)
	g.oneTimeC.SetChecked(cfg.OneTime)
	if cfg.ChromeProfile != "" {
		g.profileE.SetText(cfg.ChromeProfile)
	}
}

func (g *GUI) initButtons() {
	g.runBtn = widget.NewButton("RUN", g.onRun)
	g.runBtn.Importance = widget.HighImportance

	g.stopBtn = widget.NewButton("STOP", g.onStop)
	g.stopBtn.Importance = widget.DangerImportance
	g.stopBtn.Disable()
}

// ─── Header ───────────────────────────────────────────────────────────────────

func (g *GUI) buildHeader() fyne.CanvasObject {
	title := canvas.NewText("PARKBOT", palFg)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 18

	// U+2022 BULLET - safe to use because Fyne bundles NotoSans which includes it.
	g.statusDot = canvas.NewText("\u2022", palMuted)
	g.statusDot.TextSize = 20

	g.statusText = canvas.NewText("IDLE", palMuted)
	g.statusText.TextStyle = fyne.TextStyle{Bold: true}
	g.statusText.TextSize = 11

	row := container.NewBorder(nil, nil,
		container.NewHBox(title),
		container.NewHBox(g.statusDot, g.statusText),
	)

	bg := canvas.NewRectangle(color.RGBA{R: 0x0A, G: 0x0A, B: 0x0A, A: 0xFF})
	accentLine := &canvas.Rectangle{FillColor: palAccent}
	accentLine.SetMinSize(fyne.NewSize(0, 2))

	return container.NewVBox(
		container.NewStack(bg, container.NewPadded(row)),
		accentLine,
	)
}

// ─── Form panel ───────────────────────────────────────────────────────────────

func (g *GUI) buildFormPanel() fyne.CanvasObject {
	permitItem := widget.NewFormItem("Permit keyword", g.permitE)
	permitItem.HintText = "Required. Case-insensitive."
	vehicleItem := widget.NewFormItem("Vehicle keyword", g.vehicleE)
	vehicleItem.HintText = "Empty = first vehicle"
	addressItem := widget.NewFormItem("Address keyword", g.addressE)
	addressItem.HintText = "Empty = first address"

	permitSection := makeSection("PERMIT SELECTION",
		widget.NewForm(
			permitItem,
			vehicleItem,
			addressItem,
			widget.NewFormItem("Email", g.emailE),
		),
	)

	billingSection := makeSection("BILLING",
		widget.NewForm(
			widget.NewFormItem("Card number", g.cardE),
			widget.NewFormItem("Expiry", g.expiryE),
			widget.NewFormItem("CVV", g.cvvE),
			widget.NewFormItem("Name", g.nameE),
			widget.NewFormItem("ZIP", g.zipE),
		),
	)

	optionsSection := makeSection("OPTIONS",
		widget.NewForm(
			widget.NewFormItem("", g.oneTimeC),
			widget.NewFormItem("Chrome profile", g.profileE),
		),
	)

	saveBtn := widget.NewButton("SAVE CONFIG", g.onSave)
	saveBtn.Importance = widget.LowImportance

	btnRow := container.NewGridWithColumns(3, saveBtn, g.runBtn, g.stopBtn)
	formContent := container.NewVBox(permitSection, billingSection, optionsSection)

	return container.NewBorder(
		nil,
		container.NewPadded(btnRow),
		nil, nil,
		container.NewScroll(formContent),
	)
}

// ─── Log panel ────────────────────────────────────────────────────────────────

func (g *GUI) buildLogPanel() fyne.CanvasObject {
	g.logRich = widget.NewRichText()
	g.logRich.Wrapping = fyne.TextWrapWord
	g.logScroll = container.NewScroll(g.logRich)

	logTitle := canvas.NewText("ACTIVITY LOG", palMuted)
	logTitle.TextStyle = fyne.TextStyle{Bold: true}
	logTitle.TextSize = 10

	clearBtn := widget.NewButton("CLEAR", func() {
		g.logMu.Lock()
		g.logRich.Segments = nil
		g.logMu.Unlock()
		g.logRich.Refresh()
	})
	clearBtn.Importance = widget.LowImportance

	toolbar := container.NewBorder(nil, nil,
		container.NewPadded(logTitle),
		clearBtn,
	)

	termBg := canvas.NewRectangle(color.RGBA{R: 0x09, G: 0x09, B: 0x09, A: 0xFF})
	logArea := container.NewStack(termBg, g.logScroll)

	return container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		nil, nil, nil,
		logArea,
	)
}

// ─── Lock banner ─────────────────────────────────────────────────────────────

func (g *GUI) buildLockBanner() fyne.CanvasObject {
	// Use ASCII dash instead of em dash for broader font compatibility on Linux.
	msg := canvas.NewText(
		"LOCK FILE EXISTS - permit already purchased. Remove the lock to run again.",
		palLogErr,
	)
	msg.TextStyle = fyne.TextStyle{Bold: true}
	msg.TextSize = 12

	removeBtn := widget.NewButton("REMOVE LOCK", func() {
		if err := os.Remove(lockFile); err != nil {
			dialog.ShowError(fmt.Errorf("could not remove lock file: %w", err), g.win)
			return
		}
		g.lockBanner.Hide()
		g.win.Content().Refresh()
		log.Println("Lock file removed.")
	})
	removeBtn.Importance = widget.DangerImportance

	inner := container.NewBorder(nil, nil, nil, removeBtn, container.NewPadded(msg))
	bg := &canvas.Rectangle{
		FillColor:    color.RGBA{R: 0x28, G: 0x08, B: 0x08, A: 0xFF},
		StrokeColor:  palAccent,
		StrokeWidth:  1,
		CornerRadius: 0,
	}
	return container.NewStack(bg, container.NewPadded(inner))
}

// ─── makeSection ─────────────────────────────────────────────────────────────

func makeSection(title string, content fyne.CanvasObject) fyne.CanvasObject {
	lbl := canvas.NewText(title, palMuted)
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.TextSize = 10

	accentBar := &canvas.Rectangle{FillColor: palAccent, CornerRadius: 1}
	accentBar.SetMinSize(fyne.NewSize(2, 0))

	inner := container.NewVBox(container.NewPadded(lbl), content)
	innerWithAccent := container.NewBorder(nil, nil, accentBar, nil, inner)

	bg := &canvas.Rectangle{FillColor: palSurface, CornerRadius: 4}
	return container.NewPadded(container.NewStack(bg, container.NewPadded(innerWithAccent)))
}

// ─── Config helpers ───────────────────────────────────────────────────────────

func (g *GUI) readFormConfig() *Config {
	return &Config{
		PermitKeyword:  strings.ToUpper(strings.TrimSpace(g.permitE.Text)),
		VehicleKeyword: strings.ToUpper(strings.TrimSpace(g.vehicleE.Text)),
		AddressKeyword: strings.ToUpper(strings.TrimSpace(g.addressE.Text)),
		Email:          strings.TrimSpace(g.emailE.Text),
		OneTime:        g.oneTimeC.Checked,
		ChromeProfile:  strings.TrimSpace(g.profileE.Text),
		Billing: Billing{
			CardNumber: strings.ReplaceAll(g.cardE.Text, " ", ""),
			Expiry:     strings.TrimSpace(g.expiryE.Text),
			CVV:        strings.TrimSpace(g.cvvE.Text),
			Name:       strings.TrimSpace(g.nameE.Text),
			Zip:        strings.TrimSpace(g.zipE.Text),
		},
	}
}

func (g *GUI) onSave() {
	cfg := g.readFormConfig()
	if cfg.ChromeProfile == "" {
		cfg.ChromeProfile = defaultChromeProfile()
	}
	if err := cfg.Save(g.cfgPath); err != nil {
		dialog.ShowError(err, g.win)
		return
	}
	log.Printf("Config saved to %s", g.cfgPath)
}

// ─── Bot lifecycle ────────────────────────────────────────────────────────────

func (g *GUI) onRun() {
	cfg := g.readFormConfig()
	if cfg.ChromeProfile == "" {
		cfg.ChromeProfile = defaultChromeProfile()
	}
	if err := cfg.validate(); err != nil {
		dialog.ShowError(fmt.Errorf("configuration error: %w", err), g.win)
		return
	}

	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.running = true
	g.mu.Unlock()

	g.setStatus(true)
	log.Println("Bot started.")

	go func() {
		err := runBot(ctx, cfg)

		g.mu.Lock()
		g.running = false
		g.cancel = nil
		g.mu.Unlock()

		if err != nil {
			log.Printf("Bot error: %v", err)
		} else {
			log.Println("Bot finished successfully.")
		}

		if _, statErr := os.Stat(lockFile); statErr == nil {
			g.lockBanner.Show()
			g.win.Content().Refresh()
		}

		g.setStatus(false)
	}()
}

func (g *GUI) onStop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cancel != nil {
		g.cancel()
		log.Println("Stop requested.")
	}
}

func (g *GUI) setStatus(running bool) {
	if running {
		g.statusDot.Color = palLogOK
		g.statusText.Color = palLogOK
		g.statusText.Text = "RUNNING"
		g.runBtn.Disable()
		g.stopBtn.Enable()
	} else {
		g.statusDot.Color = palMuted
		g.statusText.Color = palMuted
		g.statusText.Text = "IDLE"
		g.stopBtn.Disable()
		g.runBtn.Enable()
	}
	g.statusDot.Refresh()
	g.statusText.Refresh()
}
