package config

import (
	"os"
	"time"
)

type Config struct {
	Port            string
	APIKey          string
	TokenVault      TokenVaultConfig
	VisaEndpoint    AcquirerEndpoint
	MCEndpoint      AcquirerEndpoint
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type TokenVaultConfig struct {
	Provider  string // "tokenex" or "vgs"
	BaseURL   string
	APIKey    string
	TokenexID string // TokenEx-specific
}

type AcquirerEndpoint struct {
	Host    string
	Port    int
	Timeout time.Duration
	Enabled bool
}

func Load() *Config {
	return &Config{
		Port:   envOrDefault("GOPAY_PORT", "8080"),
		APIKey: envOrDefault("GOPAY_API_KEY", "dev-api-key"),
		TokenVault: TokenVaultConfig{
			Provider:  envOrDefault("TOKEN_VAULT_PROVIDER", "tokenex"),
			BaseURL:   envOrDefault("TOKEN_VAULT_URL", "https://test-api.tokenex.com"),
			APIKey:    envOrDefault("TOKEN_VAULT_API_KEY", ""),
			TokenexID: envOrDefault("TOKENEX_ID", ""),
		},
		VisaEndpoint: AcquirerEndpoint{
			Host:    envOrDefault("VISA_HOST", "localhost"),
			Port:    envOrDefaultInt("VISA_PORT", 10000),
			Timeout: 30 * time.Second,
			Enabled: envOrDefault("VISA_ENABLED", "false") == "true",
		},
		MCEndpoint: AcquirerEndpoint{
			Host:    envOrDefault("MC_HOST", "localhost"),
			Port:    envOrDefaultInt("MC_PORT", 10001),
			Timeout: 30 * time.Second,
			Enabled: envOrDefault("MC_ENABLED", "false") == "true",
		},
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
