package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	portalURL = "https://ctitt-iastate.cticloudhost.com/TickeTrak.WebPortal/sso/Home/Index"
	lockFile  = "purchased.lock"
)

func checkLock() error {
	if _, err := os.Stat(lockFile); err == nil {
		return fmt.Errorf("purchased.lock exists — permit already purchased. Delete the lock file to re-run")
	}
	return nil
}

func writeLock() {
	content := fmt.Sprintf("purchased at %s\n", time.Now().Format(time.RFC3339))
	if err := os.WriteFile(lockFile, []byte(content), 0644); err != nil {
		log.Printf("WARNING: could not write lock file: %v", err)
	} else {
		log.Println("Lock file written: purchased.lock")
	}
}

func runBot(ctx context.Context, cfg *Config) error {
	// Step 1: one-time guard
	if err := checkLock(); err != nil {
		return err
	}

	// Step 2: launch Chrome with user's profile.
	// NewUserMode connects to the already-running Chrome instance (via debug port)
	// or launches a fresh one with the profile. If Chrome is open without a debug port,
	// it must be quit first (session cookies are preserved in the profile directory).
	log.Printf("Launching Chrome with profile: %s", cfg.ChromeProfile)
	l := launcher.NewUserMode().
		UserDataDir(cfg.ChromeProfile).
		Headless(false)

	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launching Chrome: %w\n\nTip: fully quit Chrome (Cmd+Q) and re-run — your login session is saved in the profile and will be reused automatically", err)
	}

	browser := rod.New().ControlURL(url).Context(ctx).MustConnect()
	defer browser.Close()

	page := browser.MustPage(portalURL)
	page.MustSetViewport(1280, 900, 1, false)

	// Screenshot helper on error
	var lastErr error
	defer func() {
		if lastErr != nil {
			if img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{}); err == nil {
				_ = os.WriteFile("error_screenshot.png", img, 0644)
				log.Println("Error screenshot saved: error_screenshot.png")
			}
		}
	}()

	// Step 4: wait for page to finish loading, then log where we landed.
	log.Println("Waiting for portal page to load...")
	page.MustWaitLoad()
	info, _ := page.Info()
	log.Printf("Page URL: %s", info.URL)
	log.Printf("Page title: %s", info.Title)

	// Step 5: dashboard has AJAX-rendered content; poll until "Request New Permit" appears,
	// then click it. If it never appears, assume we're already on the purchase page.
	log.Println("Looking for 'Request New Permit' link (up to 15s for AJAX to render)...")
	if clicked := clickRequestPermit(page, 15*time.Second); clicked {
		log.Println("Clicked 'Request New Permit' — waiting for purchase page to load...")
		page.MustWaitLoad()
		info, _ = page.Info()
		log.Printf("After click — URL: %s", info.URL)
		// Wait for page to settle after navigation
		time.Sleep(2 * time.Second)
	} else {
		log.Println("No 'Request New Permit' link found — assuming already on purchase page.")
	}

	// Dump grid structure for debugging (wait up to 10s for AJAX tables to appear)
	waitForTables(page, 10*time.Second)
	dumpGrids(page)

	// Step 6: select permit row
	log.Printf("Selecting permit matching keyword: %q", cfg.PermitKeyword)
	if err := selectPermit(page, cfg.PermitKeyword); err != nil {
		lastErr = fmt.Errorf("selecting permit: %w", err)
		return lastErr
	}

	// Step 7: select vehicle
	log.Printf("Selecting vehicle matching keyword: %q", cfg.VehicleKeyword)
	if err := selectVehicle(page, cfg.VehicleKeyword); err != nil {
		lastErr = fmt.Errorf("selecting vehicle: %w", err)
		return lastErr
	}

	// Step 8: select address
	log.Printf("Selecting address matching keyword: %q", cfg.AddressKeyword)
	if err := selectAddress(page, cfg.AddressKeyword); err != nil {
		lastErr = fmt.Errorf("selecting address: %w", err)
		return lastErr
	}

	// Step 9: click "Add to Cart"
	log.Println("Clicking 'Add to Cart'...")
	if err := clickProceedButton(page); err != nil {
		lastErr = fmt.Errorf("clicking Add to Cart: %w", err)
		return lastErr
	}

	// Wait for page to settle, then check for any dialog/message
	time.Sleep(2 * time.Second)
	if msg := pageAlertText(page); msg != "" {
		log.Printf("  Page message after Add to Cart: %q", msg)
		msgLow := strings.ToLower(msg)
		if strings.Contains(msgLow, "maximum") && strings.Contains(msgLow, "permit") {
			log.Println("  WARNING: maximum permits reached for this vehicle — proceeding to check cart anyway")
		}
		// Dismiss the dialog and continue
		dismissDialog(page)
		time.Sleep(1 * time.Second)
	}

	// Step 10: navigate to cart, select payment type, click checkout
	log.Println("Going to cart and checking out...")
	if err := goToCheckout(page, cfg.Email); err != nil {
		lastErr = fmt.Errorf("cart checkout: %w", err)
		return lastErr
	}

	// Step 11: fill billing on the payment processor page
	log.Println("Filling billing information...")
	if err := fillBilling(page, &cfg.Billing); err != nil {
		lastErr = fmt.Errorf("filling billing: %w", err)
		return lastErr
	}

	// Step 12: submit
	log.Println("Submitting payment...")
	if err := clickSubmitButton(page); err != nil {
		lastErr = fmt.Errorf("clicking submit: %w", err)
		return lastErr
	}

	// Step 13: wait for confirmation
	log.Println("Waiting for confirmation (up to 30s)...")
	if err := waitForConfirmation(page, 30*time.Second); err != nil {
		lastErr = fmt.Errorf("waiting for confirmation: %w", err)
		return lastErr
	}

	log.Println("Purchase confirmed!")

	// Step 13: write lock file
	if cfg.OneTime {
		writeLock()
	}

	return nil
}

// dumpGrids logs table IDs/classes via JavaScript so it never blocks.
func dumpGrids(page *rod.Page) {
	result, err := page.Eval(`() => {
		const tables = document.querySelectorAll('table');
		return Array.from(tables).map((t, i) => {
			const rows = t.querySelectorAll('tbody tr');
			const firstRow = rows[0] ? rows[0].innerText.replace(/\s+/g,' ').trim().slice(0,120) : '';
			return { i, id: t.id, cls: t.className, rows: rows.length, firstRow };
		});
	}`)
	if err != nil {
		log.Printf("  [debug] dumpGrids JS error: %v", err)
		return
	}
	arr := result.Value.Arr()
	log.Printf("  [debug] found %d table(s) on page", len(arr))
	for _, v := range arr {
		log.Printf("  [debug] table[%d] id=%q class=%q rows=%d firstRow=%q",
			v.Get("i").Int(), v.Get("id").Str(), v.Get("cls").Str(),
			v.Get("rows").Int(), v.Get("firstRow").Str())
	}
}

// selectPermit selects the permit matching keyword from any permit-looking table.
func selectPermit(page *rod.Page, keyword string) error {
	gridIDs := []string{"#DecalGrid", "#PermitGrid", "#permitGrid", "#tblPermit", "#GridPermit"}
	for _, id := range gridIDs {
		rows, _ := page.Elements(id + " tbody tr")
		if len(rows) > 0 {
			log.Printf("  Permit grid found: %s (%d rows)", id, len(rows))
			return selectRowByKeyword(page, id, rows, keyword, 0)
		}
	}
	// Fallback: find any table whose first header contains "permit" or "lot"
	tables, _ := page.Elements("table")
	for i, t := range tables {
		headers, _ := t.Elements("thead th, thead td")
		for _, h := range headers {
			text, _ := h.Text()
			if strings.Contains(strings.ToUpper(text), "LOT") || strings.Contains(strings.ToUpper(text), "TYPE") {
				rows, _ := t.Elements("tbody tr")
				log.Printf("  Using fallback permit table[%d] (%d rows)", i, len(rows))
				return selectRowByKeyword(page, fmt.Sprintf("table[%d]", i), rows, keyword, 0)
			}
		}
	}
	return fmt.Errorf("could not find permit grid (tried: %v)", gridIDs)
}

// selectVehicle selects the vehicle matching keyword.
func selectVehicle(page *rod.Page, keyword string) error {
	gridIDs := []string{"#VehicleGrid", "#vehicleGrid", "#tblVehicle", "#GridVehicle"}
	for _, id := range gridIDs {
		rows, _ := page.Elements(id + " tbody tr")
		if len(rows) > 0 {
			log.Printf("  Vehicle grid found: %s (%d rows)", id, len(rows))
			return selectRowByKeyword(page, id, rows, keyword, 0)
		}
	}
	// Fallback: find table with "vehicle" or "plate" header
	tables, _ := page.Elements("table")
	for i, t := range tables {
		headers, _ := t.Elements("thead th, thead td")
		for _, h := range headers {
			text, _ := h.Text()
			upper := strings.ToUpper(text)
			if strings.Contains(upper, "VEHICLE") || strings.Contains(upper, "PLATE") || strings.Contains(upper, "LICENSE") {
				rows, _ := t.Elements("tbody tr")
				log.Printf("  Using fallback vehicle table[%d] (%d rows)", i, len(rows))
				return selectRowByKeyword(page, fmt.Sprintf("table[%d]", i), rows, keyword, 0)
			}
		}
	}
	return fmt.Errorf("could not find vehicle grid (tried: %v)", gridIDs)
}

// selectAddress selects the address matching keyword.
func selectAddress(page *rod.Page, keyword string) error {
	gridIDs := []string{"#AddressGrid", "#addressGrid", "#tblAddress", "#GridAddress"}
	for _, id := range gridIDs {
		rows, _ := page.Elements(id + " tbody tr")
		if len(rows) > 0 {
			log.Printf("  Address grid found: %s (%d rows)", id, len(rows))
			return selectRowByKeyword(page, id, rows, keyword, 0)
		}
	}
	// Fallback: find table with "address" header
	tables, _ := page.Elements("table")
	for i, t := range tables {
		headers, _ := t.Elements("thead th, thead td")
		for _, h := range headers {
			text, _ := h.Text()
			if strings.Contains(strings.ToUpper(text), "ADDRESS") {
				rows, _ := t.Elements("tbody tr")
				log.Printf("  Using fallback address table[%d] (%d rows)", i, len(rows))
				return selectRowByKeyword(page, fmt.Sprintf("table[%d]", i), rows, keyword, 0)
			}
		}
	}
	return fmt.Errorf("could not find address grid (tried: %v)", gridIDs)
}

// selectRowByKeyword finds a matching row and clicks its radio/checkbox input.
func selectRowByKeyword(page *rod.Page, label string, rows []*rod.Element, keyword string, colIndex int) error {
	for i, row := range rows {
		match := false
		if keyword == "" {
			match = true
		} else {
			text, _ := row.Text()
			if strings.Contains(strings.ToUpper(text), keyword) {
				match = true
			}
		}
		if match {
			log.Printf("  Row %d matched in %s", i+1, label)
			input, err := row.Element("input[type=radio], input[type=checkbox]")
			if err != nil {
				if err2 := row.Click(proto.InputMouseButtonLeft, 1); err2 != nil {
					return fmt.Errorf("clicking row %d in %s: %w", i+1, label, err2)
				}
			} else {
				if err := input.Click(proto.InputMouseButtonLeft, 1); err != nil {
					return fmt.Errorf("clicking input in row %d of %s: %w", i+1, label, err)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("no row matching %q found in %s", keyword, label)
}

// clickRequestPermit polls until "Request New Permit" appears then clicks it via JS.
// Returns true if clicked, false if not found within timeout.
func clickRequestPermit(page *rod.Page, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
			const candidates = document.querySelectorAll('a, button, input[type=submit], input[type=button]');
			for (const el of links = candidates) {
				const text = (el.textContent || el.value || '').trim().toLowerCase();
				if (text.includes('request new permit') || text.includes('request a permit')) {
					el.click();
					return true;
				}
			}
			// Also check hrefs
			for (const a of document.querySelectorAll('a[href]')) {
				const href = a.href.toLowerCase();
				if (href.includes('requestpermit') || href.includes('newpermit') || href.includes('permit/request')) {
					a.click();
					return true;
				}
			}
			return false;
		}`)
		if err == nil && result != nil && result.Value.Bool() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// waitForTables polls until at least one table appears (AJAX content).
func waitForTables(page *rod.Page, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		result, _ := page.Eval(`() => document.querySelectorAll('table').length`)
		if result != nil && result.Value.Int() > 0 {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Println("  [debug] no tables appeared within timeout")
}

// waitForElement polls for a selector until it appears or timeout expires.
func waitForElement(page *rod.Page, selector string, timeout time.Duration) (*rod.Element, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		el, err := page.Element(selector)
		if err == nil && el != nil {
			visible, _ := el.Visible()
			if visible {
				return el, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("element %q not found within %s", selector, timeout)
}

// pageAlertText returns any visible alert/error/dialog message text on the page.
func pageAlertText(page *rod.Page) string {
	result, err := page.Eval(`() => {
		// Check for modal dialogs, alerts, or error messages
		const selectors = [
			'[role=dialog]', '[role=alert]', '.modal', '.alert', '.error',
			'[class*=modal]', '[class*=dialog]', '[class*=alert]', '[class*=error]',
			'[class*=message]', '[class*=popup]', '#errorMessage', '.validation-summary-errors'
		];
		for (const sel of selectors) {
			const els = document.querySelectorAll(sel);
			for (const el of els) {
				const text = el.innerText?.trim();
				if (text && text.length > 0 && el.offsetParent !== null) return text;
			}
		}
		return null;
	}`)
	if err != nil || result == nil || result.Value.Nil() {
		return ""
	}
	return result.Value.Str()
}

// dismissDialog clicks OK/Close/Dismiss buttons in any visible dialog.
func dismissDialog(page *rod.Page) {
	page.Eval(`() => {
		const btns = document.querySelectorAll('button, input[type=button], input[type=submit]');
		for (const btn of btns) {
			const text = (btn.textContent || btn.value || '').trim().toLowerCase();
			if (['ok', 'close', 'dismiss', 'cancel', 'x'].includes(text)) {
				const rect = btn.getBoundingClientRect();
				if (rect.width > 0) { btn.click(); return; }
			}
		}
	}`)
}

// goToCheckout navigates to cart, selects Credit Card payment type, optionally fills
// email, then clicks the checkout/process button to reach the payment processor page.
func goToCheckout(page *rod.Page, email string) error {
	// Click Cart icon in header (badge makes text "Cart\n1" not just "cart")
	result, err := page.Eval(`() => {
		const els = document.querySelectorAll('a, button');
		// Debug: collect all link texts
		const all = Array.from(els).map(el => ({text: el.textContent.trim().slice(0,30), href: el.href}));
		for (const el of els) {
			const text = (el.textContent || '').replace(/\s+/g,' ').trim().toLowerCase();
			const href = (el.href || '').toLowerCase();
			if (text.startsWith('cart') || text.includes('cart') || href.includes('cart') || href.includes('checkout')) {
				el.click();
				return 'clicked:' + text + ' href=' + el.href;
			}
		}
		return 'not-found:' + JSON.stringify(all.slice(0,15));
	}`)
	if err != nil {
		return fmt.Errorf("cart nav JS error: %w", err)
	}
	val := result.Value.Str()
	log.Printf("  Cart search result: %s", truncate(val, 300))
	if strings.HasPrefix(val, "not-found:") {
		return fmt.Errorf("no Cart link found in header — %s", val)
	}
	time.Sleep(2 * time.Second)
	page.MustWaitLoad()
	info, _ := page.Info()
	log.Printf("  Cart page URL: %s", info.URL)

	// Select "Credit Card" in the PaymentType Kendo dropdown using Kendo JS API
	log.Println("  Selecting Credit Card payment type...")
	result1, _ := page.Eval(`() => {
		try {
			// Kendo UI API
			const ddl = jQuery && jQuery('#PaymentType').data('kendoDropDownList');
			if (ddl) {
				const data = ddl.dataSource.data();
				for (let i = 0; i < data.length; i++) {
					const txt = String(data[i].Text || data[i].text || data[i].Name || data[i].name || '').toLowerCase();
					if (txt.includes('credit') || txt.includes('card')) {
						ddl.select(i);
						ddl.trigger('change');
						return 'kendo-selected:' + txt;
					}
				}
				// Log all options for debugging
				return 'kendo-options:' + JSON.stringify(data.toJSON().map(d=>d.Text||d.text||d.Name||d.name));
			}
		} catch(e) {}
		// Fallback: open dropdown and look for credit card option
		const wrap = document.querySelector('.k-dropdown-wrap, [aria-owns*="PaymentType"]');
		if (wrap) wrap.click();
		return 'opened-dropdown';
	}`)
	if result1 != nil {
		log.Printf("  Payment type result: %s", truncate(result1.Value.Str(), 200))
	}
	time.Sleep(800 * time.Millisecond)

	// If dropdown opened, click the Credit Card option in the popup list
	result1b, _ := page.Eval(`() => {
		const items = document.querySelectorAll('.k-popup li, .k-list li, [id*="PaymentType_listbox"] li, ul.k-list-ul li');
		const all = Array.from(items).map(li=>li.textContent.trim());
		for (const li of items) {
			const text = li.textContent.trim().toLowerCase();
			if (text.includes('credit') || text.includes('card')) {
				li.click();
				return 'list-clicked:' + text;
			}
		}
		return 'list-options:' + JSON.stringify(all);
	}`)
	if result1b != nil {
		log.Printf("  List click result: %s", truncate(result1b.Value.Str(), 200))
	}
	time.Sleep(500 * time.Millisecond)

	// Fill email if provided
	if email != "" {
		page.Eval(fmt.Sprintf(`() => {
			const el = document.querySelector('#Emails, input[name="Emails"]');
			if (el) { el.value = %q; el.dispatchEvent(new Event('input',{bubbles:true})); el.dispatchEvent(new Event('change',{bubbles:true})); }
		}`, email))
		log.Printf("  Filled email: %s", email)
	}

	// Check "Accept Agreement" checkbox
	log.Println("  Checking Accept Agreement checkbox...")
	agreementResult, _ := page.Eval(`() => {
		const checks = document.querySelectorAll('input[type=checkbox]');
		for (const c of checks) {
			const label = c.id || c.name || '';
			const nearby = (c.labels && c.labels[0] && c.labels[0].textContent) ||
			               (c.parentElement && c.parentElement.textContent) || '';
			if (label.toLowerCase().includes('agree') || nearby.toLowerCase().includes('agree')) {
				if (!c.checked) { c.checked = true; c.dispatchEvent(new Event('change',{bubbles:true})); }
				return 'checked:' + label;
			}
		}
		// Just check all unchecked checkboxes as fallback
		let count = 0;
		for (const c of checks) { if (!c.checked) { c.checked=true; c.dispatchEvent(new Event('change',{bubbles:true})); count++; } }
		return 'checked-all:' + count;
	}`)
	if agreementResult != nil {
		log.Printf("  Agreement: %s", agreementResult.Value.Str())
	}

	// Click the checkout/process button
	log.Println("  Clicking checkout/process button...")
	result2, err := page.Eval(`() => {
		const keywords = ['check out', 'checkout', 'process', 'pay now', 'proceed to payment'];
		const els = document.querySelectorAll('a, button, input[type=submit], input[type=button]');
		for (const el of els) {
			const text = (el.textContent || el.value || '').trim().toLowerCase();
			for (const kw of keywords) {
				if (text === kw || text.includes(kw)) {
					const rect = el.getBoundingClientRect();
					if (rect.width > 0) { el.click(); return text; }
				}
			}
		}
		// Debug: list all buttons
		return 'not-found:' + JSON.stringify(Array.from(document.querySelectorAll('a,button,input[type=submit],input[type=button]')).map(el=>(el.textContent||el.value||'').trim().slice(0,20)).filter(t=>t));
	}`)
	if err != nil {
		return fmt.Errorf("checkout button JS error: %w", err)
	}
	if result2.Value.Nil() {
		// Dump available buttons for debugging
		btns, _ := page.Eval(`() => Array.from(document.querySelectorAll('a,button,input[type=submit],input[type=button]')).map(el=>(el.textContent||el.value||'').trim()).filter(t=>t.length>0)`)
		log.Printf("  [debug] available buttons: %s", truncate(btns.Value.String(), 300))
		return fmt.Errorf("no checkout/process button found")
	}
	log.Printf("  Clicked checkout: %q", result2.Value.Str())

	// Wait for payment processor page to load
	time.Sleep(3 * time.Second)
	page.MustWaitLoad()
	info, _ = page.Info()
	log.Printf("  Payment page URL: %s", info.URL)

	// Dump inputs for debugging
	inputInfo, _ := page.Eval(`() => Array.from(document.querySelectorAll('input:not([type=hidden]):not([type=radio]):not([type=checkbox])')).map(el=>({name:el.name,id:el.id,type:el.type,placeholder:el.placeholder}))`)
	if inputInfo != nil {
		log.Printf("  [debug] payment page inputs: %s", truncate(inputInfo.Value.String(), 400))
	}
	return nil
}

// clickProceedButton clicks the first visible Next/Continue/Add to Cart button via JS.
func clickProceedButton(page *rod.Page) error {
	result, err := page.Eval(`() => {
		const keywords = ['add to cart', 'next', 'continue', 'proceed', 'checkout'];
		const els = document.querySelectorAll('a, button, input[type=submit], input[type=button]');
		for (const el of els) {
			const text = (el.textContent || el.value || '').trim().toLowerCase();
			for (const kw of keywords) {
				if (text === kw || text.includes(kw)) {
					const rect = el.getBoundingClientRect();
					if (rect.width > 0 && rect.height > 0) {
						el.click();
						return text;
					}
				}
			}
		}
		return null;
	}`)
	if err != nil {
		return fmt.Errorf("JS proceed button error: %w", err)
	}
	if result.Value.Nil() {
		return fmt.Errorf("no proceed button found (Next/Continue/Add to Cart)")
	}
	log.Printf("  Clicked proceed button: %q", result.Value.Str())
	return nil
}

// fillBilling fills billing fields. Tries JS on the main page first (works for Touchnet),
// then falls back to same-origin iframes.
func fillBilling(page *rod.Page, b *Billing) error {
	// Dump all visible inputs for debugging
	inputInfo, _ := page.Eval(`() => {
		return Array.from(document.querySelectorAll('input:not([type=hidden]):not([type=radio]):not([type=checkbox])')).map(el => ({
			name: el.name, id: el.id, type: el.type, placeholder: el.placeholder, autocomplete: el.autocomplete
		}));
	}`)
	if inputInfo != nil {
		log.Printf("  [debug] inputs on page: %s", truncate(inputInfo.Value.String(), 400))
	}

	// Try JS fill on the main page first (Touchnet fields are directly in the page DOM)
	if err := fillBillingJS(page, b); err == nil {
		return nil
	}
	log.Println("  JS fill on main page didn't fill all fields, checking iframes...")

	// Fallback: try same-origin iframes (skip cross-origin ones like reCAPTCHA)
	iframes, _ := page.Elements("iframe")
	if len(iframes) > 0 {
		sameOriginFrames := filterSameOriginIframes(iframes, page)
		if len(sameOriginFrames) > 0 {
			log.Printf("  Trying %d same-origin iframe(s)...", len(sameOriginFrames))
			if err := fillBillingIframe(page, b, sameOriginFrames); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("could not fill billing fields (tried JS and iframes)")
}

// filterSameOriginIframes returns iframes that are likely same-origin (skip reCAPTCHA, Google, etc.).
func filterSameOriginIframes(iframes []*rod.Element, page *rod.Page) []*rod.Element {
	info, _ := page.Info()
	var pageHost string
	if info != nil {
		// Extract host from URL
		u := info.URL
		if idx := strings.Index(u, "://"); idx >= 0 {
			u = u[idx+3:]
		}
		if idx := strings.Index(u, "/"); idx >= 0 {
			u = u[:idx]
		}
		pageHost = u
	}

	crossOriginPatterns := []string{"recaptcha", "google", "gstatic", "doubleclick", "facebook"}
	var filtered []*rod.Element
	for _, iframe := range iframes {
		src, _ := iframe.Attribute("src")
		if src == nil || *src == "" {
			filtered = append(filtered, iframe) // no src = probably same origin
			continue
		}
		crossOrigin := false
		for _, pat := range crossOriginPatterns {
			if strings.Contains(strings.ToLower(*src), pat) {
				crossOrigin = true
				break
			}
		}
		if !crossOrigin && (pageHost == "" || strings.Contains(*src, pageHost)) {
			filtered = append(filtered, iframe)
		}
	}
	return filtered
}

// fillBillingJS fills billing form fields using JavaScript evaluation (non-blocking).
// All values and keywords are embedded directly into the JS string.
func fillBillingJS(page *rod.Page, b *Billing) error {
	type fieldFill struct {
		keywords []string
		value    string
		label    string
	}
	fields := []fieldFill{
		{[]string{"card", "cc-number", "cardnumber", "ccn", "accountnumber"}, b.CardNumber, "card number"},
		{[]string{"expir", "cc-exp", "exp", "expdate"}, b.Expiry, "expiry"},
		{[]string{"cvv", "cvc", "cvv2", "security", "cc-csc"}, b.CVV, "cvv"},
		{[]string{"name", "cc-name", "cardholder", "nameoncard"}, b.Name, "name"},
		{[]string{"zip", "postal", "billing_zip", "billingzip"}, b.Zip, "zip"},
	}

	filled := 0
	for _, f := range fields {
		// Build keyword JSON array
		kwJSON := "["
		for i, kw := range f.keywords {
			if i > 0 {
				kwJSON += ","
			}
			kwJSON += `"` + kw + `"`
		}
		kwJSON += "]"

		// Embed value and keywords directly into the self-invoking JS function
		js := fmt.Sprintf(`() => {
			const value = %q;
			const keywords = %s;
			const inputs = document.querySelectorAll('input:not([type=hidden]):not([type=radio]):not([type=checkbox])');
			for (const inp of inputs) {
				const attrs = [inp.name, inp.id, inp.placeholder, inp.autocomplete].join(' ').toLowerCase();
				for (const kw of keywords) {
					if (attrs.includes(kw)) {
						inp.focus();
						inp.value = value;
						inp.dispatchEvent(new Event('input', {bubbles:true}));
						inp.dispatchEvent(new Event('change', {bubbles:true}));
						return inp.name || inp.id || inp.placeholder || '(filled)';
					}
				}
			}
			return null;
		}`, f.value, kwJSON)

		result, err := page.Eval(js)
		if err == nil && result != nil && !result.Value.Nil() {
			log.Printf("  Filled %s → field=%q", f.label, result.Value.Str())
			filled++
		} else if err != nil {
			log.Printf("  [debug] JS fill %s error: %v", f.label, err)
		}
	}

	if filled == 0 {
		return fmt.Errorf("no billing fields filled via JS")
	}
	log.Printf("  Filled %d/%d billing fields via JS", filled, len(fields))

	// Also fill <select> elements: credit card type, expiry month/year
	if err := fillBillingSelects(page, b); err != nil {
		log.Printf("  [debug] select fill: %v", err)
	}
	return nil
}

// fillBillingSelects handles <select> fields: CC type, expiry month, expiry year.
func fillBillingSelects(page *rod.Page, b *Billing) error {
	// Parse expiry "MM/YY" or "MM/YYYY"
	expMonth, expYear := "", ""
	parts := strings.SplitN(b.Expiry, "/", 2)
	if len(parts) == 2 {
		expMonth = strings.TrimSpace(parts[0])
		yr := strings.TrimSpace(parts[1])
		if len(yr) == 2 {
			yr = "20" + yr
		}
		expYear = yr
	}

	// Detect card type from number prefix
	ccType := "visa"
	if strings.HasPrefix(b.CardNumber, "5") {
		ccType = "master"
	} else if strings.HasPrefix(b.CardNumber, "3") {
		ccType = "amex"
	} else if strings.HasPrefix(b.CardNumber, "6") {
		ccType = "discover"
	}

	js := fmt.Sprintf(`() => {
		const results = [];
		const selects = document.querySelectorAll('select');
		for (const sel of selects) {
			const id = sel.id.toLowerCase();
			const name = sel.name.toLowerCase();
			const opts = Array.from(sel.options).map(o=>o.text.toLowerCase());
			// Credit card type
			if (id.includes('type') || name.includes('type') || name.includes('cc_type') || id.includes('cardtype')) {
				for (let i = 0; i < sel.options.length; i++) {
					if (sel.options[i].text.toLowerCase().includes(%q)) {
						sel.selectedIndex = i;
						sel.dispatchEvent(new Event('change', {bubbles:true}));
						results.push('cctype:'+sel.options[i].text);
						break;
					}
				}
			}
			// Expiry month
			if (id.includes('mm') || name.includes('mm') || id.includes('month') || name.includes('month')) {
				for (let i = 0; i < sel.options.length; i++) {
					const v = sel.options[i].value;
					if (v === %q || v === %q.replace(/^0/,'')) {
						sel.selectedIndex = i;
						sel.dispatchEvent(new Event('change', {bubbles:true}));
						results.push('month:'+v);
						break;
					}
				}
			}
			// Expiry year
			if (id.includes('yy') || name.includes('yy') || id.includes('year') || name.includes('year')) {
				for (let i = 0; i < sel.options.length; i++) {
					const v = sel.options[i].value;
					if (v === %q || v === %q.slice(-2)) {
						sel.selectedIndex = i;
						sel.dispatchEvent(new Event('change', {bubbles:true}));
						results.push('year:'+v);
						break;
					}
				}
			}
		}
		return results.join(', ');
	}`, ccType, expMonth, expMonth, expYear, expYear)

	result, err := page.Eval(js)
	if err != nil {
		return fmt.Errorf("select fill JS error: %w", err)
	}
	log.Printf("  Select fills: %q", result.Value.Str())
	return nil
}

func fillBillingIframe(page *rod.Page, b *Billing, iframes []*rod.Element) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("iframe billing panic: %v", r)
		}
	}()
	for i, frameEl := range iframes {
		src, _ := frameEl.Attribute("src")
		srcStr := ""
		if src != nil {
			srcStr = *src
		}
		log.Printf("  iframe[%d] src=%q", i, srcStr)

		frame, err := frameEl.Frame()
		if err != nil {
			log.Printf("  Could not enter iframe[%d]: %v", i, err)
			continue
		}

		type iframeField struct {
			sel   string
			value string
			label string
		}
		ifields := []iframeField{
			{`input[id*="card" i], input[name*="card" i], input[autocomplete="cc-number"]`, b.CardNumber, "card"},
			{`input[id*="expir" i], input[name*="expir" i], input[autocomplete="cc-exp"]`, b.Expiry, "expiry"},
			{`input[id*="cvv" i], input[name*="cvv" i], input[autocomplete="cc-csc"]`, b.CVV, "cvv"},
			{`input[id*="name" i], input[name*="name" i], input[autocomplete="cc-name"]`, b.Name, "name"},
			{`input[id*="zip" i], input[name*="zip" i], input[autocomplete="postal-code"]`, b.Zip, "zip"},
		}

		filled := 0
		for _, f := range ifields {
			els, _ := frame.Elements(f.sel)
			for _, el := range els {
				visible, _ := el.Visible()
				if visible {
					log.Printf("    Filling %s in iframe[%d]", f.label, i)
					el.MustInput(f.value)
					filled++
					break
				}
			}
		}

		if filled > 0 {
			log.Printf("  Filled %d fields in iframe[%d]", filled, i)
			return nil
		}
	}
	return fmt.Errorf("could not fill billing fields in any iframe")
}

// clickSubmitButton finds and clicks the final submit/continue/pay button via JS.
// Prefers <button type=submit> and <input type=submit> over nav links.
func clickSubmitButton(page *rod.Page) error {
	result, err := page.Eval(`() => {
		const keywords = ['continue', 'submit', 'pay now', 'pay', 'purchase', 'complete purchase', 'place order', 'confirm'];
		// Prefer form submit buttons first
		for (const el of document.querySelectorAll('button[type=submit], input[type=submit]')) {
			const text = (el.textContent || el.value || '').trim().toLowerCase();
			for (const kw of keywords) {
				if (text === kw || text.includes(kw)) {
					const rect = el.getBoundingClientRect();
					if (rect.width > 0 && rect.height > 0) { el.click(); return 'submit:' + text; }
				}
			}
		}
		// Generic buttons
		for (const el of document.querySelectorAll('button, input[type=button]')) {
			const text = (el.textContent || el.value || '').trim().toLowerCase();
			for (const kw of keywords) {
				if (text === kw || text.includes(kw)) {
					const rect = el.getBoundingClientRect();
					if (rect.width > 0 && rect.height > 0) { el.click(); return 'button:' + text; }
				}
			}
		}
		// Fallback: visible links (skip short nav items)
		for (const el of document.querySelectorAll('a')) {
			const text = (el.textContent || '').trim().toLowerCase();
			if (text.length < 3 || text.length > 30) continue;
			for (const kw of keywords) {
				if (text === kw || text.includes(kw)) {
					const rect = el.getBoundingClientRect();
					if (rect.width > 0 && rect.height > 0) { el.click(); return 'link:' + text; }
				}
			}
		}
		const all = Array.from(document.querySelectorAll('button,input[type=submit],input[type=button]')).map(el=>(el.textContent||el.value||'').trim().slice(0,20));
		return 'not-found:' + JSON.stringify(all);
	}`)
	if err != nil {
		return fmt.Errorf("JS submit button error: %w", err)
	}
	val := result.Value.Str()
	if strings.HasPrefix(val, "not-found:") {
		return fmt.Errorf("no submit button found — %s", val)
	}
	log.Printf("  Clicked submit button: %q", result.Value.Str())
	return nil
}

// waitForConfirmation polls for success text on the page using JS (non-blocking per check).
func waitForConfirmation(page *rod.Page, timeout time.Duration) error {
	keywords := []string{"thank you", "receipt", "confirmed", "successfully", "order complete", "purchase complete"}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		result, err := page.Eval(`(keywords) => {
			const body = document.body.innerText.toLowerCase();
			for (const kw of keywords) {
				if (body.includes(kw)) return kw;
			}
			return null;
		}`, keywords)
		if err == nil && result != nil && !result.Value.Nil() {
			log.Printf("  Confirmation text found: %q", result.Value.Str())
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("no confirmation found within %s", timeout)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
