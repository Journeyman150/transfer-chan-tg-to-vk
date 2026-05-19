package telegram

import "time"

// MediaType represents different media types
type MediaType string

const (
	MediaTypePhoto     MediaType = "photo"
	MediaTypeVideo     MediaType = "video"
	MediaTypeDocument  MediaType = "document"
	MediaTypeAudio     MediaType = "audio"
	MediaTypeVoice     MediaType = "voice"
	MediaTypeSticker   MediaType = "sticker"
	MediaTypeAnimation MediaType = "animation" // GIF
)

// Media represents a media attachment
type Media struct {
	Type        MediaType
	FileID      string
	FileUniqueID string
	FileSize    int64
	Width       int    // for photos/videos
	Height      int    // for photos/videos
	Duration    int    // for videos/audio in seconds
	FileName    string // for documents
	MimeType    string // for documents
	Caption     string
}

// Post represents a Telegram channel post
type Post struct {
	ID            int64
	Date          time.Time
	EditDate      *time.Time
	Text          string
	Media         []Media
	ForwardedFrom *ForwardInfo
	Views         int
	Reactions     []Reaction
	Link          string // t.me/channel/123
	HasSpoiler    bool
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

// GetPostsOptions defines parameters for fetching posts
type GetPostsOptions struct {
	Limit      int       // Maximum number of posts to fetch (0 = default)
	OffsetID   int64     // ID of the message to start from (0 = most recent)
	OffsetDate time.Time // Date to start from
	StartDate  time.Time // Only fetch posts after this date
	EndDate    time.Time // Only fetch posts before this date
	OnlyMedia  bool      // Only fetch posts with media
}