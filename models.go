package main

import (
	"os/exec"
	"time"
)

// AIProvider represents supported AI providers
type AIProvider string

const (
	ProviderGemini        AIProvider = "gemini"
	ProviderClaude        AIProvider = "claude"
	ProviderCodex         AIProvider = "codex"
	ProviderQwen          AIProvider = "qwen"
	ProviderIFlow         AIProvider = "iflow"
	ProviderAntigravity   AIProvider = "antigravity"
	ProviderVertex        AIProvider = "vertex"
	ProviderKiro          AIProvider = "kiro"
	ProviderGitHubCopilot AIProvider = "github_copilot"
	ProviderCursor        AIProvider = "cursor"
)

// ProviderInfo holds display information for a provider
type ProviderInfo struct {
	Name   string
	Symbol string
	Color  string
}

// GetProviderInfo returns display info for a provider
func GetProviderInfo(provider AIProvider) ProviderInfo {
	switch provider {
	case ProviderGemini:
		return ProviderInfo{Name: "Gemini", Symbol: "💎", Color: "blue"}
	case ProviderClaude:
		return ProviderInfo{Name: "Claude", Symbol: "🤖", Color: "orange"}
	case ProviderCodex:
		return ProviderInfo{Name: "Codex", Symbol: "⚡", Color: "green"}
	case ProviderQwen:
		return ProviderInfo{Name: "Qwen", Symbol: "🐉", Color: "red"}
	case ProviderIFlow:
		return ProviderInfo{Name: "iFlow", Symbol: "🌊", Color: "cyan"}
	case ProviderAntigravity:
		return ProviderInfo{Name: "Antigravity", Symbol: "🚀", Color: "purple"}
	case ProviderVertex:
		return ProviderInfo{Name: "Vertex AI", Symbol: "🔷", Color: "blue"}
	case ProviderKiro:
		return ProviderInfo{Name: "Kiro", Symbol: "🎯", Color: "yellow"}
	case ProviderGitHubCopilot:
		return ProviderInfo{Name: "GitHub Copilot", Symbol: "🐙", Color: "gray"}
	case ProviderCursor:
		return ProviderInfo{Name: "Cursor", Symbol: "➡️", Color: "white"}
	default:
		return ProviderInfo{Name: string(provider), Symbol: "❓", Color: "white"}
	}
}

// GetAllProviders returns all supported providers
func GetAllProviders() []AIProvider {
	return []AIProvider{
		ProviderGemini,
		ProviderClaude,
		ProviderCodex,
		ProviderQwen,
		ProviderIFlow,
		ProviderAntigravity,
		ProviderVertex,
		ProviderKiro,
		ProviderGitHubCopilot,
		ProviderCursor,
	}
}

// ProxyStatus represents the proxy server status
type ProxyStatus struct {
	Running bool
	Port    int
}

// AuthFile represents an authenticated account
type AuthFile struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Provider AIProvider `json:"provider"`
	Status   string     `json:"status"` // "active", "expired", "error"
	Email    string     `json:"email"`
	Token    string     `json:"token"`
	ExpireAt *time.Time `json:"expire_at"`
}

// UsageStats represents overall usage statistics
type UsageStats struct {
	TotalRequests   int     `json:"total_requests"`
	SuccessRequests int     `json:"success_requests"`
	FailedRequests  int     `json:"failed_requests"`
	TotalTokens     int     `json:"total_tokens"`
	SuccessRate     float64 `json:"success_rate"`
	LastUpdated     time.Time
}

// QuotaInfo represents quota information for an account
type QuotaInfo struct {
	AccountID    string     `json:"account_id"`
	AccountName  string     `json:"account_name"`
	Provider     AIProvider `json:"provider"`
	Used         int        `json:"used"`
	Limit        int        `json:"limit"`
	UsagePercent float64    `json:"usage_percent"`
	Status       string     `json:"status"` // "ok", "warning", "exceeded"
	ResetTime    *time.Time `json:"reset_time"`
}

// LogLevel represents log entry severity
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelDebug LogLevel = "debug"
)

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

// Agent represents a CLI agent
type Agent struct {
	Name       string
	Command    string // Command to check if installed
	Installed  bool
	Configured bool
}

// checkCommandExists checks if a command exists in PATH
func checkCommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetAllAgents returns all supported CLI agents with real detection
func GetAllAgents() []Agent {
	agents := []Agent{
		{Name: "Claude Code", Command: "claude"},
		{Name: "Codex CLI", Command: "codex"},
		{Name: "Gemini CLI", Command: "gemini"},
		{Name: "Amp CLI", Command: "amp"},
		{Name: "OpenCode", Command: "opencode"},
		{Name: "Aider", Command: "aider"},
	}

	// Check installation status for each agent
	for i := range agents {
		agents[i].Installed = checkCommandExists(agents[i].Command)
		// Configuration check would require reading config files
		// For now, mark as configured if installed
		agents[i].Configured = agents[i].Installed
	}

	return agents
}

// RoutingStrategy represents load balancing strategy
type RoutingStrategy string

const (
	RoutingRoundRobin RoutingStrategy = "round-robin"
	RoutingFillFirst  RoutingStrategy = "fill-first"
)

// Config represents application configuration
type Config struct {
	Port                  int             `json:"port"`
	RoutingStrategy       RoutingStrategy `json:"routing_strategy"`
	AutoStart             bool            `json:"auto_start"`
	DebugMode             bool            `json:"debug_mode"`
	LogToFile             bool            `json:"log_to_file"`
	UsageStatsEnabled     bool            `json:"usage_stats_enabled"`
	RequestRetryCount     int             `json:"request_retry_count"`
	APIKeys               []string        `json:"api_keys"`
	QuotaExceededBehavior string          `json:"quota_exceeded_behavior"` // "skip", "stop", "continue"
}

// NewDefaultConfig returns a config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Port:                  8317,
		RoutingStrategy:       RoutingRoundRobin,
		AutoStart:             false,
		DebugMode:             false,
		LogToFile:             false,
		UsageStatsEnabled:     true,
		RequestRetryCount:     3,
		APIKeys:               []string{},
		QuotaExceededBehavior: "skip",
	}
}
