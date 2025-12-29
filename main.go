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
	pm.AddLogExternal(LogLevelInfo, "LazyL2M TUI started")

	// Auto-start if configured and binary is installed
	if config.AutoStart && pm.IsBinaryInstalled() {
		if err := pm.Start(); err != nil {
			pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to auto-start: %v", err))
		}
	} else if config.AutoStart && !pm.IsBinaryInstalled() {
		pm.AddLogExternal(LogLevelWarn, "Auto-start enabled but CLIProxyAPI not installed. Press 'I' on dashboard to install.")
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

	// Create sidebar navigation with enhanced styling
	sidebar := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorDodgerBlue)

	sidebar.
		AddItem(" 📊 Dashboard", "", 'd', nil).
		AddItem(" 📈 Quota", "", 'q', nil).
		AddItem(" 🤖 Providers", "", 'p', nil).
		AddItem(" ⚙️  Agents", "", 'a', nil).
		AddItem(" 🔑 API Keys", "", 'k', nil).
		AddItem(" 📋 Logs", "", 'l', nil).
		AddItem(" 🔧 Settings", "", 0, nil).
		AddItem("", "", 0, nil).
		AddItem(" ❌ Quit", "", 'x', nil)

	sidebar.SetBorder(true).SetTitle(" ☰ Navigation ").SetBorderColor(tcell.ColorDodgerBlue).SetTitleColor(tcell.ColorLightCyan)

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

	// Main layout with styled flex
	mainFlex := tview.NewFlex().
		AddItem(sidebar, 22, 0, true).
		AddItem(content, 0, 1, false)

	// Root pages for modal overlay support
	rootPages := tview.NewPages().
		AddPage("main", mainFlex, true, true)

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
		case 8: // Quit (index 8 because of empty separator at 7)
			showQuitConfirmation(app, pm, rootPages, mainFlex)
		}
	})

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
		}

		// Screen-specific keys
		if currentScreen == "dashboard" {
			switch event.Rune() {
			case 's', 'S': // Toggle server
				if !pm.IsBinaryInstalled() {
					pm.AddLogExternal(LogLevelWarn, "Binary not installed. Press 'I' to install first")
					return nil
				}
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
			case 'i', 'I': // Install binary
				if pm.IsBinaryInstalled() {
					pm.AddLogExternal(LogLevelInfo, "CLIProxyAPI is already installed")
					return nil
				}
				if pm.IsDownloading() {
					pm.AddLogExternal(LogLevelInfo, "Download already in progress...")
					return nil
				}
				pm.AddLogExternal(LogLevelInfo, "Starting CLIProxyAPI installation...")
				go func() {
					if err := pm.DownloadAndInstallBinary(); err != nil {
						pm.AddLogExternal(LogLevelError, fmt.Sprintf("Installation failed: %v", err))
					}
					app.QueueUpdateDraw(func() {
						dashboardScreen.Update()
					})
				}()
				dashboardScreen.Update()
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

		if currentScreen == "providers" {
			if event.Key() == tcell.KeyEnter {
				showProviderDetails(app, pm, providersScreen, rootPages, mainFlex)
				return nil
			}
			switch event.Rune() {
			case 'r', 'R': // Refresh
				pm.FetchAuthFiles()
				providersScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "Providers refreshed")
				return nil
			}
		}

		if currentScreen == "agents" {
			if event.Key() == tcell.KeyEnter {
				showAgentDetails(app, pm, agentsScreen, rootPages, mainFlex)
				return nil
			}
			switch event.Rune() {
			case 'r', 'R': // Refresh
				agentsScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "Agents refreshed")
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
			case 'g', 'G': // Generate new key using secure generation
				newKey, err := GenerateSecureKey()
				if err != nil {
					pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to generate key: %v", err))
					return nil
				}
				config.APIKeys = append(config.APIKeys, newKey)
				apiKeysScreen.Update()
				pm.AddLogExternal(LogLevelInfo, "New secure API key generated")
				return nil
			case 'd', 'D': // Delete selected key
				if len(config.APIKeys) > 0 {
					showDeleteKeyConfirmation(app, pm, config, apiKeysScreen, rootPages, mainFlex)
				}
				return nil
			}
		}

		// Handle 'x' for quit with confirmation
		if event.Rune() == 'x' {
			showQuitConfirmation(app, pm, rootPages, mainFlex)
			return nil
		}

		return event
	})

	// Background refresh ticker
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Increased to 15 seconds per quotio patterns
		defer ticker.Stop()

		for range ticker.C {
			// Refresh data if server is running
			status := pm.GetStatus()
			if status.Running {
				pm.FetchAuthFiles()
				pm.FetchUsageStats()
				pm.FetchQuotaInfo()
			} else {
				// Still scan auth directory even when not running
				pm.FetchAuthFiles()
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
	if err := app.SetRoot(rootPages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	// Cleanup
	if pm.GetStatus().Running {
		pm.Stop()
	}
	pm.AddLogExternal(LogLevelInfo, "LazyL2M TUI stopped")
}

// showQuitConfirmation displays a confirmation modal before quitting
func showQuitConfirmation(app *tview.Application, pm *ProxyManager, rootPages *tview.Pages, mainFlex *tview.Flex) {
	modal := tview.NewModal().
		SetText("Are you sure you want to quit LazyL2M?").
		AddButtons([]string{"Cancel", "Quit"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				if pm.GetStatus().Running {
					pm.Stop()
				}
				app.Stop()
			} else {
				// Remove modal and return to main
				rootPages.RemovePage("modal")
				app.SetFocus(mainFlex)
			}
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	modal.SetTextColor(tcell.ColorWhite)
	modal.SetButtonBackgroundColor(tcell.ColorDodgerBlue)

	rootPages.AddPage("modal", modal, true, true)
}

// showDeleteKeyConfirmation displays a confirmation modal before deleting an API key
func showDeleteKeyConfirmation(app *tview.Application, pm *ProxyManager, config *Config, apiKeysScreen *APIKeysScreen, rootPages *tview.Pages, mainFlex *tview.Flex) {
	selectedIdx := apiKeysScreen.GetSelectedIndex()
	if selectedIdx < 0 || selectedIdx >= len(config.APIKeys) {
		return
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete API Key #%d?\n\nThis action cannot be undone.", selectedIdx+1)).
		AddButtons([]string{"Cancel", "Delete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				// Delete the key
				config.APIKeys = append(config.APIKeys[:selectedIdx], config.APIKeys[selectedIdx+1:]...)
				apiKeysScreen.Update()
				pm.AddLogExternal(LogLevelInfo, fmt.Sprintf("API key #%d deleted", selectedIdx+1))
				SaveConfig(config)
			}
			// Remove modal and return to main
			rootPages.RemovePage("modal")
			app.SetFocus(mainFlex)
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	modal.SetTextColor(tcell.ColorWhite)
	modal.SetButtonBackgroundColor(tcell.ColorIndianRed)

	rootPages.AddPage("modal", modal, true, true)
}

// showProviderDetails displays provider details and connected accounts
func showProviderDetails(app *tview.Application, pm *ProxyManager, providersScreen *ProvidersScreen, rootPages *tview.Pages, mainFlex *tview.Flex) {
	provider, info, count := providersScreen.GetSelectedProvider()
	if provider == "" {
		return
	}

	// Build accounts list
	accounts := providersScreen.GetAccountsForProvider(provider)
	accountsList := ""
	if len(accounts) == 0 {
		accountsList = "\n\nNo accounts connected."
	} else {
		accountsList = "\n\nConnected accounts:\n"
		for _, acc := range accounts {
			statusIcon := "✓"
			if acc.Status != "active" {
				statusIcon = "✗"
			}
			accountsList += fmt.Sprintf("  %s %s (%s)\n", statusIcon, acc.Email, acc.Status)
		}
	}

	modalText := fmt.Sprintf("%s %s\n\n%d account(s) connected%s",
		info.Symbol, info.Name, count, accountsList)

	modal := tview.NewModal().
		SetText(modalText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			rootPages.RemovePage("modal")
			app.SetFocus(mainFlex)
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	modal.SetTextColor(tcell.ColorWhite)
	modal.SetButtonBackgroundColor(tcell.ColorDodgerBlue)

	rootPages.AddPage("modal", modal, true, true)
}

// showAgentDetails displays agent configuration details
func showAgentDetails(app *tview.Application, pm *ProxyManager, agentsScreen *AgentsScreen, rootPages *tview.Pages, mainFlex *tview.Flex) {
	agent := agentsScreen.GetSelectedAgent()
	if agent == nil {
		return
	}

	// Build status info
	installedStatus := "❌ Not Installed"
	if agent.Installed {
		installedStatus = "✅ Installed"
	}

	configuredStatus := "❌ Not Configured"
	if agent.Configured {
		configuredStatus = "✅ Configured"
	}

	// Configuration instructions
	instructions := ""
	if !agent.Installed {
		instructions = fmt.Sprintf("\n\nTo install:\n  Install '%s' CLI tool first.", agent.Command)
	} else if !agent.Configured {
		instructions = fmt.Sprintf("\n\nTo configure:\n  Set environment variables or\n  update %s config to use:\n  %s", agent.Command, pm.GetEndpoint())
	} else {
		instructions = fmt.Sprintf("\n\nEndpoint: %s", pm.GetEndpoint())
	}

	modalText := fmt.Sprintf("%s\n\n%s\n%s%s",
		agent.Name, installedStatus, configuredStatus, instructions)

	modal := tview.NewModal().
		SetText(modalText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			rootPages.RemovePage("modal")
			app.SetFocus(mainFlex)
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	modal.SetTextColor(tcell.ColorWhite)
	modal.SetButtonBackgroundColor(tcell.ColorDodgerBlue)

	rootPages.AddPage("modal", modal, true, true)
}
