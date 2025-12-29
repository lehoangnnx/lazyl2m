package main

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// Screen is an interface for all screens
type Screen interface {
	GetView() tview.Primitive
	Update()
}

// DashboardScreen shows server status and statistics
type DashboardScreen struct {
	view         *tview.Flex
	statusText   *tview.TextView
	statsText    *tview.TextView
	accountsText *tview.TextView
	pm           *ProxyManager
}

func NewDashboardScreen(pm *ProxyManager) *DashboardScreen {
	ds := &DashboardScreen{pm: pm}
	
	// Title
	title := tview.NewTextView().
		SetText("[::b]Dashboard[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	// Server status section
	ds.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	
	statusBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("[yellow::b]Server Status[::-]").SetDynamicColors(true), 1, 0, false).
		AddItem(ds.statusText, 0, 1, false)
	
	// Usage statistics section
	ds.statsText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	
	statsBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("[yellow::b]Usage Statistics[::-]").SetDynamicColors(true), 1, 0, false).
		AddItem(ds.statsText, 0, 1, false)
	
	// Connected accounts section
	ds.accountsText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	
	accountsBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("[yellow::b]Connected Accounts[::-]").SetDynamicColors(true), 1, 0, false).
		AddItem(ds.accountsText, 0, 1, false)
	
	// Help text
	help := tview.NewTextView().
		SetText("[gray]Press 's' to toggle server | 'r' to refresh | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	// Main layout
	ds.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(statusBox, 5, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(statsBox, 8, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(accountsBox, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	ds.Update()
	return ds
}

func (ds *DashboardScreen) GetView() tview.Primitive {
	return ds.view
}

func (ds *DashboardScreen) Update() {
	status := ds.pm.GetStatus()
	stats := ds.pm.GetUsageStats()
	authFiles := ds.pm.GetAuthFiles()
	
	// Update status
	statusColor := "[red]"
	statusText := "Stopped"
	if status.Running {
		statusColor = "[green]"
		statusText = "Running"
	}
	ds.statusText.SetText(fmt.Sprintf(
		"%s● %s[-]\nPort: [white]%d[-]",
		statusColor, statusText, status.Port,
	))
	
	// Update statistics
	ds.statsText.SetText(fmt.Sprintf(
		"Total Requests:   [white]%d[-]\n"+
		"Success Requests: [green]%d[-]\n"+
		"Failed Requests:  [red]%d[-]\n"+
		"Total Tokens:     [cyan]%d[-]\n"+
		"Success Rate:     [yellow]%.1f%%[-]\n"+
		"Last Updated:     [gray]%s[-]",
		stats.TotalRequests,
		stats.SuccessRequests,
		stats.FailedRequests,
		stats.TotalTokens,
		stats.SuccessRate,
		stats.LastUpdated.Format("15:04:05"),
	))
	
	// Update accounts
	accountsList := ""
	for _, auth := range authFiles {
		info := GetProviderInfo(auth.Provider)
		statusColor := "[green]"
		if auth.Status != "active" {
			statusColor = "[red]"
		}
		accountsList += fmt.Sprintf("%s %s - %s%s[-]\n", info.Symbol, auth.Name, statusColor, auth.Status)
	}
	if accountsList == "" {
		accountsList = "[gray]No connected accounts[-]"
	}
	ds.accountsText.SetText(accountsList)
}

// QuotaScreen shows quota usage for all accounts
type QuotaScreen struct {
	view  *tview.Flex
	table *tview.Table
	pm    *ProxyManager
}

func NewQuotaScreen(pm *ProxyManager) *QuotaScreen {
	qs := &QuotaScreen{pm: pm}
	
	title := tview.NewTextView().
		SetText("[::b]Quota Usage[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	qs.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)
	
	help := tview.NewTextView().
		SetText("[gray]Press 'r' to refresh | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	qs.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(qs.table, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	qs.Update()
	return qs
}

func (qs *QuotaScreen) GetView() tview.Primitive {
	return qs.view
}

func (qs *QuotaScreen) Update() {
	qs.table.Clear()
	
	// Headers
	headers := []string{"Provider", "Account", "Used", "Limit", "Usage %", "Status", "Reset Time"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s[::-]", header)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		qs.table.SetCell(0, col, cell)
	}
	
	// Data rows
	quotas := qs.pm.GetQuotaInfos()
	for row, quota := range quotas {
		info := GetProviderInfo(quota.Provider)
		
		// Status color
		statusColor := "[green]"
		if quota.Status == "warning" {
			statusColor = "[yellow]"
		} else if quota.Status == "exceeded" {
			statusColor = "[red]"
		}
		
		// Reset time
		resetTime := "N/A"
		if quota.ResetTime != nil {
			resetTime = quota.ResetTime.Format("2006-01-02 15:04")
		}
		
		cells := []string{
			fmt.Sprintf("%s %s", info.Symbol, info.Name),
			quota.AccountName,
			fmt.Sprintf("%d", quota.Used),
			fmt.Sprintf("%d", quota.Limit),
			fmt.Sprintf("%.1f%%", quota.UsagePercent),
			fmt.Sprintf("%s%s[-]", statusColor, quota.Status),
			resetTime,
		}
		
		for col, text := range cells {
			cell := tview.NewTableCell(text).
				SetAlign(tview.AlignLeft).
				SetSelectable(false)
			qs.table.SetCell(row+1, col, cell)
		}
	}
}

// ProvidersScreen shows all supported providers
type ProvidersScreen struct {
	view *tview.Flex
	list *tview.List
	pm   *ProxyManager
}

func NewProvidersScreen(pm *ProxyManager) *ProvidersScreen {
	ps := &ProvidersScreen{pm: pm}
	
	title := tview.NewTextView().
		SetText("[::b]AI Providers[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ps.list = tview.NewList()
	
	help := tview.NewTextView().
		SetText("[gray]Press Enter to manage accounts | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ps.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(ps.list, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	ps.Update()
	return ps
}

func (ps *ProvidersScreen) GetView() tview.Primitive {
	return ps.view
}

func (ps *ProvidersScreen) Update() {
	ps.list.Clear()
	
	providers := GetAllProviders()
	authFiles := ps.pm.GetAuthFiles()
	
	for _, provider := range providers {
		info := GetProviderInfo(provider)
		
		// Count accounts for this provider
		count := 0
		for _, auth := range authFiles {
			if auth.Provider == provider {
				count++
			}
		}
		
		mainText := fmt.Sprintf("%s %s", info.Symbol, info.Name)
		secondaryText := fmt.Sprintf("%d account(s)", count)
		
		ps.list.AddItem(mainText, secondaryText, 0, nil)
	}
}

// AgentsScreen shows CLI agent configuration status
type AgentsScreen struct {
	view  *tview.Flex
	table *tview.Table
}

func NewAgentsScreen() *AgentsScreen {
	as := &AgentsScreen{}
	
	title := tview.NewTextView().
		SetText("[::b]CLI Agents[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	as.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	
	help := tview.NewTextView().
		SetText("[gray]Press Enter to configure | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	as.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(as.table, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	as.Update()
	return as
}

func (as *AgentsScreen) GetView() tview.Primitive {
	return as.view
}

func (as *AgentsScreen) Update() {
	as.table.Clear()
	
	// Headers
	headers := []string{"Agent Name", "Installed", "Configured"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s[::-]", header)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		as.table.SetCell(0, col, cell)
	}
	
	// Data rows
	agents := GetAllAgents()
	for row, agent := range agents {
		installedText := "[red]✗ No[-]"
		if agent.Installed {
			installedText = "[green]✓ Yes[-]"
		}
		
		configuredText := "[red]✗ No[-]"
		if agent.Configured {
			configuredText = "[green]✓ Yes[-]"
		}
		
		cells := []string{
			agent.Name,
			installedText,
			configuredText,
		}
		
		for col, text := range cells {
			cell := tview.NewTableCell(text).
				SetAlign(tview.AlignLeft)
			as.table.SetCell(row+1, col, cell)
		}
	}
}

// APIKeysScreen shows API key management
type APIKeysScreen struct {
	view *tview.Flex
	list *tview.List
	pm   *ProxyManager
	cfg  *Config
}

func NewAPIKeysScreen(pm *ProxyManager, cfg *Config) *APIKeysScreen {
	aks := &APIKeysScreen{pm: pm, cfg: cfg}
	
	title := tview.NewTextView().
		SetText("[::b]API Keys[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	aks.list = tview.NewList()
	
	help := tview.NewTextView().
		SetText("[gray]Press 'g' to generate new key | 'd' to delete | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	aks.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(aks.list, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	aks.Update()
	return aks
}

func (aks *APIKeysScreen) GetView() tview.Primitive {
	return aks.view
}

func (aks *APIKeysScreen) Update() {
	aks.list.Clear()
	
	if len(aks.cfg.APIKeys) == 0 {
		aks.list.AddItem("[gray]No API keys generated[-]", "", 0, nil)
		return
	}
	
	for i, key := range aks.cfg.APIKeys {
		// Mask the key for display
		masked := key
		if len(key) > 12 {
			masked = key[:8] + "..." + key[len(key)-4:]
		}
		
		mainText := fmt.Sprintf("Key %d: %s", i+1, masked)
		aks.list.AddItem(mainText, "", 0, nil)
	}
}

// LogsScreen shows application logs
type LogsScreen struct {
	view     *tview.Flex
	textView *tview.TextView
	pm       *ProxyManager
}

func NewLogsScreen(pm *ProxyManager) *LogsScreen {
	ls := &LogsScreen{pm: pm}
	
	title := tview.NewTextView().
		SetText("[::b]Logs[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ls.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			// Auto-scroll to bottom
			ls.textView.ScrollToEnd()
		})
	
	help := tview.NewTextView().
		SetText("[gray]Press 'c' to clear logs | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ls.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(ls.textView, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	ls.Update()
	return ls
}

func (ls *LogsScreen) GetView() tview.Primitive {
	return ls.view
}

func (ls *LogsScreen) Update() {
	logs := ls.pm.GetLogs()
	
	var logText strings.Builder
	for _, log := range logs {
		color := "white"
		switch log.Level {
		case LogLevelInfo:
			color = "white"
		case LogLevelWarn:
			color = "yellow"
		case LogLevelError:
			color = "red"
		case LogLevelDebug:
			color = "gray"
		}
		
		logText.WriteString(fmt.Sprintf(
			"[gray]%s[-] [%s]%5s[-] %s\n",
			log.Timestamp.Format("15:04:05"),
			color,
			strings.ToUpper(string(log.Level)),
			log.Message,
		))
	}
	
	if logText.Len() == 0 {
		logText.WriteString("[gray]No logs available[-]")
	}
	
	ls.textView.SetText(logText.String())
}

// SettingsScreen shows configuration form
type SettingsScreen struct {
	view *tview.Flex
	form *tview.Form
	pm   *ProxyManager
	cfg  *Config
	app  *tview.Application
}

func NewSettingsScreen(pm *ProxyManager, cfg *Config, app *tview.Application) *SettingsScreen {
	ss := &SettingsScreen{pm: pm, cfg: cfg, app: app}
	
	title := tview.NewTextView().
		SetText("[::b]Settings[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ss.form = tview.NewForm()
	ss.buildForm()
	
	help := tview.NewTextView().
		SetText("[gray]Use arrow keys to navigate | Enter to edit | Tab to switch focus[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	ss.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(ss.form, 0, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)
	
	return ss
}

func (ss *SettingsScreen) GetView() tview.Primitive {
	return ss.view
}

func (ss *SettingsScreen) Update() {
	// Settings don't need dynamic updates
}

func (ss *SettingsScreen) buildForm() {
	ss.form.Clear(true)
	
	// Port
	ss.form.AddInputField("Port", fmt.Sprintf("%d", ss.cfg.Port), 20, nil, func(text string) {
		var port int
		fmt.Sscanf(text, "%d", &port)
		if port > 0 && port < 65536 {
			ss.cfg.Port = port
		}
	})
	
	// Routing Strategy
	strategyIndex := 0
	if ss.cfg.RoutingStrategy == RoutingFillFirst {
		strategyIndex = 1
	}
	ss.form.AddDropDown("Routing Strategy", []string{"Round Robin", "Fill First"}, strategyIndex, func(option string, optionIndex int) {
		if optionIndex == 0 {
			ss.cfg.RoutingStrategy = RoutingRoundRobin
		} else {
			ss.cfg.RoutingStrategy = RoutingFillFirst
		}
	})
	
	// Auto-start
	ss.form.AddCheckbox("Auto-start Server", ss.cfg.AutoStart, func(checked bool) {
		ss.cfg.AutoStart = checked
	})
	
	// Debug Mode
	ss.form.AddCheckbox("Debug Mode", ss.cfg.DebugMode, func(checked bool) {
		ss.cfg.DebugMode = checked
	})
	
	// Log to File
	ss.form.AddCheckbox("Log to File", ss.cfg.LogToFile, func(checked bool) {
		ss.cfg.LogToFile = checked
	})
	
	// Usage Stats
	ss.form.AddCheckbox("Usage Statistics", ss.cfg.UsageStatsEnabled, func(checked bool) {
		ss.cfg.UsageStatsEnabled = checked
	})
	
	// Request Retry Count
	ss.form.AddInputField("Request Retry Count", fmt.Sprintf("%d", ss.cfg.RequestRetryCount), 20, nil, func(text string) {
		var count int
		fmt.Sscanf(text, "%d", &count)
		if count >= 0 {
			ss.cfg.RequestRetryCount = count
		}
	})
	
	// Buttons
	ss.form.AddButton("Save", func() {
		if err := SaveConfig(ss.cfg); err != nil {
			ss.pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to save config: %v", err))
		} else {
			ss.pm.AddLogExternal(LogLevelInfo, "Configuration saved successfully")
		}
	})
	
	ss.form.AddButton("Reset", func() {
		ss.cfg = NewDefaultConfig()
		ss.buildForm()
		ss.pm.AddLogExternal(LogLevelInfo, "Configuration reset to defaults")
	})
}
