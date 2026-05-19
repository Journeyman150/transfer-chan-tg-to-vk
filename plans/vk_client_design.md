# VK API Client Design

## Overview
Client for posting content to VK (VKontakte) groups/channels, including text and media attachments.

## VK API Basics

### Authentication
- **Access Token**: Required for all API calls
- **Permissions Needed**:
  - `wall` - post to wall, read wall
  - `photos` - upload photos
  - `video` - upload videos  
  - `docs` - upload documents
  - `groups` - manage groups (if posting to group)
  - `offline` - token doesn't expire (for long-running transfers)

### API Limits
- **Rate Limits**: ~3 requests per second
- **Daily Limits**: Varies by method, typically 5000-10000 calls/day
- **Upload Limits**: 
  - Photos: up to 5MB (JPG, PNG, GIF)
  - Videos: up to 2GB (MP4, AVI, etc.)
  - Documents: up to 2GB (any file type)

## Key API Methods

### Wall Posts
- `wall.post` - create a post on user or group wall
- `wall.edit` - edit existing post
- `wall.getById` - get post by ID

### Photo Upload
1. `photos.getWallUploadServer` - get upload URL for wall photos
2. Upload photo via POST to returned URL
3. `photos.saveWallPhoto` - save uploaded photo to wall
4. Use returned `photo` object in attachments

### Video Upload
1. `video.save` - get upload URL with video details
2. Upload video via POST to returned URL
3. `video.save` (again) - confirm upload completion
4. Use returned `video` object in attachments

### Document Upload
1. `docs.getWallUploadServer` - get upload URL for documents
2. Upload document via POST to returned URL
3. `docs.save` - save uploaded document
4. Use returned `doc` object in attachments

## Data Structures

### Go Types
```go
package vk

import "time"

// AttachmentType represents VK attachment types
type AttachmentType string

const (
	AttachmentTypePhoto   AttachmentType = "photo"
	AttachmentTypeVideo   AttachmentType = "video"
	AttachmentTypeDoc     AttachmentType = "doc"
	AttachmentTypeAudio   AttachmentType = "audio"
	AttachmentTypeLink    AttachmentType = "link"
	AttachmentTypePoll    AttachmentType = "poll"
)

// Attachment represents a VK attachment
type Attachment struct {
	Type    AttachmentType
	OwnerID int    // Negative for groups
	MediaID int
	AccessKey string // For private content
}

// Post represents a VK wall post
type Post struct {
	Text        string
	Attachments []Attachment
	PublishDate *time.Time
	FromGroup   bool      // true for group posts, false for user
	Signed      bool      // Show author signature (for groups)
	FriendsOnly bool      // Visible to friends only
	Services    string    // "twitter", "facebook" for cross-posting
	Lat         float64   // Latitude for location
	Long        float64   // Longitude for location
	PlaceID     int       // Location ID
}

// UploadResult represents uploaded media
type UploadResult struct {
	Server   int    `json:"server"`
	Photo    string `json:"photo"`    // For photos
	Video    string `json:"video"`    // For videos
	File     string `json:"file"`     // For documents
	Hash     string `json:"hash"`
	OwnerID  int    `json:"owner_id"`
	MediaID  int    `json:"id"`
	AccessKey string `json:"access_key"`
}

// PostResult represents created post
type PostResult struct {
	PostID  int `json:"post_id"`
	PostHash string `json:"post_hash"`
}
```

## Client Interface

```go
package vk

import (
	"context"
	"time"
)

// Client interface for VK operations
type Client interface {
	// Post creates a new wall post
	Post(ctx context.Context, post Post) (*PostResult, error)
	
	// UploadPhoto uploads photo to wall
	UploadPhoto(ctx context.Context, filePath string, groupID int) (*Attachment, error)
	
	// UploadVideo uploads video to wall
	UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*Attachment, error)
	
	// UploadDocument uploads document to wall
	UploadDocument(ctx context.Context, filePath, title string, groupID int) (*Attachment, error)
	
	// GetGroupInfo fetches group information
	GetGroupInfo(ctx context.Context, groupID int) (*GroupInfo, error)
	
	// Close releases resources
	Close() error
}

// Config holds VK client configuration
type Config struct {
	AccessToken string        `yaml:"access_token" env:"VK_ACCESS_TOKEN"`
	APIURL      string        `yaml:"api_url" env:"VK_API_URL"` // Default: https://api.vk.com/method
	APIVersion  string        `yaml:"api_version" env:"VK_API_VERSION"` // Default: 5.199
	Timeout     time.Duration `yaml:"timeout" env:"VK_TIMEOUT"` // Default: 60s
	Language    string        `yaml:"language" env:"VK_LANGUAGE"` // Default: "ru"
	
	// Rate limiting
	RequestsPerSecond float64 `yaml:"requests_per_second" env:"VK_RPS_LIMIT"` // Default: 3.0
	MaxRetries        int     `yaml:"max_retries" env:"VK_MAX_RETRIES"`       // Default: 3
	
	// Group settings
	GroupID   int  `yaml:"group_id" env:"VK_GROUP_ID"`
	FromGroup bool `yaml:"from_group" env:"VK_FROM_GROUP"` // Post as group
}

// GroupInfo contains group metadata
type GroupInfo struct {
	ID          int
	Name        string
	ScreenName  string
	Description string
	Members     int
	PhotoURL    string
	Type        string // "group", "page", "event"
}
```

## Implementation Details

### Using vksdk Library
```go
import vk "github.com/SevereCloud/vksdk/v2/api"

type SDKClient struct {
	api      *vk.VK
	config   Config
	limiter  *rate.Limiter
	logger   *zap.Logger
	groupID  int
}

func NewSDKClient(config Config) (*SDKClient, error) {
	api := vk.NewVK(config.AccessToken)
	
	// Set API version
	api.Version = config.APIVersion
	
	// Set language
	api.Lang = config.Language
	
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	api.Client = httpClient
	
	return &SDKClient{
		api:     api,
		config:  config,
		limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 1),
		logger:  zap.L().Named("vk"),
		groupID: config.GroupID,
	}, nil
}
```

### Post Creation
```go
func (c *SDKClient) Post(ctx context.Context, post Post) (*PostResult, error) {
	// Rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	
	// Prepare parameters
	params := vk.Params{
		"owner_id":   c.getOwnerID(post.FromGroup),
		"message":    post.Text,
		"friends_only": boolToInt(post.FriendsOnly),
		"services":   post.Services,
		"signed":     boolToInt(post.Signed),
		"lat":        post.Lat,
		"long":       post.Long,
		"place_id":   post.PlaceID,
	}
	
	// Add attachments
	if len(post.Attachments) > 0 {
		attachments := make([]string, 0, len(post.Attachments))
		for _, att := range post.Attachments {
			attachments = append(attachments, c.formatAttachment(att))
		}
		params["attachments"] = strings.Join(attachments, ",")
	}
	
	// Schedule post if publish date is in future
	if post.PublishDate != nil && post.PublishDate.After(time.Now()) {
		params["publish_date"] = post.PublishDate.Unix()
	}
	
	// Make API call
	var response struct {
		PostID int `json:"post_id"`
	}
	
	err := c.api.WallPost(params, &response)
	if err != nil {
		return nil, fmt.Errorf("wall.post failed: %w", err)
	}
	
	c.logger.Info("Post created",
		zap.Int("post_id", response.PostID),
		zap.Int("attachments", len(post.Attachments)))
	
	return &PostResult{
		PostID: response.PostID,
	}, nil
}

func (c *SDKClient) getOwnerID(fromGroup bool) int {
	if fromGroup {
		return -c.groupID // Negative for groups
	}
	return c.groupID
}

func (c *SDKClient) formatAttachment(att Attachment) string {
	// Format: type{owner_id}_{media_id}_{access_key}
	// Example: photo-123456_7890123
	result := fmt.Sprintf("%s%d_%d", string(att.Type), att.OwnerID, att.MediaID)
	if att.AccessKey != "" {
		result += "_" + att.AccessKey
	}
	return result
}
```

### Photo Upload Process
```go
func (c *SDKClient) UploadPhoto(ctx context.Context, filePath string, groupID int) (*Attachment, error) {
	// 1. Get upload server
	serverResp, err := c.api.PhotosGetWallUploadServer(vk.Params{
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get upload server: %w", err)
	}
	
	// 2. Upload file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("photo", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}
	
	writer.Close()
	
	req, err := http.NewRequestWithContext(ctx, "POST", serverResp.UploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 3. Parse upload response
	var uploadResult UploadResult
	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}
	
	// 4. Save photo
	saveResp, err := c.api.PhotosSaveWallPhoto(vk.Params{
		"group_id": groupID,
		"server":   uploadResult.Server,
		"photo":    uploadResult.Photo,
		"hash":     uploadResult.Hash,
	})
	if err != nil || len(saveResp) == 0 {
		return nil, fmt.Errorf("failed to save photo: %w", err)
	}
	
	photo := saveResp[0]
	return &Attachment{
		Type:      AttachmentTypePhoto,
		OwnerID:   photo.OwnerID,
		MediaID:   photo.ID,
		AccessKey: photo.AccessKey,
	}, nil
}
```

### Video Upload Process (Simplified)
```go
func (c *SDKClient) UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*Attachment, error) {
	// 1. Get upload URL
	resp, err := c.api.VideoSave(vk.Params{
		"name":        title,
		"description": description,
		"group_id":    groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get video upload URL: %w", err)
	}
	
	// 2. Upload file (similar to photo upload)
	// ... upload implementation ...
	
	// 3. Confirm upload
	confirmResp, err := c.api.VideoSave(vk.Params{
		"video_id":    resp.VideoID,
		"owner_id":    resp.OwnerID,
		"name":        title,
		"description": description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to confirm video upload: %w", err)
	}
	
	return &Attachment{
		Type:    AttachmentTypeVideo,
		OwnerID: confirmResp.OwnerID,
		MediaID: confirmResp.ID,
	}, nil
}
```

## Error Handling

### Common VK API Errors
1. `1` - Unknown error
2. `5` - Authorization failed
3. `6` - Too many requests per second
4. `9` - Flood control
5. `10` - Internal server error
6. `14` - Captcha needed
7. `100` - Invalid parameter
8. `203` - Access to post denied
9. `214` - Posting to wall denied

### Retry Strategy
- For error 6 (rate limit): Wait 1 second and retry
- For error 9 (flood control): Wait longer (seconds in error message)
- For error 10 (server error): Exponential backoff
- Maximum 3 retries for transient errors

### Flood Control Handling
```go
func isFloodError(err error) (bool, time.Duration) {
	var apiErr *vk.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 9 {
			// Parse wait time from error message
			// Format: "Flood control: wait N seconds"
			re := regexp.MustCompile(`wait (\d+)`)
			matches := re.FindStringSubmatch(apiErr.Message)
			if len(matches) > 1 {
				seconds, _ := strconv.Atoi(matches[1])
				return true, time.Duration(seconds) * time.Second
			}
			return true, 5 * time.Second // Default wait
		}
	}
	return false, 0
}
```

## Rate Limiting
- Default: 3 requests per second (VK's limit)
- Implement token bucket algorithm
- Respect flood control errors with dynamic backoff

## Testing Strategy

### Unit Tests
- Mock VK API responses
- Test error handling and retries
- Test attachment formatting

### Integration Tests
- Use test access token with limited permissions
- Test with test group
- Verify post creation and media upload

### Test Data
Create test posts with:
1. Text only
2. Text with photo
3. Text with video
4. Text with document
5. Multiple attachments
6. Scheduled posts

## Configuration Example
```yaml
vk:
  access_token: "vk1.a.abcdef1234567890"
  api_url: "https://api.vk.com/method"
  api_version: "5.199"
  timeout: 60s
  language: "ru"
  requests_per_second: 3.0
  max_retries: 3
  group_id: 123456789
  from_group: true
```

## Dependencies
```go
import (
	vk "github.com/SevereCloud/vksdk/v2/api"
	"golang.org/x/time/rate"
	"go.uber.org/zap"
)
```

## Security Considerations
1. **Token Security**: Store access token in configuration file with restricted permissions
2. **Environment Variables**: Support env vars for CI/CD pipelines
3. **Token Rotation**: Implement token refresh if using user tokens (not needed for service tokens)
4. **Error Logging**: Don't log full access tokens in error messages

## Performance Optimizations
1. **Parallel Uploads**: Upload multiple media files concurrently (within rate limits)
2. **Batch Processing**: Group small posts if appropriate
3. **Connection Pooling**: Reuse HTTP connections
4. **Compression**: Enable gzip for API responses

## Next Steps
1. Implement basic client with authentication
2. Add wall.post functionality
3. Implement photo upload
4. Implement video and document upload
5. Add error handling and retries
6. Write comprehensive tests