package media

import (
	"time"
)

// MediaType represents supported media types
type MediaType string

const (
	TypePhoto    MediaType = "photo"
	TypeVideo    MediaType = "video"
	TypeDocument MediaType = "document"
	TypeAudio    MediaType = "audio"
	TypeVoice    MediaType = "voice"
	TypeSticker  MediaType = "sticker"
	TypeAnimation MediaType = "animation"
)

// MediaFile represents a media file on disk
type MediaFile struct {
	Path        string
	Type        MediaType
	Size        int64
	Width       int      // for images/videos
	Height      int      // for images/videos
	Duration    int      // for videos/audio in seconds
	MimeType    string
	OriginalURL string   // Source URL from Telegram
	Checksum    string   // MD5/SHA256 for deduplication
	CreatedAt   time.Time
}

// DownloadRequest represents a media download request
type DownloadRequest struct {
	FileID      string
	URL         string
	Type        MediaType
	FileName    string
	Caption     string
	MaxFileSize int64 // 0 = unlimited
}

// DownloadResult represents download outcome
type DownloadResult struct {
	File      *MediaFile
	Error     error
	Retries   int
	Duration  time.Duration
}

// ProcessingOptions defines media processing parameters
type ProcessingOptions struct {
	// Photo options
	MaxWidth      int
	MaxHeight     int
	Quality       int // 1-100
	ConvertToJPEG bool
	
	// Video options
	MaxVideoSize   int64 // in bytes
	MaxDuration    int   // in seconds
	VideoCodec     string
	AudioCodec     string
	ConvertGIFToMP4 bool
	
	// Document options
	MaxDocumentSize int64
	RenameDocuments bool
	
	// General
	KeepOriginal   bool
	OutputDir      string
	TempDir        string
}