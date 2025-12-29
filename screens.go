package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
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

	// ASCII Logo Header
	logo := `[#00d7ff::b]
 ██╗      █████╗ ███████╗██╗   ██╗██╗     ██████╗ ███╗   ███╗
 ██║     ██╔══██╗╚══███╔╝╚██╗ ██╔╝██║     ╚════██╗████╗ ████║
 ██║     ███████║  ███╔╝  ╚████╔╝ ██║      █████╔╝██╔████╔██║
 ██║     ██╔══██║ ███╔╝    ╚██╔╝  ██║     ██╔═══╝ ██║╚██╔╝██║
 ███████╗██║  ██║███████╗   ██║   ███████╗███████╗██║ ╚═╝ ██║
 ╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝╚══════╝╚═╝     ╚═╝[-]
 [gray]───────────────────────────────────────────────────────────[-]
 [#87d7ff]          CLIProxyAPI Management Terminal v1.0[-]`

	header := tview.NewTextView().
		SetText(logo).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Server status section with box border
	ds.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)

	statusBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ds.statusText, 0, 1, false)
	statusBox.SetBorder(true).SetTitle(" 🖥️  Server Status ").SetTitleAlign(tview.AlignLeft).SetBorderColor(tcell.ColorDodgerBlue)

	// Usage statistics section with box border
	ds.statsText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)

	statsBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ds.statsText, 0, 1, false)
	statsBox.SetBorder(true).SetTitle(" 📊 Usage Statistics ").SetTitleAlign(tview.AlignLeft).SetBorderColor(tcell.ColorDodgerBlue)

	// Connected accounts section with box border
	ds.accountsText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	accountsBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ds.accountsText, 0, 1, false)
	accountsBox.SetBorder(true).SetTitle(" 👤 Connected Accounts ").SetTitleAlign(tview.AlignLeft).SetBorderColor(tcell.ColorDodgerBlue)

	// Info panels in horizontal layout
	infoRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(statusBox, 0, 1, false).
		AddItem(statsBox, 0, 2, false)

	// Help text with styled shortcuts
	help := tview.NewTextView().
		SetText("[#5f87af]╔══════════════════════════════════════════════════════════════════╗\n║ [#87d7ff]S[-][white] Toggle Server  [#87d7ff]I[-][white] Install  [#87d7ff]R[-][white] Refresh  [#87d7ff]Tab[-][white] Focus  [#87d7ff]X[-][white] Quit [#5f87af]║\n╚══════════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Main layout
	ds.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 10, 0, false).
		AddItem(infoRow, 10, 0, false).
		AddItem(accountsBox, 0, 1, false).
		AddItem(help, 4, 0, false)

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

	// Check if binary is installed
	if !ds.pm.IsBinaryInstalled() {
		if ds.pm.IsDownloading() {
			progress := ds.pm.GetDownloadProgress() * 100
			ds.statusText.SetText(fmt.Sprintf(
				"\n  [yellow]◉ DOWNLOADING[-]\n\n"+
					"  [#87d7ff]Progress:[white] %.0f%%[-]\n"+
					"  [gray]Installing CLIProxyAPI...[-]",
				progress,
			))
		} else {
			ds.statusText.SetText(
				"\n  [red]◉ NOT INSTALLED[-]\n\n" +
					"  [gray]CLIProxyAPI binary not found[-]\n" +
					"  [yellow]Press 'I' to install[-]",
			)
		}
	} else {
		// Update status with visual indicator
		statusIcon := "[red]◉ STOPPED[-]"
		statusDetail := "[gray]Server is not running. Press 'S' to start[-]"
		if status.Running {
			statusIcon = "[green]◉ RUNNING[-]"
			statusDetail = "[white]Accepting connections[-]"
		}
		ds.statusText.SetText(fmt.Sprintf(
			"\n  %s\n\n  [#87d7ff]Port:[white] %d[-]\n  [#87d7ff]Endpoint:[white] %s[-]\n  %s",
			statusIcon, status.Port, ds.pm.GetEndpoint(), statusDetail,
		))
	}

	// Update statistics with visual bars
	successBar := createProgressBar(stats.SuccessRate, 20)
	ds.statsText.SetText(fmt.Sprintf(
		"\n  [#87d7ff]Total Requests:  [-] [white]%d[-]\n"+
			"  [#87d7ff]Success:         [-] [green]%d[-]\n"+
			"  [#87d7ff]Failed:          [-] [red]%d[-]\n"+
			"  [#87d7ff]Total Tokens:    [-] [cyan]%d[-]\n\n"+
			"  [#87d7ff]Success Rate:[-]    %s [yellow]%.1f%%[-]\n\n"+
			"  [gray]Updated: %s[-]",
		stats.TotalRequests,
		stats.SuccessRequests,
		stats.FailedRequests,
		stats.TotalTokens,
		successBar,
		stats.SuccessRate,
		stats.LastUpdated.Format("15:04:05"),
	))

	// Update accounts with table-like formatting
	var accountsList strings.Builder
	accountsList.WriteString("\n")
	if len(authFiles) == 0 {
		accountsList.WriteString("  [gray]No connected accounts. Start the server to sync.[-]\n")
	} else {
		for i, auth := range authFiles {
			info := GetProviderInfo(auth.Provider)
			statusColor := "[green]●[-]"
			statusText := "active"
			if auth.Status != "active" {
				statusColor = "[red]●[-]"
				statusText = auth.Status
			}
			accountsList.WriteString(fmt.Sprintf(
				"  %s %s %-20s %s [gray]%s[-]\n",
				statusColor, info.Symbol, auth.Name, fmt.Sprintf("[#5f87af](%s)[-]", info.Name), statusText,
			))
			if i < len(authFiles)-1 {
				accountsList.WriteString("  [#303030]────────────────────────────────────────[-]\n")
			}
		}
	}
	ds.accountsText.SetText(accountsList.String())
}

// createProgressBar creates a visual progress bar
func createProgressBar(percent float64, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	filled := int(percent / 100 * float64(width))
	empty := width - filled
	bar := "[green]" + strings.Repeat("█", filled) + "[-][#303030]" + strings.Repeat("░", empty) + "[-]"
	return bar
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         📈 QUOTA USAGE MONITOR\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	qs.table = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	qs.table.SetBorderColor(tcell.ColorDodgerBlue)

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]R[-][white] Refresh Data   [#87d7ff]Tab[-][white] Switch Focus   [#87d7ff]↑↓[-][white] Navigate      [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	tableContainer := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(qs.table, 0, 1, true)
	tableContainer.SetBorder(true).SetTitle(" Quota Details ").SetBorderColor(tcell.ColorDodgerBlue)

	qs.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(tableContainer, 0, 1, true).
		AddItem(help, 4, 0, false)

	qs.Update()
	return qs
}

func (qs *QuotaScreen) GetView() tview.Primitive {
	return qs.view
}

func (qs *QuotaScreen) Update() {
	qs.table.Clear()

	// Headers with enhanced styling
	headers := []string{"Provider", "Account", "Used", "Limit", "Usage", "Status", "Reset Time"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf(" [#00d7ff::b]%s[::-] ", header)).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDarkSlateGray)
		qs.table.SetCell(0, col, cell)
	}

	// Data rows with visual progress bars
	quotas := qs.pm.GetQuotaInfos()
	for row, quota := range quotas {
		info := GetProviderInfo(quota.Provider)

		// Status color and icon
		statusColor := "[green]"
		statusIcon := "✓"
		if quota.Status == "warning" {
			statusColor = "[yellow]"
			statusIcon = "⚠"
			statusIcon = "⚠"
		} else if quota.Status == "exceeded" {
			statusColor = "[red]"
			statusIcon = "✗"
		}

		// Reset time
		resetTime := "[gray]N/A[-]"
		if quota.ResetTime != nil {
			resetTime = quota.ResetTime.Format("Jan 02 15:04")
		}

		// Create mini progress bar for usage
		usageBar := createMiniProgressBar(quota.UsagePercent, 10)

		cells := []string{
			fmt.Sprintf(" %s %s", info.Symbol, info.Name),
			quota.AccountName,
			fmt.Sprintf("%d", quota.Used),
			fmt.Sprintf("%d", quota.Limit),
			fmt.Sprintf("%s %.0f%%", usageBar, quota.UsagePercent),
			fmt.Sprintf("%s%s %s[-]", statusColor, statusIcon, quota.Status),
			resetTime,
		}

		for col, text := range cells {
			align := tview.AlignLeft
			if col >= 2 && col <= 4 {
				align = tview.AlignRight
			}
			cell := tview.NewTableCell(text).
				SetAlign(align).
				SetSelectable(true)
			qs.table.SetCell(row+1, col, cell)
		}
	}
}

// createMiniProgressBar creates a compact progress bar
func createMiniProgressBar(percent float64, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	filled := int(percent / 100 * float64(width))
	empty := width - filled

	color := "[green]"
	if percent > 90 {
		color = "[red]"
	} else if percent > 70 {
		color = "[yellow]"
	}
	return color + strings.Repeat("▓", filled) + "[-][#404040]" + strings.Repeat("░", empty) + "[-]"
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         🤖 AI PROVIDERS\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ps.list = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSecondaryTextColor(tcell.ColorDarkCyan)
	ps.list.SetBorder(true).SetTitle(" Available Providers ").SetBorderColor(tcell.ColorDodgerBlue)

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]Enter[-][white] View Details   [#87d7ff]R[-][white] Refresh   [#87d7ff]Tab[-][white] Switch Focus      [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ps.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(ps.list, 0, 1, true).
		AddItem(help, 4, 0, false)

	ps.Update()
	return ps
}

func (ps *ProvidersScreen) GetView() tview.Primitive {
	return ps.view
}

// GetList returns the list component for external access
func (ps *ProvidersScreen) GetList() *tview.List {
	return ps.list
}

// GetSelectedProvider returns the currently selected provider
func (ps *ProvidersScreen) GetSelectedProvider() (AIProvider, ProviderInfo, int) {
	idx := ps.list.GetCurrentItem()
	providers := GetAllProviders()
	if idx >= 0 && idx < len(providers) {
		provider := providers[idx]
		return provider, GetProviderInfo(provider), ps.getAccountCount(provider)
	}
	return "", ProviderInfo{}, 0
}

// getAccountCount returns the number of accounts for a provider
func (ps *ProvidersScreen) getAccountCount(provider AIProvider) int {
	count := 0
	for _, auth := range ps.pm.GetAuthFiles() {
		if auth.Provider == provider {
			count++
		}
	}
	return count
}

// GetAccountsForProvider returns all accounts for the selected provider
func (ps *ProvidersScreen) GetAccountsForProvider(provider AIProvider) []AuthFile {
	var accounts []AuthFile
	for _, auth := range ps.pm.GetAuthFiles() {
		if auth.Provider == provider {
			accounts = append(accounts, auth)
		}
	}
	return accounts
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         ⚙️  CLI AGENTS\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	as.table = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	as.table.SetBorderColor(tcell.ColorDodgerBlue)

	tableContainer := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(as.table, 0, 1, true)
	tableContainer.SetBorder(true).SetTitle(" Agent Status ").SetBorderColor(tcell.ColorDodgerBlue)

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]Enter[-][white] View Details   [#87d7ff]R[-][white] Refresh   [#87d7ff]Tab[-][white] Switch Focus      [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	as.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(tableContainer, 0, 1, true).
		AddItem(help, 4, 0, false)

	as.Update()
	return as
}

func (as *AgentsScreen) GetView() tview.Primitive {
	return as.view
}

// GetTable returns the table component for external access
func (as *AgentsScreen) GetTable() *tview.Table {
	return as.table
}

// GetSelectedAgent returns the currently selected agent
func (as *AgentsScreen) GetSelectedAgent() *Agent {
	row, _ := as.table.GetSelection()
	agents := GetAllAgents()
	// Adjust for header row
	if row > 0 && row <= len(agents) {
		return &agents[row-1]
	}
	return nil
}

func (as *AgentsScreen) Update() {
	as.table.Clear()

	// Headers with enhanced styling
	headers := []string{"Agent Name", "Installed", "Configured"}
	for col, header := range headers {
		cell := tview.NewTableCell(fmt.Sprintf(" [#00d7ff::b]%s[::-] ", header)).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDarkSlateGray)
		as.table.SetCell(0, col, cell)
	}

	// Data rows with icons
	agents := GetAllAgents()
	for row, agent := range agents {
		installedText := "[red]  ✗ Not Installed[-]"
		if agent.Installed {
			installedText = "[green]  ✓ Installed[-]"
		}

		configuredText := "[red]  ✗ Not Configured[-]"
		if agent.Configured {
			configuredText = "[green]  ✓ Configured[-]"
		}

		// Agent icon based on name
		agentIcon := "🔧"
		switch {
		case strings.Contains(agent.Name, "Claude"):
			agentIcon = "🤖"
		case strings.Contains(agent.Name, "Codex"):
			agentIcon = "⚡"
		case strings.Contains(agent.Name, "Gemini"):
			agentIcon = "💎"
		case strings.Contains(agent.Name, "Amp"):
			agentIcon = "🔊"
		case strings.Contains(agent.Name, "OpenCode"):
			agentIcon = "📝"
		case strings.Contains(agent.Name, "Droid"):
			agentIcon = "🤖"
		}

		cells := []string{
			fmt.Sprintf(" %s %s", agentIcon, agent.Name),
			installedText,
			configuredText,
		}

		for col, text := range cells {
			cell := tview.NewTableCell(text).
				SetAlign(tview.AlignLeft).
				SetSelectable(true)
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         🔑 API KEYS\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	aks.list = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSecondaryTextColor(tcell.ColorGray)
	aks.list.SetBorder(true).SetTitle(" Your API Keys ").SetBorderColor(tcell.ColorDodgerBlue)

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]G[-][white] Generate New Key   [#87d7ff]D[-][white] Delete Selected   [#87d7ff]Tab[-][white] Switch Focus  [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	aks.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(aks.list, 0, 1, true).
		AddItem(help, 4, 0, false)

	aks.Update()
	return aks
}

// GetSelectedIndex returns the currently selected key index
func (aks *APIKeysScreen) GetSelectedIndex() int {
	return aks.list.GetCurrentItem()
}

// DeleteSelectedKey deletes the currently selected API key
func (aks *APIKeysScreen) DeleteSelectedKey() bool {
	idx := aks.list.GetCurrentItem()
	if idx >= 0 && idx < len(aks.cfg.APIKeys) {
		// Remove the key at index
		aks.cfg.APIKeys = append(aks.cfg.APIKeys[:idx], aks.cfg.APIKeys[idx+1:]...)
		return true
	}
	return false
}

func (aks *APIKeysScreen) GetView() tview.Primitive {
	return aks.view
}

func (aks *APIKeysScreen) Update() {
	aks.list.Clear()

	if len(aks.cfg.APIKeys) == 0 {
		aks.list.AddItem("  [gray]No API keys generated yet[-]", "  Press 'G' to generate your first key", 0, nil)
		return
	}

	for i, key := range aks.cfg.APIKeys {
		// Mask the key for display
		masked := key
		if len(key) > 16 {
			masked = key[:8] + "••••••••" + key[len(key)-4:]
		}

		mainText := fmt.Sprintf("  🔐 Key #%d", i+1)
		secondaryText := fmt.Sprintf("     [#5f87af]%s[-]", masked)
		aks.list.AddItem(mainText, secondaryText, 0, nil)
	}
}

// GenerateSecureKey generates a cryptographically secure API key
func GenerateSecureKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "lzm_" + hex.EncodeToString(bytes), nil
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         📋 APPLICATION LOGS\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ls.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			// Auto-scroll to bottom
			ls.textView.ScrollToEnd()
		})
	ls.textView.SetBorder(true).SetTitle(" Log Output ").SetBorderColor(tcell.ColorDodgerBlue)

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]C[-][white] Clear Logs   [#87d7ff]↑↓[-][white] Scroll   [#87d7ff]Tab[-][white] Switch Focus            [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ls.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(ls.textView, 0, 1, true).
		AddItem(help, 4, 0, false)

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
		var levelIcon, levelColor string
		switch log.Level {
		case LogLevelInfo:
			levelIcon = "ℹ️ "
			levelColor = "#87d7ff"
		case LogLevelWarn:
			levelIcon = "⚠️ "
			levelColor = "yellow"
		case LogLevelError:
			levelIcon = "❌"
			levelColor = "red"
		case LogLevelDebug:
			levelIcon = "🔍"
			levelColor = "#5f87af"
		}

		logText.WriteString(fmt.Sprintf(
			" [#404040]%s[-]  %s [%s]%-5s[-]  %s\n",
			log.Timestamp.Format("15:04:05"),
			levelIcon,
			levelColor,
			strings.ToUpper(string(log.Level)),
			log.Message,
		))
	}

	if logText.Len() == 0 {
		logText.WriteString("\n  [gray]No logs available. Logs will appear here as events occur.[-]\n")
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
		SetText("[#00d7ff::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n         ⚙️  SETTINGS\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ss.form = tview.NewForm()
	ss.form.SetBorder(true).SetTitle(" Configuration ").SetBorderColor(tcell.ColorDodgerBlue)
	ss.form.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ss.form.SetButtonBackgroundColor(tcell.ColorDodgerBlue)
	ss.form.SetLabelColor(tcell.ColorLightCyan)
	ss.buildForm()

	// Capture input to prevent global shortcuts while editing
	ss.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Allow normal form navigation
		return event
	})

	help := tview.NewTextView().
		SetText("[#5f87af]╔════════════════════════════════════════════════════════════╗\n║  [#87d7ff]↑↓[-][white] Navigate   [#87d7ff]Enter[-][white] Edit/Toggle   [#87d7ff]Tab[-][white] Switch Focus       [#5f87af]║\n╚════════════════════════════════════════════════════════════╝[-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	ss.view = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 4, 0, false).
		AddItem(ss.form, 0, 1, true).
		AddItem(help, 4, 0, false)

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
			// Update proxy manager config
			ss.pm.UpdateConfig()
			ss.pm.AddLogExternal(LogLevelInfo, "Configuration saved successfully")
		}
	})

	ss.form.AddButton("Reset", func() {
		// Reset by copying default values to the existing config pointer
		defaultCfg := NewDefaultConfig()
		ss.cfg.Port = defaultCfg.Port
		ss.cfg.RoutingStrategy = defaultCfg.RoutingStrategy
		ss.cfg.AutoStart = defaultCfg.AutoStart
		ss.cfg.DebugMode = defaultCfg.DebugMode
		ss.cfg.LogToFile = defaultCfg.LogToFile
		ss.cfg.UsageStatsEnabled = defaultCfg.UsageStatsEnabled
		ss.cfg.RequestRetryCount = defaultCfg.RequestRetryCount
		ss.cfg.QuotaExceededBehavior = defaultCfg.QuotaExceededBehavior
		// Keep existing API keys on reset
		ss.buildForm()
		ss.pm.AddLogExternal(LogLevelInfo, "Configuration reset to defaults")
	})
}
