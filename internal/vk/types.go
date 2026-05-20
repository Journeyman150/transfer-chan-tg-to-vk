package vk

import (
	"time"
)

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
	Type      AttachmentType
	OwnerID   int // Negative for groups
	MediaID   int
	AccessKey string // For private content
}

// Post represents a VK wall post
type Post struct {
	Text        string
	Attachments []Attachment
	PublishDate *time.Time
	FromGroup   bool   // true for group posts, false for user
	Signed      bool   // Show author signature (for groups)
	FriendsOnly bool   // Visible to friends only
	Services    string // "twitter", "facebook" for cross-posting
	Lat         float64
	Long        float64
	PlaceID     int
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
	PostID   int    `json:"post_id"`
	PostHash string `json:"post_hash"`
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