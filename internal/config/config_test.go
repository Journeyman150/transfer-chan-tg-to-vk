package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken:         "token",
			ChannelID:        "@channel",
			RequestsPerSecond: 1,
			MaxRetries:       3,
		},
		VK: VKConfig{
			AccessToken:      "token",
			GroupID:          123,
			RequestsPerSecond: 1,
			MaxRetries:       3,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestConfig_Validate_MissingTelegramBotToken(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{
			ChannelID:        "@channel",
			RequestsPerSecond: 1,
		},
		VK: VKConfig{
			AccessToken:      "token",
			GroupID:          123,
			RequestsPerSecond: 1,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing bot token")
	}
	if err.Error() != "configuration validation failed: telegram.bot_token is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig_Validate_MissingVKAccessToken(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken:         "token",
			ChannelID:        "@channel",
			RequestsPerSecond: 1,
		},
		VK: VKConfig{
			GroupID:          123,
			RequestsPerSecond: 1,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing access token")
	}
	if err.Error() != "configuration validation failed: vk.access_token is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig_Validate_InvalidDateRange(t *testing.T) {
	start := time.Now()
	end := start.Add(-24 * time.Hour) // end before start
	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken:         "token",
			ChannelID:        "@channel",
			RequestsPerSecond: 1,
			StartDate:        &start,
			EndDate:          &end,
		},
		VK: VKConfig{
			AccessToken:      "token",
			GroupID:          123,
			RequestsPerSecond: 1,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid date range")
	}
	if err.Error() != "configuration validation failed: telegram.start_date must be before end_date" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig_Validate_NegativeRetries(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken:         "token",
			ChannelID:        "@channel",
			RequestsPerSecond: 1,
			MaxRetries:       -1,
		},
		VK: VKConfig{
			AccessToken:      "token",
			GroupID:          123,
			RequestsPerSecond: 1,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for negative retries")
	}
	if err.Error() != "configuration validation failed: telegram.max_retries cannot be negative" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig_Validate_ZeroRequestsPerSecond(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken:         "token",
			ChannelID:        "@channel",
			RequestsPerSecond: 0,
		},
		VK: VKConfig{
			AccessToken:      "token",
			GroupID:          123,
			RequestsPerSecond: 1,
		},
		Transfer: TransferConfig{
			MaxPostsPerBatch:       10,
			MaxAttachmentsPerPost: 10,
		},
		Media: MediaConfig{
			TempDir:                 "/tmp",
			MaxConcurrentDownloads: 5,
			MaxConcurrentUploads:   5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero requests per second")
	}
	if err.Error() != "configuration validation failed: telegram.requests_per_second must be positive" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoader_Load_FromFile(t *testing.T) {
	// Create a temporary config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configContent := `
telegram:
  bot_token: "test_token"
  channel_id: "@test"
  timeout: "30s"
  requests_per_second: 1
  max_retries: 3

vk:
  access_token: "vk_token"
  group_id: 123
  timeout: "60s"
  post_delay: "3s"
  requests_per_second: 1
  max_retries: 3

transfer:
  max_posts_per_batch: 10
  max_attachments_per_post: 10

media:
  temp_dir: "/tmp"
  max_concurrent_downloads: 5
  max_concurrent_uploads: 5
  download_timeout: "30s"
  upload_timeout: "300s"

error_handling:
  retry:
    base_delay: "1s"
    max_delay: "30s"
  circuit_breaker:
    reset_timeout: "60s"
  checkpoint:
    save_interval: "60s"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg == nil {
		t.Fatal("config is nil")
	}
	if cfg.Telegram.BotToken != "test_token" {
		t.Errorf("expected bot token 'test_token', got %q", cfg.Telegram.BotToken)
	}
	if cfg.VK.GroupID != 123 {
		t.Errorf("expected group id 123, got %d", cfg.VK.GroupID)
	}
}

func TestLoader_Load_Defaults(t *testing.T) {
	loader := NewLoader()
	// No config file, should use defaults
	cfg, err := loader.Load("")
	if err == nil {
		t.Log("config loaded with defaults (validation will fail due to missing required fields)")
		// Validation will fail because required fields are missing
		err = cfg.Validate()
		if err == nil {
			t.Error("expected validation error for missing required fields")
		}
	} else {
		// Expected because required fields are missing
		t.Logf("load failed as expected: %v", err)
	}
}

func TestDuration_UnmarshalText(t *testing.T) {
	var d Duration
	err := d.UnmarshalText([]byte("1h30m"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Duration(d) != 90*time.Minute {
		t.Errorf("expected 90 minutes, got %v", time.Duration(d))
	}
}

func TestDuration_MarshalText(t *testing.T) {
	d := Duration(90 * time.Minute)
	text, err := d.MarshalText()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(text) != "1h30m0s" {
		t.Errorf("expected '1h30m0s', got %q", string(text))
	}
}