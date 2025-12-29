package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to load config: %v\n", err)
		config = NewDefaultConfig()
	}

	// Create proxy manager
	pm := NewProxyManager(config)
	pm.AddLogExternal(LogLevelInfo, "Quotio TUI started")

	// Auto-start if configured
	if config.AutoStart {
		if err := pm.Start(); err != nil {
			pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to auto-start: %v", err))
		}
	}

	// Create tview application
	app := tview.NewApplication()

	// Create screens
	dashboardScreen := NewDashboardScreen(pm)
	quotaScreen := NewQuotaScreen(pm)
	providersScreen := NewProvidersScreen(pm)
	agentsScreen := NewAgentsScreen()
	apiKeysScreen := NewAPIKeysScreen(pm, config)
	logsScreen := NewLogsScreen(pm)
	settingsScreen := NewSettingsScreen(pm, config, app)

	// Store screens
	screens := map[string]Screen{
		"dashboard": dashboardScreen,
		"quota":     quotaScreen,
		"providers": providersScreen,
		"agents":    agentsScreen,
		"apikeys":   apiKeysScreen,
		"logs":      logsScreen,
		"settings":  settingsScreen,
	}

	// Create sidebar navigation
	sidebar := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)

	sidebar.
		AddItem("📊 Dashboard (d)", "", 'd', nil).
		AddItem("📈 Quota (q)", "", 'q', nil).
		AddItem("🤖 Providers (p)", "", 'p', nil).
		AddItem("⚙️  Agents (a)", "", 'a', nil).
		AddItem("🔑 API Keys (k)", "", 'k', nil).
		AddItem("📋 Logs (l)", "", 'l', nil).
		AddItem("⚙️  Settings (s)", "", 's', nil).
		AddItem("❌ Quit (x)", "", 'x', nil)

	sidebar.SetBorder(true).SetTitle("Navigation")

	// Content area
	content := tview.NewPages()

	// Add all screens to pages
	content.AddPage("dashboard", dashboardScreen.GetView(), true, true)
	content.AddPage("quota", quotaScreen.GetView(), true, false)
	content.AddPage("providers", providersScreen.GetView(), true, false)
	content.AddPage("agents", agentsScreen.GetView(), true, false)
	content.AddPage("apikeys", apiKeysScreen.GetView(), true, false)
	content.AddPage("logs", logsScreen.GetView(), true, false)
	content.AddPage("settings", settingsScreen.GetView(), true, false)

	// Current screen tracking
	currentScreen := "dashboard"
	sidebar.SetCurrentItem(0)

	// Function to switch screens
	switchScreen := func(screenName string, index int) {
		currentScreen = screenName
		content.SwitchToPage(screenName)
		sidebar.SetCurrentItem(index)

		// Update the screen
		if screen, ok := screens[screenName]; ok {
			screen.Update()
		}

		app.SetFocus(content)
	}

	// Handle sidebar selection
	sidebar.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		switch index {
		case 0:
			switchScreen("dashboard", 0)
		case 1:
			switchScreen("quota", 1)
		case 2:
			switchScreen("providers", 2)
		case 3:
			switchScreen("agents", 3)
		case 4:
			switchScreen("apikeys", 4)
		case 5:
			switchScreen("logs", 5)
		case 6:
			switchScreen("settings", 6)
		case 7:
			app.Stop()
		}
	})

	// Main layout
	mainFlex := tview.NewFlex().
		AddItem(sidebar, 30, 0, true).
		AddItem(content, 0, 1, false)

	// Global key handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Tab to toggle focus
		if event.Key() == tcell.KeyTab {
			if app.GetFocus() == sidebar {
				app.SetFocus(content)
			} else {
				app.SetFocus(sidebar)
			}
			return nil
		}

		// Handle screen shortcuts
		switch event.Rune() {
		case 'd':
			switchScreen("dashboard", 0)
			return nil
		case 'q':
			switchScreen("quota", 1)
			return nil
		case 'p':
			switchScreen("providers", 2)
			return nil
		case 'a':
			switchScreen("agents", 3)
			return nil
		case 'k':
			switchScreen("apikeys", 4)
			return nil
		case 'l':
			switchScreen("logs", 5)
			return nil
		case 's':
			switchScreen("settings", 6)
			return nil
		case 'x':
			app.Stop()
			return nil
		}

		// Screen-specific keys
		if currentScreen == "dashboard" {
			switch event.Rune() {
			case 's', 'S': // Toggle server
				status := pm.GetStatus()
				if status.Running {
					if err := pm.Stop(); err != nil {
						pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to stop: %v", err))
					}
				} else {
					if err := pm.Start(); err != nil {
						pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to start: %v", err))
					}
				}
				dashboardScreen.Update()
				return nil
			case 'r', 'R': // Refresh
				pm.FetchAuthFiles()
				pm.FetchUsageStats()
				dashboardScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "Dashboard refreshed")
				return nil
			}
		}

		if currentScreen == "quota" {
			switch event.Rune() {
			case 'r', 'R': // Refresh
				quotaScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "Quota data refreshed")
				return nil
			}
		}

		if currentScreen == "logs" {
			switch event.Rune() {
			case 'c', 'C': // Clear logs
				pm.ClearLogs()
				logsScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "Logs cleared")
				return nil
			}
		}

		if currentScreen == "apikeys" {
			switch event.Rune() {
			case 'g', 'G': // Generate new key
				newKey := fmt.Sprintf("key_%d_%d", time.Now().Unix(), len(config.APIKeys)+1)
				config.APIKeys = append(config.APIKeys, newKey)
				apiKeysScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "New API key generated")
				return nil
			}
		}

		return event
	})

	// Background refresh ticker
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Refresh data if server is running
			status := pm.GetStatus()
			if status.Running {
				pm.FetchAuthFiles()
				pm.FetchUsageStats()
			}

			// Update current screen
			app.QueueUpdateDraw(func() {
				if screen, ok := screens[currentScreen]; ok {
					screen.Update()
				}
			})
		}
	}()

	// Run the application
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	// Cleanup
	if pm.GetStatus().Running {
		pm.Stop()
	}
	pm.AddLogExternal(LogLevelInfo, "Quotio TUI stopped")
}
