# Telegram to VK Content Transfer - Architecture Design

## Project Overview
A pure Go application for one-time transfer of all existing posts (including text, photos, videos, and documents) from a Telegram channel to a VK group/channel.

## Requirements Summary
- **Transfer Type**: One-time historical data transfer
- **Content Types**: Text with photos, videos, and documents
- **Architecture**: Pure Go application with configuration file (no UI)
- **Platforms**: Telegram API → VK API

## System Architecture

### High-Level Data Flow
```
Telegram Channel → Telegram API Client → Content Fetcher → Media Downloader → Content Transformer → VK API Client → VK Group/Channel
```

### Components

1. **Configuration Manager**
   - Reads `config.yaml` or `config.json`
   - Stores API credentials, channel IDs, and settings
   - Validates required parameters

2. **Telegram API Client**
   - Uses Telegram Bot API or MTProto (depending on access)
   - Fetches channel posts with pagination
   - Extracts text, media URLs, and metadata
   - Handles rate limiting and errors

3. **Media Downloader**
   - Downloads photos, videos, documents to local temp storage
   - Maintains original file names and formats
   - Implements retry logic for failed downloads

4. **Content Transformer**
   - Converts Telegram message format to VK post format
   - Handles text formatting (Markdown → HTML if needed)
   - Manages media type conversions (if required)
   - Preserves post order and timestamps

5. **VK API Client**
   - Uses VK API with proper authentication
   - Uploads media to VK servers (different endpoints for photos/videos/docs)
   - Creates posts with attached media
   - Handles VK API limitations and quotas

6. **Error Handler & Logger**
   - Structured logging for debugging
   - Error recovery and retry mechanisms
   - Progress tracking and reporting

## Data Structures

### Telegram Post
```go
type TelegramPost struct {
    ID          int64
    Date        time.Time
    Text        string
    Media       []TelegramMedia
    ForwardedFrom *ForwardInfo
    // ... other fields
}

type TelegramMedia struct {
    Type    string // "photo", "video", "document", "audio"
    URL     string
    FileID  string
    Caption string
    FileSize int64
}
```

### VK Post
```go
type VKPost struct {
    Text     string
    Attachments []VKAttachment
    PublishDate *time.Time
    FromGroup  bool
}

type VKAttachment struct {
    Type string // "photo", "video", "doc", "audio"
    OwnerID int
    MediaID int
    URL     string
}
```

## Configuration File Format
```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN"
  channel_id: "@channel_username"
  # or use: chat_id: -1001234567890

vk:
  access_token: "VK_ACCESS_TOKEN"
  group_id: 123456789
  album_id: 0 # optional for photos

transfer:
  start_date: "2020-01-01" # optional
  end_date: "2023-12-31"   # optional
  include_media: true
  max_posts: 0 # 0 for all
  dry_run: false

storage:
  temp_dir: "./temp"
  keep_files: false

logging:
  level: "info"
  file: "transfer.log"
```

## API Considerations

### Telegram API Options
1. **Bot API** (simpler, but limited to public channels or channels where bot is admin)
   - Requires bot token
   - Can fetch up to 100 messages per request
   - Media accessible via file URLs

2. **MTProto (User API)** (more powerful, can access private channels)
   - Requires phone number authentication
   - More complex implementation
   - Can access all channel types

**Recommendation**: Start with Bot API for simplicity unless private channel access is needed.

### VK API Limitations
- Photo upload: separate endpoint for each photo
- Video upload: multi-step process (get upload server, upload file, save)
- Document upload: similar to photos
- Rate limits: ~3 requests per second
- Post size limits: text up to 10,000 characters

## Error Handling Strategy
1. **Retry with exponential backoff** for transient API errors
2. **Checkpoint system** to resume interrupted transfers
3. **Detailed error logging** with context
4. **Graceful degradation** (skip failed media, continue with text)

## Security Considerations
- Store credentials in configuration file (not in code)
- Use environment variables for sensitive data
- Clean up temporary files after transfer
- Validate input to prevent injection attacks

## Implementation Phases

### Phase 1: Core Infrastructure
- Project setup with Go modules
- Configuration system
- Basic logging
- Error handling framework

### Phase 2: Telegram Integration
- Telegram API client
- Message fetching with pagination
- Basic text transfer

### Phase 3: Media Handling
- Media download from Telegram
- Local storage management
- File type detection

### Phase 4: VK Integration
- VK API client
- Media upload procedures
- Post creation with attachments

### Phase 5: Integration & Testing
- End-to-end transfer
- Error recovery
- Performance optimization

## Dependencies
- Go 1.19+
- Telegram Bot API library (e.g., `go-telegram-bot-api`)
- VK API library (e.g., `github.com/SevereCloud/vksdk`)
- YAML/JSON config parsing
- HTTP client with retry support

## Next Steps
1. Create project structure
2. Implement configuration system
3. Build Telegram client prototype
4. Test with sample channel
5. Iterate based on results