package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	maxLogEntries      = 1000
	githubRepo         = "router-for-me/CLIProxyAPIPlus"
	githubAPIURL       = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	defaultBinaryName  = "CLIProxyAPI"
	managementBasePath = "/v0/management"
)

// ProxyManager manages the CLIProxyAPI process
type ProxyManager struct {
	config        *Config
	status        ProxyStatus
	process       *exec.Cmd
	authFiles     []AuthFile
	usageStats    UsageStats
	quotaInfos    []QuotaInfo
	logEntries    []LogEntry
	mutex         sync.RWMutex

	// Paths
	binaryPath    string
	configPath    string
	authDir       string
	managementKey string

	// State
	isDownloading    bool
	downloadProgress float64
	lastError        string
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(config *Config) *ProxyManager {
	homeDir, _ := os.UserHomeDir()
	appDir := filepath.Join(homeDir, ".local", "share", "lazyl2m")
	authDir := filepath.Join(homeDir, ".cli-proxy-api")

	// Create directories if they don't exist
	os.MkdirAll(appDir, 0755)
	os.MkdirAll(authDir, 0755)

	pm := &ProxyManager{
		config:        config,
		status:        ProxyStatus{Running: false, Port: config.Port},
		authFiles:     []AuthFile{},
		quotaInfos:    []QuotaInfo{},
		logEntries:    []LogEntry{},
		usageStats:    UsageStats{},
		binaryPath:    filepath.Join(appDir, defaultBinaryName),
		configPath:    filepath.Join(appDir, "config.yaml"),
		authDir:       authDir,
		managementKey: generateManagementKey(),
	}

	// Ensure config file exists
	pm.ensureConfigExists()

	return pm
}

// generateManagementKey generates a random management key
func generateManagementKey() string {
	return fmt.Sprintf("lzm-%d", time.Now().UnixNano())
}

// IsBinaryInstalled checks if the CLIProxyAPI binary is installed
func (pm *ProxyManager) IsBinaryInstalled() bool {
	_, err := os.Stat(pm.binaryPath)
	return err == nil
}

// GetBinaryPath returns the path to the binary
func (pm *ProxyManager) GetBinaryPath() string {
	return pm.binaryPath
}

// IsDownloading returns whether the binary is being downloaded
func (pm *ProxyManager) IsDownloading() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.isDownloading
}

// GetDownloadProgress returns the download progress (0.0 to 1.0)
func (pm *ProxyManager) GetDownloadProgress() float64 {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.downloadProgress
}

// GetLastError returns the last error message
func (pm *ProxyManager) GetLastError() string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.lastError
}

// GetEndpoint returns the API endpoint URL
func (pm *ProxyManager) GetEndpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d/v1", pm.config.Port)
}

// GetManagementURL returns the management API URL
func (pm *ProxyManager) GetManagementURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", pm.config.Port, managementBasePath)
}

// ensureConfigExists creates default config if it doesn't exist
func (pm *ProxyManager) ensureConfigExists() {
	if _, err := os.Stat(pm.configPath); err == nil {
		return
	}

	defaultConfig := fmt.Sprintf(`host: "127.0.0.1"
port: %d
auth-dir: "%s"

api-keys:
  - "lazyl2m-local-%d"

remote-management:
  allow-remote: false
  secret-key: "%s"

debug: %t
logging-to-file: %t
usage-statistics-enabled: %t

routing:
  strategy: "%s"

quota-exceeded:
  switch-project: true
  switch-preview-model: true

request-retry: %d
max-retry-interval: 30
`,
		pm.config.Port,
		pm.authDir,
		time.Now().UnixNano(),
		pm.managementKey,
		pm.config.DebugMode,
		pm.config.LogToFile,
		pm.config.UsageStatsEnabled,
		pm.config.RoutingStrategy,
		pm.config.RequestRetryCount,
	)

	os.WriteFile(pm.configPath, []byte(defaultConfig), 0644)
}

// UpdateConfig updates the config file with current settings
func (pm *ProxyManager) UpdateConfig() error {
	os.Remove(pm.configPath)
	pm.ensureConfigExists()
	return nil
}

// Start starts the proxy server
func (pm *ProxyManager) Start() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.status.Running {
		return fmt.Errorf("proxy server already running")
	}

	if !pm.IsBinaryInstalled() {
		return fmt.Errorf("CLIProxyAPI binary not installed. Please install it first")
	}

	pm.AddLog(LogLevelInfo, fmt.Sprintf("Starting proxy server on port %d", pm.config.Port))
	pm.lastError = ""

	// Create the process
	pm.process = exec.Command(pm.binaryPath, "-config", pm.configPath)
	pm.process.Dir = filepath.Dir(pm.binaryPath)

	// Set up pipes for output
	stdout, _ := pm.process.StdoutPipe()
	stderr, _ := pm.process.StderrPipe()

	// Set environment
	pm.process.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start the process
	if err := pm.process.Start(); err != nil {
		pm.lastError = err.Error()
		pm.AddLog(LogLevelError, fmt.Sprintf("Failed to start proxy: %v", err))
		return err
	}

	// Handle process output in background
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				pm.AddLogExternal(LogLevelDebug, strings.TrimSpace(string(buf[:n])))
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				pm.AddLogExternal(LogLevelWarn, strings.TrimSpace(string(buf[:n])))
			}
			if err != nil {
				break
			}
		}
	}()

	// Monitor process exit
	go func() {
		if pm.process != nil {
			err := pm.process.Wait()
			pm.mutex.Lock()
			pm.status.Running = false
			pm.process = nil
			if err != nil {
				pm.lastError = err.Error()
				pm.AddLog(LogLevelError, fmt.Sprintf("Proxy exited with error: %v", err))
			} else {
				pm.AddLog(LogLevelInfo, "Proxy process exited")
			}
			pm.mutex.Unlock()
		}
	}()

	// Wait for the server to start (check if process is running after a brief delay)
	time.Sleep(1500 * time.Millisecond)

	if pm.process != nil && pm.process.Process != nil {
		// Check if process is still alive
		if err := pm.process.Process.Signal(syscall.Signal(0)); err == nil {
			pm.status.Running = true
			pm.AddLog(LogLevelInfo, "Proxy server started successfully")
			return nil
		}
	}

	pm.lastError = "Process failed to start"
	return fmt.Errorf("failed to start proxy server")
}

// Stop stops the proxy server
func (pm *ProxyManager) Stop() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		return fmt.Errorf("proxy server not running")
	}

	pm.AddLog(LogLevelInfo, "Stopping proxy server")

	if pm.process != nil && pm.process.Process != nil {
		// Try graceful termination first
		pm.process.Process.Signal(syscall.SIGTERM)

		// Wait a bit for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- pm.process.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if still running
			pm.process.Process.Kill()
			pm.AddLog(LogLevelWarn, "Force killed proxy process")
		}
		pm.process = nil
	}

	// Also kill any processes on the port (cleanup orphans)
	pm.killProcessOnPort(pm.config.Port)

	pm.status.Running = false
	pm.AddLog(LogLevelInfo, "Proxy server stopped")

	return nil
}

// killProcessOnPort kills any process listening on the specified port
func (pm *ProxyManager) killProcessOnPort(port int) {
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	for _, pidStr := range pids {
		var pid int
		if _, err := fmt.Sscanf(pidStr, "%d", &pid); err == nil {
			syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

// GetStatus returns the current proxy status
func (pm *ProxyManager) GetStatus() ProxyStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.status
}

// FetchAuthFiles fetches authenticated accounts from the management API
func (pm *ProxyManager) FetchAuthFiles() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		// If server is not running, try to read from auth directory directly
		pm.authFiles = pm.scanAuthDirectory()
		return nil
	}

	url := fmt.Sprintf("%s/auth-files", pm.GetManagementURL())

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Management-Key", pm.managementKey)

	resp, err := client.Do(req)
	if err != nil {
		// Fallback to scanning auth directory
		pm.authFiles = pm.scanAuthDirectory()
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pm.authFiles = pm.scanAuthDirectory()
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var authFiles []AuthFile
	if err := json.Unmarshal(body, &authFiles); err != nil {
		return err
	}

	pm.authFiles = authFiles
	pm.AddLog(LogLevelDebug, fmt.Sprintf("Fetched %d auth files from API", len(authFiles)))

	return nil
}

// scanAuthDirectory reads auth files directly from the auth directory
func (pm *ProxyManager) scanAuthDirectory() []AuthFile {
	var authFiles []AuthFile

	entries, err := os.ReadDir(pm.authDir)
	if err != nil {
		return authFiles
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(pm.authDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Parse provider and email from filename
		provider, email := pm.parseAuthFileName(entry.Name())

		// Try to parse the JSON to get more info
		var authData map[string]interface{}
		json.Unmarshal(data, &authData)

		auth := AuthFile{
			ID:       entry.Name(),
			Name:     email,
			Provider: provider,
			Status:   "active",
			Email:    email,
		}

		authFiles = append(authFiles, auth)
	}

	return authFiles
}

// parseAuthFileName extracts provider and email from auth file name
func (pm *ProxyManager) parseAuthFileName(filename string) (AIProvider, string) {
	// Filename format: provider-email.json (e.g., gemini-cli-user@email.com.json)
	name := strings.TrimSuffix(filename, ".json")

	providers := map[string]AIProvider{
		"gemini-cli":     ProviderGemini,
		"gemini":         ProviderGemini,
		"claude":         ProviderClaude,
		"codex":          ProviderCodex,
		"qwen":           ProviderQwen,
		"iflow":          ProviderIFlow,
		"antigravity":    ProviderAntigravity,
		"vertex":         ProviderVertex,
		"kiro":           ProviderKiro,
		"github-copilot": ProviderGitHubCopilot,
		"copilot":        ProviderGitHubCopilot,
		"cursor":         ProviderCursor,
	}

	for prefix, prov := range providers {
		if strings.HasPrefix(name, prefix+"-") {
			email := strings.TrimPrefix(name, prefix+"-")
			return prov, email
		}
	}

	return ProviderGemini, name
}

// FetchUsageStats fetches usage statistics from the management API
func (pm *ProxyManager) FetchUsageStats() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		pm.usageStats = UsageStats{LastUpdated: time.Now()}
		return nil
	}

	url := fmt.Sprintf("%s/usage-statistics", pm.GetManagementURL())

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Management-Key", pm.managementKey)

	resp, err := client.Do(req)
	if err != nil {
		// Keep last stats with updated timestamp
		pm.usageStats.LastUpdated = time.Now()
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pm.usageStats.LastUpdated = time.Now()
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var stats UsageStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return err
	}

	stats.LastUpdated = time.Now()
	pm.usageStats = stats
	pm.AddLog(LogLevelDebug, "Updated usage statistics")

	return nil
}

// GetAuthFiles returns the current auth files
func (pm *ProxyManager) GetAuthFiles() []AuthFile {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.authFiles
}

// GetUsageStats returns the current usage statistics
func (pm *ProxyManager) GetUsageStats() UsageStats {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.usageStats
}

// GetQuotaInfos returns quota information for all accounts
func (pm *ProxyManager) GetQuotaInfos() []QuotaInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Generate quota info from auth files
	quotas := []QuotaInfo{}
	for _, auth := range pm.authFiles {
		// Simulate quota data
		used := 1000 + len(auth.ID)*100
		limit := 10000
		percent := float64(used) / float64(limit) * 100

		status := "ok"
		if percent > 90 {
			status = "exceeded"
		} else if percent > 70 {
			status = "warning"
		}

		resetTime := time.Now().Add(24 * time.Hour)
		quotas = append(quotas, QuotaInfo{
			AccountID:    auth.ID,
			AccountName:  auth.Name,
			Provider:     auth.Provider,
			Used:         used,
			Limit:        limit,
			UsagePercent: percent,
			Status:       status,
			ResetTime:    &resetTime,
		})
	}

	return quotas
}

// GetLogs returns all log entries
func (pm *ProxyManager) GetLogs() []LogEntry {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.logEntries
}

// AddLog adds a log entry (internal, holds lock)
func (pm *ProxyManager) AddLog(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	pm.logEntries = append(pm.logEntries, entry)

	// Keep only last maxLogEntries
	if len(pm.logEntries) > maxLogEntries {
		pm.logEntries = pm.logEntries[len(pm.logEntries)-maxLogEntries:]
	}
}

// AddLogExternal adds a log entry (external, acquires lock)
func (pm *ProxyManager) AddLogExternal(level LogLevel, message string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.AddLog(level, message)
}

// ClearLogs clears all log entries
func (pm *ProxyManager) ClearLogs() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.logEntries = []LogEntry{}
}

// Mock data generators for when API is not available

func (pm *ProxyManager) getMockAuthFiles() []AuthFile {
	expireTime := time.Now().Add(30 * 24 * time.Hour)
	return []AuthFile{
		{
			ID:       "auth-1",
			Name:     "Gemini Account 1",
			Provider: ProviderGemini,
			Status:   "active",
			Email:    "user@example.com",
			ExpireAt: &expireTime,
		},
		{
			ID:       "auth-2",
			Name:     "Claude Account 1",
			Provider: ProviderClaude,
			Status:   "active",
			Email:    "user@example.com",
			ExpireAt: &expireTime,
		},
		{
			ID:       "auth-3",
			Name:     "Codex Account 1",
			Provider: ProviderCodex,
			Status:   "active",
			Email:    "dev@example.com",
			ExpireAt: &expireTime,
		},
	}
}

func (pm *ProxyManager) getMockUsageStats() UsageStats {
	return UsageStats{
		TotalRequests:   1234,
		SuccessRequests: 1180,
		FailedRequests:  54,
		TotalTokens:     567890,
		SuccessRate:     95.6,
		LastUpdated:     time.Now(),
	}
}

// DownloadAndInstallBinary downloads and installs the CLIProxyAPI binary
func (pm *ProxyManager) DownloadAndInstallBinary() error {
	pm.mutex.Lock()
	pm.isDownloading = true
	pm.downloadProgress = 0
	pm.lastError = ""
	pm.mutex.Unlock()

	defer func() {
		pm.mutex.Lock()
		pm.isDownloading = false
		pm.mutex.Unlock()
	}()

	pm.AddLogExternal(LogLevelInfo, "Starting CLIProxyAPI download...")

	// Fetch latest release info
	releaseInfo, err := pm.fetchLatestRelease()
	if err != nil {
		pm.mutex.Lock()
		pm.lastError = err.Error()
		pm.mutex.Unlock()
		pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to fetch release info: %v", err))
		return err
	}

	pm.mutex.Lock()
	pm.downloadProgress = 0.1
	pm.mutex.Unlock()

	// Find compatible asset
	asset := pm.findCompatibleAsset(releaseInfo)
	if asset == nil {
		err := fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
		pm.mutex.Lock()
		pm.lastError = err.Error()
		pm.mutex.Unlock()
		pm.AddLogExternal(LogLevelError, err.Error())
		return err
	}

	pm.AddLogExternal(LogLevelInfo, fmt.Sprintf("Found asset: %s", asset.Name))

	// Download the asset
	data, err := pm.downloadAsset(asset.DownloadURL)
	if err != nil {
		pm.mutex.Lock()
		pm.lastError = err.Error()
		pm.mutex.Unlock()
		pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to download: %v", err))
		return err
	}

	pm.mutex.Lock()
	pm.downloadProgress = 0.7
	pm.mutex.Unlock()

	// Extract and install
	if err := pm.extractAndInstall(data, asset.Name); err != nil {
		pm.mutex.Lock()
		pm.lastError = err.Error()
		pm.mutex.Unlock()
		pm.AddLogExternal(LogLevelError, fmt.Sprintf("Failed to install: %v", err))
		return err
	}

	pm.mutex.Lock()
	pm.downloadProgress = 1.0
	pm.mutex.Unlock()

	pm.AddLogExternal(LogLevelInfo, "CLIProxyAPI installed successfully!")
	return nil
}

// GitHub release structures
type releaseInfo struct {
	TagName string       `json:"tag_name"`
	Assets  []assetInfo  `json:"assets"`
}

type assetInfo struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// fetchLatestRelease fetches the latest release info from GitHub
func (pm *ProxyManager) fetchLatestRelease() (*releaseInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", githubAPIURL, nil)
	req.Header.Set("User-Agent", "LazyL2M/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release releaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// findCompatibleAsset finds a compatible binary asset for the current platform
func (pm *ProxyManager) findCompatibleAsset(release *releaseInfo) *assetInfo {
	// Determine target platform
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		arch = runtime.GOARCH
	}

	platform := runtime.GOOS
	targetPattern := fmt.Sprintf("%s_%s", platform, arch)

	skipPatterns := []string{"windows", "linux", "checksum", ".sha256", ".md5"}
	if platform != "linux" {
		skipPatterns = append(skipPatterns, "linux")
	}
	if platform != "darwin" {
		skipPatterns = append(skipPatterns, "darwin")
	}

	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)

		// Skip non-matching platforms
		shouldSkip := false
		for _, skip := range skipPatterns {
			if strings.Contains(name, skip) && !strings.Contains(targetPattern, skip) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Check for target pattern
		if strings.Contains(name, targetPattern) {
			return &asset
		}
	}

	return nil
}

// downloadAsset downloads an asset from the given URL
func (pm *ProxyManager) downloadAsset(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "LazyL2M/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	pm.mutex.Lock()
	pm.downloadProgress = 0.6
	pm.mutex.Unlock()

	return data, nil
}

// extractAndInstall extracts and installs the binary
func (pm *ProxyManager) extractAndInstall(data []byte, assetName string) error {
	tempDir, err := os.MkdirTemp("", "lazyl2m-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	downloadedFile := filepath.Join(tempDir, assetName)
	if err := os.WriteFile(downloadedFile, data, 0644); err != nil {
		return err
	}

	var binaryPath string

	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		// Extract tar.gz
		binaryPath, err = pm.extractTarGz(downloadedFile, tempDir)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(assetName, ".zip") {
		// Extract zip
		binaryPath, err = pm.extractZip(downloadedFile, tempDir)
		if err != nil {
			return err
		}
	} else {
		// Assume it's the binary itself
		binaryPath = downloadedFile
	}

	if binaryPath == "" {
		return fmt.Errorf("could not find binary in archive")
	}

	// Copy to final destination
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}

	// Remove old binary if exists
	os.Remove(pm.binaryPath)

	// Write new binary
	if err := os.WriteFile(pm.binaryPath, binaryData, 0755); err != nil {
		return err
	}

	// Make executable
	os.Chmod(pm.binaryPath, 0755)

	pm.AddLogExternal(LogLevelInfo, fmt.Sprintf("Binary installed to %s", pm.binaryPath))
	return nil
}

// extractTarGz extracts a tar.gz archive and returns the path to the binary
func (pm *ProxyManager) extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var binaryPath string
	binaryNames := []string{"CLIProxyAPI", "cli-proxy-api", "cli-proxy-api-plus", "proxy"}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return "", err
			}
			io.Copy(outFile, tr)
			outFile.Close()

			// Check if this is the binary
			baseName := filepath.Base(header.Name)
			for _, name := range binaryNames {
				if strings.EqualFold(baseName, name) {
					binaryPath = target
					break
				}
			}

			// If no match by name, check if it's executable
			if binaryPath == "" && header.Mode&0111 != 0 && !strings.HasSuffix(baseName, ".sh") {
				binaryPath = target
			}
		}
	}

	return binaryPath, nil
}

// extractZip extracts a zip archive and returns the path to the binary
func (pm *ProxyManager) extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var binaryPath string
	binaryNames := []string{"CLIProxyAPI", "cli-proxy-api", "cli-proxy-api-plus", "proxy"}

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		outFile, err := os.Create(target)
		if err != nil {
			rc.Close()
			return "", err
		}

		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		// Check if this is the binary
		baseName := filepath.Base(f.Name)
		for _, name := range binaryNames {
			if strings.EqualFold(baseName, name) {
				binaryPath = target
				break
			}
		}

		// If no match by name, check if it might be executable
		if binaryPath == "" && f.Mode()&0111 != 0 && !strings.HasSuffix(baseName, ".sh") {
			binaryPath = target
		}
	}

	return binaryPath, nil
}

// FetchQuotaInfo fetches quota info from the management API
func (pm *ProxyManager) FetchQuotaInfo() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		return nil
	}

	url := fmt.Sprintf("%s/quotas", pm.GetManagementURL())

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Management-Key", pm.managementKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var quotas []QuotaInfo
	if err := json.Unmarshal(body, &quotas); err != nil {
		return err
	}

	pm.quotaInfos = quotas
	return nil
}
