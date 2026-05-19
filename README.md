# Telegram to VK Content Transfer

A pure Go application for one-time transfer of historical content (text, photos, videos, documents) from a Telegram channel to a VK group/channel.

## Features

- **Complete Content Transfer**: Transfer text, photos, videos, and documents
- **Safe by Default**: Dry-run mode enabled by default to prevent accidental posting
- **Resumable Transfers**: Checkpoint system allows resuming interrupted transfers
- **Smart Formatting**: Preserves text formatting and converts media appropriately
- **Rate Limiting**: Respects both Telegram and VK API rate limits
- **Structured Logging**: Comprehensive logging for debugging and monitoring

## Quick Start

### Prerequisites

- Go 1.19 or later
- Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- VK Access Token with required permissions
- VK Group ID

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/ai-transfer-tg-to-vk.git
   cd ai-transfer-tg-to-vk
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Generate example configuration:
   ```bash
   make generate-config
   ```

4. Edit the configuration file:
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   # Edit configs/config.yaml with your credentials
   ```

### Configuration

Minimal configuration requires just 4 values:

```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN_HERE"
  channel_id: "@example_channel"

vk:
  access_token: "YOUR_VK_ACCESS_TOKEN_HERE"
  group_id: 123456789
```

See `configs/config.example.yaml` for all available options.

### Usage

1. **Dry-run (recommended first)**: The default configuration has `dry_run: true`
   ```bash
   make run
   ```

2. **Actual transfer**: Set `dry_run: false` in your config and run:
   ```bash
   make run
   ```

3. **Build and run**:
   ```bash
   make build
   ./bin/transfer
   ```

## Project Structure

```
ai-transfer-tg-to-vk/
├── cmd/transfer/main.go          # Application entry point
├── internal/
│   ├── config/                   # Configuration loading/validation
│   ├── telegram/                 # Telegram API client
│   ├── vk/                       # VK API client
│   ├── media/                    # Media download/upload
│   ├── transfer/                 # Core transfer logic
│   ├── transformer/              # Content transformation
│   └── logger/                   # Structured logging
├── configs/                      # Configuration files
├── plans/                        # Architecture and design documents
├── scripts/                      # Build and utility scripts
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Development

### Building

```bash
make build          # Build the application
make build-linux    # Build for Linux
make build-windows  # Build for Windows
make build-mac      # Build for macOS
```

### Testing

```bash
make test           # Run unit tests
make test-verbose   # Run tests with verbose output
```

### Code Quality

```bash
make fmt            # Format Go code
make vet            # Run go vet
make lint           # Run golint (requires installation)
make all            # Run all checks and build
```

### Dependencies

```bash
make deps           # Download dependencies
make tidy           # Tidy go.mod
```

## Design Principles

1. **Safety First**: Dry-run mode by default, validation before execution
2. **Minimal Configuration**: Only 4 values required to start
3. **Progressive Enhancement**: Start with text-only, add media gradually
4. **Resilience**: Retry logic, circuit breakers, checkpointing
5. **Transparency**: Detailed logging and progress reporting

## API Requirements

### Telegram
- Bot token with access to the channel
- Bot must be added as administrator to private channels

### VK
- Access token with these permissions: `wall`, `photos`, `video`, `docs`, `groups`
- User must be administrator of the target group

## Limitations

- **One-time transfer**: Designed for historical data migration, not real-time sync
- **Media size limits**: Respects VK's file size limits (photos: 50MB, videos: 2GB, docs: 2GB)
- **Rate limits**: Transfers large channels slowly to avoid API restrictions
- **Formatting**: Some Telegram formatting may not translate perfectly to VK

## Troubleshooting

### Common Issues

1. **"Config file not found"**: Ensure `configs/config.yaml` exists or use `--config` flag
2. **"Invalid bot token"**: Verify your Telegram bot token and ensure the bot has channel access
3. **"VK authentication failed"**: Check your access token permissions and group ID
4. **"Rate limit exceeded"**: The application will automatically retry with exponential backoff

### Logs

Logs are written to stdout by default. To write to a file, set `logging.output_path` in config.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[MIT License](LICENSE)

## Acknowledgments

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) for Telegram API
- [vksdk](https://github.com/SevereCloud/vksdk) for VK API
- [viper](https://github.com/spf13/viper) for configuration
- [zap](https://github.com/uber-go/zap) for logging

## Support

For issues and questions, please open an issue on GitHub.