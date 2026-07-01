package browser

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/playwright-community/playwright-go"
)

// Manager owns one Chromium instance for all relay tabs/contexts.
type Manager struct {
	cfg     Config
	pw      *playwright.Playwright
	browser playwright.Browser
	mu      sync.Mutex
}

func NewManager(ctx context.Context, cfg Config) (*Manager, error) {
	_ = ctx
	prepareSudoEnv()
	// Only the Playwright driver is needed; we launch system Chrome (channel "chrome").
	// Bare Install() downloads firefox + webkit too (~400MB extra).
	if err := playwright.Install(&playwright.RunOptions{SkipInstallBrowsers: true}); err != nil {
		log.Printf("browser: playwright driver install: %v", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwright run: %w", err)
	}
	launch := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
		},
	}
	// Prefer system Chrome — bundled headless_shell sometimes hangs under sudo/VPN.
	if ch := os.Getenv("PLAYWRIGHT_CHROME_CHANNEL"); ch != "" {
		launch.Channel = playwright.String(ch)
	} else {
		launch.Channel = playwright.String("chrome")
	}
	if cfg.ProfileDir != "" {
		launch.Args = append(launch.Args, fmt.Sprintf("--user-data-dir=%s", cfg.ProfileDir))
	}
	browser, err := pw.Chromium.Launch(launch)
	if err != nil {
		log.Printf("browser: system chrome launch failed: %v — fallback to bundled chromium (run scripts/install-chromium.sh once)", err)
		launch.Channel = nil
		browser, err = pw.Chromium.Launch(launch)
	}
	if err != nil {
		_ = pw.Stop()
		return nil, fmt.Errorf("chromium launch: %w", err)
	}
	return &Manager{cfg: cfg, pw: pw, browser: browser}, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browser != nil {
		_ = m.browser.Close()
		m.browser = nil
	}
	if m.pw != nil {
		_ = m.pw.Stop()
		m.pw = nil
	}
}

func (m *Manager) browserInstance() (playwright.Browser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browser == nil {
		return nil, fmt.Errorf("browser: manager closed")
	}
	return m.browser, nil
}
