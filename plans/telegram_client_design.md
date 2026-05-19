# Telegram API Client Design

## Overview
Client for fetching posts from Telegram channels, including text and media attachments.

## API Selection

### Options Considered
1. **Telegram Bot API** (Recommended)
   - Pros: Simple, well-documented, no phone number required
   - Cons: Limited to public channels or channels where bot is admin
   - Rate limits: 30 messages per second for bots

2. **Telegram MTProto (User API)**
   - Pros: Can access private channels, more control
   - Cons: Complex, requires phone verification, risk of ban

**Decision**: Use Bot API for simplicity unless private channel access is required.

## Bot API Endpoints

### Key Endpoints
1. `getUpdates` - For real-time updates (not needed for historical)
2. `getChat` - Get channel info
3. `getChatAdministrators` - Check bot permissions
4. `getChatMembersCount` - Get subscriber count
5. `getChatMember` - Check specific user
6. **`getChatHistory`** - Primary endpoint for fetching messages
   - Actually uses `getUpdates` with offset or `getChatHistory` via bot API limitations
   - Alternative: Use `forwardMessage` approach or iterate through message IDs

### Actual Implementation Approach
Since Bot API doesn't have direct `getChatHistory` for channels, we need to:
1. Add bot to channel as admin
2. Use `getUpdates` with large offset to get historical messages
3. Or use iterative approach with message ID ranges

**Better Approach**: Use `getChat` with `chat_id` and iterate through message IDs using `getChatHistory` (available in newer Bot API versions for channels where bot is admin).

## Data Structures

### Go Types
```go
package telegram

import "time"

// MediaType represents different media types
type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeVoice    MediaType = "voice"
	MediaTypeSticker  MediaType = "sticker"
	MediaTypeAnimation MediaType = "animation" // GIF
)

// Media represents a media attachment
type Media struct {
	Type        MediaType
	FileID      string
	FileUniqueID string
	FileSize    int64
	Width       int  // for photos/videos
	Height      int  // for photos/videos
	Duration    int  // for videos/audio in seconds
	FileName    string // for documents
	MimeType    string // for documents
	Caption     string
}

// Post represents a Telegram channel post
type Post struct {
	ID             int64
	Date           time.Time
	EditDate       *time.Time
	Text           string
	Media          []Media
	ForwardedFrom  *ForwardInfo
	Views          int
	Reactions      []Reaction
	Link           string // t.me/channel/123
	HasSpoiler     bool
}

// ForwardInfo contains information about forwarded message
type ForwardInfo struct {
	FromChatID   int64
	FromChatName string
	MessageID    int64
	Date         time.Time
}

// Reaction represents a message reaction
type Reaction struct {
	Emoji string
	Count int
}

// ChannelInfo contains channel metadata
type ChannelInfo struct {
	ID          int64
	Username    string
	Title       string
	Description string
	Members     int
	PhotoURL    string
	IsPublic    bool
}
```

## Client Interface

```go
package telegram

import (
	"context"
	"time"
)

// Client interface for Telegram operations
type Client interface {
	// GetChannelInfo fetches channel metadata
	GetChannelInfo(ctx context.Context, channelID string) (*ChannelInfo, error)
	
	// GetPosts fetches posts from channel with pagination
	GetPosts(ctx context.Context, channelID string, opts GetPostsOptions) ([]Post, error)
	
	// GetPostByID fetches a specific post
	GetPostByID(ctx context.Context, channelID string, postID int64) (*Post, error)
	
	// GetMediaURL generates download URL for media file
	GetMediaURL(ctx context.Context, fileID string) (string, error)
	
	// DownloadMedia downloads media to local file
	DownloadMedia(ctx context.Context, fileID, destPath string) error
	
	// Close releases resources
	Close() error
}

// GetPostsOptions defines parameters for fetching posts
type GetPostsOptions struct {
	Limit      int       // Maximum number of posts to fetch (0 = default)
	OffsetID   int64     // ID of the message to start from (0 = most recent)
	OffsetDate time.Time // Date to start from
	Reverse    bool      // false = newest first, true = oldest first
	MaxID      int64     // Maximum message ID to fetch
	MinID      int64     // Minimum message ID to fetch
}

// Config holds Telegram client configuration
type Config struct {
	BotToken  string        `yaml:"bot_token" env:"TELEGRAM_BOT_TOKEN"`
	APIURL    string        `yaml:"api_url" env:"TELEGRAM_API_URL"` // Default: https://api.telegram.org
	Timeout   time.Duration `yaml:"timeout" env:"TELEGRAM_TIMEOUT"` // Default: 30s
	UserAgent string        `yaml:"user_agent" env:"TELEGRAM_USER_AGENT"`
	
	// Rate limiting
	RequestsPerSecond int `yaml:"requests_per_second" env:"TELEGRAM_RPS_LIMIT"` // Default: 20
	MaxRetries        int `yaml:"max_retries" env:"TELEGRAM_MAX_RETRIES"`       // Default: 3
}
```

## Implementation Details

### Using go-telegram-bot-api Library
```go
import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type BotAPIClient struct {
	bot     *tgbotapi.BotAPI
	config  Config
	limiter *rate.Limiter
	logger  *zap.Logger
}

func NewBotAPIClient(config Config) (*BotAPIClient, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	
	bot.Debug = config.Debug
	
	// Set custom HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}
	bot.Client = client
	
	return &BotAPIClient{
		bot:     bot,
		config:  config,
		limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 1),
		logger:  zap.L().Named("telegram"),
	}, nil
}
```

### Fetching Posts Implementation
The challenge: Bot API doesn't provide direct access to channel history for bots that aren't admins.

**Solution 1: Bot as Admin**
If bot is added as channel admin:
```go
func (c *BotAPIClient) GetPosts(ctx context.Context, channelID string, opts GetPostsOptions) ([]Post, error) {
	var posts []Post
	var offsetID int64 = 0
	
	for {
		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
		
		// Create request
		cfg := tgbotapi.GetChatHistoryConfig{
			ChatID:          channelID,
			Limit:           opts.Limit,
			OffsetID:        offsetID,
		}
		
		messages, err := c.bot.GetChatHistory(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to get chat history: %w", err)
		}
		
		if len(messages) == 0 {
			break
		}
		
		// Convert messages to Post structs
		for _, msg := range messages {
			post := c.convertMessage(msg)
			posts = append(posts, post)
			offsetID = msg.MessageID
		}
		
		// Check if we have enough posts
		if opts.Limit > 0 && len(posts) >= opts.Limit {
			posts = posts[:opts.Limit]
			break
		}
		
		// Small delay to avoid hitting rate limits
		time.Sleep(100 * time.Millisecond)
	}
	
	return posts, nil
}
```

**Solution 2: Public Channel Scraping**
For public channels where bot isn't admin, use alternative methods:
1. Use `getUpdates` with large offset (limited to last 24 hours)
2. Use Telegram's JSON export feature (manual)
3. Use third-party libraries that simulate user client

**Recommendation**: Document that bot needs to be channel admin for full access.

### Media Handling
```go
func (c *BotAPIClient) DownloadMedia(ctx context.Context, fileID, destPath string) error {
	// Get file info first
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Construct download URL
	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", 
		c.config.BotToken, file.FilePath)
	
	// Download with retry
	return c.downloadWithRetry(ctx, downloadURL, destPath)
}

func (c *BotAPIClient) downloadWithRetry(ctx context.Context, url, destPath string) error {
	var lastErr error
	
	for i := 0; i < c.config.MaxRetries; i++ {
		if i > 0 {
			// Exponential backoff
			delay := time.Duration(math.Pow(2, float64(i))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		err := c.downloadFile(ctx, url, destPath)
		if err == nil {
			return nil
		}
		
		lastErr = err
		c.logger.Warn("Download failed, retrying", 
			zap.String("url", url), 
			zap.Int("attempt", i+1),
			zap.Error(err))
	}
	
	return fmt.Errorf("failed after %d retries: %w", c.config.MaxRetries, lastErr)
}
```

## Error Handling

### Common Errors
1. `400 Bad Request` - Invalid parameters
2. `401 Unauthorized` - Invalid bot token
3. `403 Forbidden` - Bot not admin in channel
4. `429 Too Many Requests` - Rate limit exceeded
5. `502 Bad Gateway` - Telegram server issues

### Retry Strategy
- Exponential backoff: 1s, 2s, 4s, 8s
- Maximum 3 retries for transient errors
- Circuit breaker pattern for persistent failures

## Rate Limiting
- Default: 20 requests per second (Telegram's limit is 30/sec for bots)
- Implement token bucket algorithm
- Respect `Retry-After` header when provided

## Testing Strategy

### Unit Tests
- Mock HTTP responses
- Test error handling
- Test rate limiting

### Integration Tests
- Use test bot token
- Test with public test channel
- Verify media download

### Sample Test Channel
Create a test channel with:
1. Text posts
2. Posts with photos
3. Posts with videos
4. Posts with documents
5. Forwarded messages
6. Posts with reactions

## Configuration Example
```yaml
telegram:
  bot_token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  api_url: "https://api.telegram.org"
  timeout: 30s
  requests_per_second: 20
  max_retries: 3
  debug: false
```

## Dependencies
```go
import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
	"go.uber.org/zap"
)
```

## Next Steps
1. Implement basic client with authentication
2. Add channel info fetching
3. Implement post fetching with pagination
4. Add media download functionality
5. Implement error handling and retries
6. Write comprehensive tests