# Project Structure and Configuration

## Directory Layout
```
ai-transfer-tg-to-vk/
├── cmd/
│   └── transfer/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration loading and validation
│   ├── telegram/
│   │   ├── client.go        # Telegram API client
│   │   ├── fetcher.go       # Post fetching logic
│   │   └── types.go         # Telegram data structures
│   ├── vk/
│   │   ├── client.go        # VK API client
│   │   ├── uploader.go      # Media upload logic
│   │   └── types.go         # VK data structures
│   ├── media/
│   │   ├── downloader.go    # Media download from Telegram
│   │   ├── processor.go     # Media processing and conversion
│   │   └── storage.go       # Temporary file management
│   ├── transfer/
│   │   ├── orchestrator.go  # Main transfer orchestration
│   │   ├── transformer.go   # Content transformation
│   │   └── checkpoint.go    # Progress tracking and resumption
│   └── logger/
│       └── logger.go        # Structured logging setup
├── pkg/
│   └── utils/
│       ├── retry.go         # Retry with exponential backoff
│       └── http.go          # HTTP client utilities
├── configs/
│   ├── config.example.yaml  # Example configuration
│   └── config.yaml          # User configuration (gitignored)
├── scripts/
│   └── setup.sh             # Setup script (optional)
├── plans/                   # Architecture and planning docs
├── go.mod                   # Go module definition
├── go.sum                   # Go dependencies
├── .gitignore
├── README.md
└── Makefile                 # Build and run commands
```

## Configuration Files

### config.example.yaml
```yaml
# Telegram Configuration
telegram:
  # Bot token from @BotFather
  bot_token: "YOUR_BOT_TOKEN_HERE"
  
  # Channel identifier (use @username for public channels or numeric ID for private)
  channel_id: "@example_channel"
  
  # Alternative: use chat_id (numeric, negative for channels)
  # chat_id: -1001234567890
  
  # Maximum number of posts to fetch (0 = all)
  max_posts: 0
  
  # Date range for filtering posts (optional)
  # start_date: "2020-01-01T00:00:00Z"
  # end_date: "2023-12-31T23:59:59Z"

# VK Configuration
vk:
  # Access token with wall.post, photos.getUploadServer, video.save, docs.getUploadServer permissions
  access_token: "YOUR_VK_ACCESS_TOKEN_HERE"
  
  # Group ID (positive for groups, negative for user walls)
  group_id: 123456789
  
  # Album ID for photos (0 = wall photos)
  album_id: 0
  
  # Post as group (true) or as user (false)
  from_group: true
  
  # Delay between posts (in seconds) to avoid rate limiting
  post_delay: 3

# Transfer Settings
transfer:
  # Include media attachments
  include_media: true
  
  # Media types to transfer
  media_types:
    - photo
    - video
    - document
    - audio
  
  # Maximum file size in bytes (0 = unlimited)
  max_file_size: 104857600  # 100MB
  
  # Dry run mode (fetch but don't post)
  dry_run: false
  
  # Skip posts that already exist in VK (based on text hash)
  skip_duplicates: true
  
  # Preserve original post dates (VK may ignore this)
  preserve_dates: false

# Storage Settings
storage:
  # Temporary directory for downloaded media
  temp_dir: "./temp"
  
  # Keep temporary files after transfer (for debugging)
  keep_files: false
  
  # Maximum concurrent downloads
  max_concurrent_downloads: 3

# Logging Settings
logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log file path (empty for stdout only)
  file: "transfer.log"
  
  # JSON format for structured logging
  json_format: false
  
  # Include timestamps in logs
  timestamps: true

# Performance Settings
performance:
  # HTTP request timeout in seconds
  http_timeout: 30
  
  # Maximum retry attempts for failed operations
  max_retries: 3
  
  # Retry delay base in seconds
  retry_delay: 2
  
  # Concurrent API requests
  max_concurrent_requests: 2
```

### .gitignore
```
# Binaries
/dist/
/transfer
/transfer.exe

# Configuration
configs/config.yaml
configs/*.token

# Temporary files
/temp/
*.tmp
*.temp

# Logs
*.log
logs/

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Go files
go.sum
```

### Makefile
```makefile
.PHONY: build run test clean setup

BINARY_NAME=transfer
CONFIG_PATH=configs/config.yaml

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/transfer

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --config $(CONFIG_PATH)

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf temp/
	rm -f *.log

setup:
	@echo "Setting up project..."
	cp configs/config.example.yaml configs/config.yaml
	@echo "Please edit configs/config.yaml with your credentials"

deps:
	@echo "Downloading dependencies..."
	go mod download

lint:
	@echo "Running linter..."
	golangci-lint run

docker-build:
	@echo "Building Docker image..."
	docker build -t ai-transfer-tg-to-vk .

help:
	@echo "Available commands:"
	@echo "  build      - Build the application"
	@echo "  run        - Build and run with config"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  setup      - Copy example config"
	@echo "  deps       - Download dependencies"
	@echo "  lint       - Run linter"
	@echo "  docker-build - Build Docker image"
```

## Dependencies

### Required Go Packages
1. **Telegram Bot API**: `github.com/go-telegram-bot-api/telegram-bot-api/v5`
2. **VK API SDK**: `github.com/SevereCloud/vksdk/v2`
3. **Configuration**: `github.com/spf13/viper` (supports YAML, JSON, env vars)
4. **Logging**: `go.uber.org/zap` (structured logging)
5. **HTTP Client**: Standard library with custom retry logic

### Installation Command
```bash
go get github.com/go-telegram-bot-api/telegram-bot-api/v5 \
        github.com/SevereCloud/vksdk/v2 \
        github.com/spf13/viper \
        go.uber.org/zap
```

## Environment Variables Support
The application will also support environment variables for sensitive data:
- `TELEGRAM_BOT_TOKEN`
- `VK_ACCESS_TOKEN`
- `VK_GROUP_ID`

Environment variables will override configuration file values.

## Next Steps
1. Create the directory structure
2. Initialize Go module with dependencies
3. Implement configuration loader
4. Create basic logging setup
5. Build Telegram client prototype