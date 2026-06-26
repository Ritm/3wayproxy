package browser

import (
	"context"
	"fmt"
	"log"
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
	if err := playwright.Install(); err != nil {
		log.Printf("browser: playwright install: %v (using existing browsers if any)", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwright run: %w", err)
	}
	launch := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
		},
	}
	if cfg.ProfileDir != "" {
		launch.Args = append(launch.Args, fmt.Sprintf("--user-data-dir=%s", cfg.ProfileDir))
	}
	browser, err := pw.Chromium.Launch(launch)
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
