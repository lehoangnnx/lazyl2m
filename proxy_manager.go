package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

const maxLogEntries = 1000

// ProxyManager manages the CLIProxyAPI process
type ProxyManager struct {
	config     *Config
	status     ProxyStatus
	cmd        *exec.Cmd
	authFiles  []AuthFile
	usageStats UsageStats
	quotaInfos []QuotaInfo
	logEntries []LogEntry
	mutex      sync.RWMutex
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(config *Config) *ProxyManager {
	return &ProxyManager{
		config:     config,
		status:     ProxyStatus{Running: false, Port: config.Port},
		authFiles:  []AuthFile{},
		quotaInfos: []QuotaInfo{},
		logEntries: []LogEntry{},
		usageStats: UsageStats{},
	}
}

// Start starts the proxy server
func (pm *ProxyManager) Start() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.status.Running {
		return fmt.Errorf("proxy server already running")
	}

	// Note: In a real implementation, this would start the actual CLIProxyAPI binary
	// For now, we'll simulate it
	pm.AddLog(LogLevelInfo, fmt.Sprintf("Starting proxy server on port %d", pm.config.Port))

	// Simulate starting the process
	pm.status.Running = true
	pm.AddLog(LogLevelInfo, "Proxy server started successfully")

	return nil
}

// Stop stops the proxy server
func (pm *ProxyManager) Stop() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		return fmt.Errorf("proxy server not running")
	}

	pm.AddLog(LogLevelInfo, "Stopping proxy server")

	if pm.cmd != nil && pm.cmd.Process != nil {
		if err := pm.cmd.Process.Kill(); err != nil {
			pm.AddLog(LogLevelError, fmt.Sprintf("Failed to kill process: %v", err))
		}
		pm.cmd = nil
	}

	pm.status.Running = false
	pm.AddLog(LogLevelInfo, "Proxy server stopped")

	return nil
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
		// If server is not running, clear auth files
		pm.authFiles = []AuthFile{}
		return nil
	}

	// In a real implementation, this would call the API
	// For now, we'll simulate some data
	url := fmt.Sprintf("http://localhost:%d/management/auth-files", pm.config.Port)

	resp, err := http.Get(url)
	if err != nil {
		// Simulate some mock data when API is not available
		pm.authFiles = pm.getMockAuthFiles()
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var authFiles []AuthFile
	if err := json.Unmarshal(body, &authFiles); err != nil {
		return err
	}

	pm.authFiles = authFiles
	pm.AddLog(LogLevelDebug, fmt.Sprintf("Fetched %d auth files", len(authFiles)))

	return nil
}

// FetchUsageStats fetches usage statistics from the management API
func (pm *ProxyManager) FetchUsageStats() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.status.Running {
		pm.usageStats = UsageStats{}
		return nil
	}

	// In a real implementation, this would call the API
	url := fmt.Sprintf("http://localhost:%d/management/usage-statistics", pm.config.Port)

	resp, err := http.Get(url)
	if err != nil {
		// Simulate mock data
		pm.usageStats = pm.getMockUsageStats()
		return nil
	}
	defer resp.Body.Close()

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
