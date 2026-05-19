package config

import (
	"fmt"
	"strings"
	"time"
)

// Duration is a wrapper around time.Duration for custom unmarshaling
type Duration time.Duration

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// Config represents the complete application configuration
type Config struct {
	Telegram   TelegramConfig   `mapstructure:"telegram"`
	VK         VKConfig         `mapstructure:"vk"`
	Transfer   TransferConfig   `mapstructure:"transfer"`
	Media      MediaConfig      `mapstructure:"media"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	ErrorHandling ErrorHandlingConfig `mapstructure:"error_handling"`
}

// TelegramConfig holds Telegram API configuration
type TelegramConfig struct {
	BotToken         string        `mapstructure:"bot_token"`
	ChannelID        string        `mapstructure:"channel_id"`
	ChatID           int64         `mapstructure:"chat_id"`
	APIURL           string        `mapstructure:"api_url"`
	Timeout          Duration      `mapstructure:"timeout"`
	Debug            bool          `mapstructure:"debug"`
	RequestsPerSecond int          `mapstructure:"requests_per_second"`
	MaxRetries       int           `mapstructure:"max_retries"`
	MaxPosts         int           `mapstructure:"max_posts"`
	BatchSize        int           `mapstructure:"batch_size"`
	StartDate        *time.Time    `mapstructure:"start_date"`
	EndDate          *time.Time    `mapstructure:"end_date"`
	IncludeMedia     bool          `mapstructure:"include_media"`
	MaxFileSize      int64         `mapstructure:"max_file_size"`
}

// VKConfig holds VK API configuration
type VKConfig struct {
	AccessToken      string        `mapstructure:"access_token"`
	GroupID          int           `mapstructure:"group_id"`
	APIURL           string        `mapstructure:"api_url"`
	APIVersion       string        `mapstructure:"api_version"`
	Timeout          Duration      `mapstructure:"timeout"`
	Language         string        `mapstructure:"language"`
	RequestsPerSecond float64      `mapstructure:"requests_per_second"`
	MaxRetries       int           `mapstructure:"max_retries"`
	FromGroup        bool          `mapstructure:"from_group"`
	Signed           bool          `mapstructure:"signed"`
	FriendsOnly      bool          `mapstructure:"friends_only"`
	PostDelay        Duration      `mapstructure:"post_delay"`
	AlbumID          int           `mapstructure:"album_id"`
}

// TransferConfig holds transfer-specific configuration
type TransferConfig struct {
	DryRun           bool          `mapstructure:"dry_run"`
	SkipDuplicates   bool          `mapstructure:"skip_duplicates"`
	PreserveDates    bool          `mapstructure:"preserve_dates"`
	IncludeText      bool          `mapstructure:"include_text"`
	IncludeMedia     bool          `mapstructure:"include_media"`
	MediaTypes       []string      `mapstructure:"media_types"`
	PreserveFormatting bool        `mapstructure:"preserve_formatting"`
	AddSourceLink    bool          `mapstructure:"add_source_link"`
	SourceLinkFormat string        `mapstructure:"source_link_format"`
	IncludeForwardInfo bool        `mapstructure:"include_forward_info"`
	ForwardFormat    string        `mapstructure:"forward_format"`
	MaxPostsPerBatch int           `mapstructure:"max_posts_per_batch"`
	MaxAttachmentsPerPost int      `mapstructure:"max_attachments_per_post"`
	SkipFailedPosts  bool          `mapstructure:"skip_failed_posts"`
	MaxErrorsBeforeStop int        `mapstructure:"max_errors_before_stop"`
	StopOnAuthError  bool          `mapstructure:"stop_on_auth_error"`
}

// MediaConfig holds media processing configuration
type MediaConfig struct {
	TempDir                  string        `mapstructure:"temp_dir"`
	KeepFiles                bool          `mapstructure:"keep_files"`
	MaxConcurrentDownloads   int           `mapstructure:"max_concurrent_downloads"`
	MaxConcurrentUploads     int           `mapstructure:"max_concurrent_uploads"`
	DownloadTimeout          Duration      `mapstructure:"download_timeout"`
	UploadTimeout            Duration      `mapstructure:"upload_timeout"`
	DownloadRetries          int           `mapstructure:"download_retries"`
	
	Photos     PhotoConfig   `mapstructure:"photos"`
	Videos     VideoConfig   `mapstructure:"videos"`
	Documents  DocumentConfig `mapstructure:"documents"`
}

// PhotoConfig holds photo processing configuration
type PhotoConfig struct {
	MaxWidth      int    `mapstructure:"max_width"`
	MaxHeight     int    `mapstructure:"max_height"`
	Quality       int    `mapstructure:"quality"`
	ConvertToJPEG bool   `mapstructure:"convert_to_jpeg"`
}

// VideoConfig holds video processing configuration
type VideoConfig struct {
	MaxSize        string `mapstructure:"max_size"`
	MaxDuration    int    `mapstructure:"max_duration"`
	ConvertGIFToMP4 bool  `mapstructure:"convert_gif_to_mp4"`
}

// DocumentConfig holds document processing configuration
type DocumentConfig struct {
	MaxSize string `mapstructure:"max_size"`
	Rename  bool   `mapstructure:"rename"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Encoding   string `mapstructure:"encoding"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	IncludeCaller bool `mapstructure:"include_caller"`
	IncludeStacktrace bool `mapstructure:"include_stacktrace"`
}

// ErrorHandlingConfig holds error handling configuration
type ErrorHandlingConfig struct {
	Retry          RetryConfig          `mapstructure:"retry"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	Checkpoint     CheckpointConfig     `mapstructure:"checkpoint"`
	PanicRecovery  bool                 `mapstructure:"panic_recovery"`
	MaxErrorsBeforeStop int             `mapstructure:"max_errors_before_stop"`
}

// RetryConfig holds retry logic configuration
type RetryConfig struct {
	MaxAttempts int      `mapstructure:"max_attempts"`
	BaseDelay   Duration `mapstructure:"base_delay"`
	MaxDelay    Duration `mapstructure:"max_delay"`
	Jitter      bool     `mapstructure:"jitter"`
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int      `mapstructure:"failure_threshold"`
	ResetTimeout     Duration `mapstructure:"reset_timeout"`
}

// CheckpointConfig holds checkpoint system configuration
type CheckpointConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	FilePath    string `mapstructure:"file_path"`
	SaveInterval Duration `mapstructure:"save_interval"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Telegram validation
	if c.Telegram.BotToken == "" {
		errors = append(errors, "telegram.bot_token is required")
	}
	if c.Telegram.ChannelID == "" && c.Telegram.ChatID == 0 {
		errors = append(errors, "telegram.channel_id or telegram.chat_id is required")
	}
	if c.Telegram.RequestsPerSecond <= 0 {
		errors = append(errors, "telegram.requests_per_second must be positive")
	}
	if c.Telegram.MaxRetries < 0 {
		errors = append(errors, "telegram.max_retries cannot be negative")
	}

	// VK validation
	if c.VK.AccessToken == "" {
		errors = append(errors, "vk.access_token is required")
	}
	if c.VK.GroupID == 0 {
		errors = append(errors, "vk.group_id is required")
	}
	if c.VK.RequestsPerSecond <= 0 {
		errors = append(errors, "vk.requests_per_second must be positive")
	}
	if c.VK.MaxRetries < 0 {
		errors = append(errors, "vk.max_retries cannot be negative")
	}

	// Transfer validation
	if c.Transfer.MaxPostsPerBatch <= 0 {
		errors = append(errors, "transfer.max_posts_per_batch must be positive")
	}
	if c.Transfer.MaxAttachmentsPerPost <= 0 {
		errors = append(errors, "transfer.max_attachments_per_post must be positive")
	}

	// Media validation
	if c.Media.TempDir == "" {
		errors = append(errors, "media.temp_dir is required")
	}
	if c.Media.MaxConcurrentDownloads <= 0 {
		errors = append(errors, "media.max_concurrent_downloads must be positive")
	}
	if c.Media.MaxConcurrentUploads <= 0 {
		errors = append(errors, "media.max_concurrent_uploads must be positive")
	}

	// Date validation
	if c.Telegram.StartDate != nil && c.Telegram.EndDate != nil {
		if c.Telegram.StartDate.After(*c.Telegram.EndDate) {
			errors = append(errors, "telegram.start_date must be before end_date")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
