package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Loader handles configuration loading and validation
type Loader struct {
	viper *viper.Viper
	logger *zap.Logger
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()
	
	// Configure viper
	v.SetConfigName("config")           // config file name without extension
	v.SetConfigType("yaml")             // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath(".")                // look for config in current directory
	v.AddConfigPath("./configs")        // look for config in configs directory
	v.AddConfigPath("$HOME/.tg2vk")     // look for config in home directory
	
	// Environment variables
	v.SetEnvPrefix("TG2VK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Set defaults
	setViperDefaults(v)
	
	return &Loader{
		viper: v,
		logger: zap.L().Named("config"),
	}
}

// Load loads configuration from file, environment variables, and defaults
func (l *Loader) Load(configPath string) (*Config, error) {
	// If config path is provided, use it
	if configPath != "" {
		l.viper.SetConfigFile(configPath)
	}
	
	// Read config
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			l.logger.Warn("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	} else {
		l.logger.Info("Loaded config file", 
			zap.String("file", l.viper.ConfigFileUsed()))
	}
	
	// Unmarshal config
	var cfg Config
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	
	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	l.logger.Info("Configuration loaded successfully")
	return &cfg, nil
}

// GenerateExampleConfig generates an example configuration file
func (l *Loader) GenerateExampleConfig(outputPath string) error {
	exampleConfig := `# Telegram to VK Transfer Configuration
# Minimal configuration - just fill in the 4 required values below!

telegram:
  # REQUIRED: Bot token from @BotFather
  bot_token: "YOUR_BOT_TOKEN_HERE"
  
  # REQUIRED: Channel identifier (use @username for public channels)
  channel_id: "@example_channel"
  
  # Optional: Most users won't need to change these
  # api_url: "https://api.telegram.org"
  # timeout: "30s"
  # requests_per_second: 20
  # max_retries: 3
  # include_media: true
  # max_file_size: "100MB"

vk:
  # REQUIRED: Access token with required permissions
  access_token: "YOUR_VK_ACCESS_TOKEN_HERE"
  
  # REQUIRED: Group ID
  group_id: 123456789
  
  # Optional: Sensible defaults
  # api_version: "5.199"
  # timeout: "60s"
  # language: "ru"
  # requests_per_second: 3.0
  # max_retries: 3
  # from_group: true
  # signed: true
  # post_delay: "3s"

transfer:
  # Safety first: dry-run by default (set to false for actual transfer)
  dry_run: true
  
  # Content handling
  include_text: true
  include_media: true
  
  # Smart defaults
  skip_duplicates: true
  preserve_formatting: true
  add_source_link: true
  source_link_format: "Источник: %s"

media:
  # Temporary storage (created automatically)
  temp_dir: "./temp"
  keep_files: false
  
  # Concurrent operations (optimized for stability)
  max_concurrent_downloads: 3
  max_concurrent_uploads: 2

# Advanced settings (most users don't need to change these)
# logging:
#   level: "info"
#   encoding: "json"
#   output_path: "./logs/transfer.log"
#
# error_handling:
#   retry:
#     max_attempts: 3
#     base_delay: "1s"
#     max_delay: "30s"
#     jitter: true
`

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(outputPath, []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("write example config: %w", err)
	}
	
	l.logger.Info("Example config generated", zap.String("path", outputPath))
	return nil
}

// setViperDefaults sets default values in viper
func setViperDefaults(v *viper.Viper) {
	// Telegram defaults
	v.SetDefault("telegram.api_url", "https://api.telegram.org")
	v.SetDefault("telegram.timeout", "30s")
	v.SetDefault("telegram.debug", false)
	v.SetDefault("telegram.requests_per_second", 20)
	v.SetDefault("telegram.max_retries", 3)
	v.SetDefault("telegram.batch_size", 100)
	v.SetDefault("telegram.include_media", true)
	v.SetDefault("telegram.max_file_size", 104857600) // 100MB
	
	// VK defaults
	v.SetDefault("vk.api_url", "https://api.vk.com/method")
	v.SetDefault("vk.api_version", "5.199")
	v.SetDefault("vk.timeout", "60s")
	v.SetDefault("vk.language", "ru")
	v.SetDefault("vk.requests_per_second", 3.0)
	v.SetDefault("vk.max_retries", 3)
	v.SetDefault("vk.from_group", true)
	v.SetDefault("vk.signed", true)
	v.SetDefault("vk.friends_only", false)
	v.SetDefault("vk.post_delay", "3s")
	v.SetDefault("vk.album_id", 0)
	
	// Transfer defaults
	v.SetDefault("transfer.dry_run", true) // Safety first!
	v.SetDefault("transfer.skip_duplicates", true)
	v.SetDefault("transfer.preserve_dates", false)
	v.SetDefault("transfer.include_text", true)
	v.SetDefault("transfer.include_media", true)
	v.SetDefault("transfer.media_types", []string{"photo", "video", "document", "audio"})
	v.SetDefault("transfer.preserve_formatting", true)
	v.SetDefault("transfer.add_source_link", true)
	v.SetDefault("transfer.source_link_format", "Источник: %s")
	v.SetDefault("transfer.include_forward_info", true)
	v.SetDefault("transfer.forward_format", "↪️ Переслано из %s")
	v.SetDefault("transfer.max_posts_per_batch", 100)
	v.SetDefault("transfer.max_attachments_per_post", 10)
	v.SetDefault("transfer.skip_failed_posts", true)
	v.SetDefault("transfer.max_errors_before_stop", 10)
	v.SetDefault("transfer.stop_on_auth_error", true)
	
	// Media defaults
	v.SetDefault("media.temp_dir", "./temp")
	v.SetDefault("media.keep_files", false)
	v.SetDefault("media.max_concurrent_downloads", 3)
	v.SetDefault("media.max_concurrent_uploads", 2)
	v.SetDefault("media.download_timeout", "30s")
	v.SetDefault("media.upload_timeout", "300s")
	v.SetDefault("media.download_retries", 3)
	
	// Photo defaults
	v.SetDefault("media.photos.max_width", 1920)
	v.SetDefault("media.photos.max_height", 1080)
	v.SetDefault("media.photos.quality", 85)
	v.SetDefault("media.photos.convert_to_jpeg", true)
	
	// Video defaults
	v.SetDefault("media.videos.max_size", "2GB")
	v.SetDefault("media.videos.max_duration", 600)
	v.SetDefault("media.videos.convert_gif_to_mp4", true)
	
	// Document defaults
	v.SetDefault("media.documents.max_size", "2GB")
	v.SetDefault("media.documents.rename", true)
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.encoding", "json")
	v.SetDefault("logging.output_path", "")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_backups", 10)
	v.SetDefault("logging.max_age", 30)
	v.SetDefault("logging.include_caller", true)
	v.SetDefault("logging.include_stacktrace", true)
	
	// Error handling defaults
	v.SetDefault("error_handling.retry.max_attempts", 3)
	v.SetDefault("error_handling.retry.base_delay", "1s")
	v.SetDefault("error_handling.retry.max_delay", "30s")
	v.SetDefault("error_handling.retry.jitter", true)
	v.SetDefault("error_handling.circuit_breaker.failure_threshold", 5)
	v.SetDefault("error_handling.circuit_breaker.reset_timeout", "60s")
	v.SetDefault("error_handling.checkpoint.enabled", true)
	v.SetDefault("error_handling.checkpoint.file_path", "./checkpoints/last_run.json")
	v.SetDefault("error_handling.checkpoint.save_interval", "60s")
	v.SetDefault("error_handling.panic_recovery", true)
	v.SetDefault("error_handling.max_errors_before_stop", 100)
}

// GetConfigFileUsed returns the path of the config file that was used
func (l *Loader) GetConfigFileUsed() string {
	return l.viper.ConfigFileUsed()
}