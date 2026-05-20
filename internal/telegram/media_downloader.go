package telegram

import (
	"context"
	"fmt"
	"time"

	"ai-transfer-tg-to-vk/internal/media"
	"go.uber.org/zap"
)

// MediaDownloader provides concurrent downloading of Telegram media with temporary storage
type MediaDownloader struct {
	client      Client
	downloader  *media.Downloader
	storage     *media.StorageManager
	logger      *zap.Logger
	tempDir     string
}

// NewMediaDownloader creates a new media downloader
func NewMediaDownloader(client Client, tempDir string, maxRetries, concurrency int, maxAge time.Duration, maxSize int64) (*MediaDownloader, error) {
	// Create storage manager
	storage, err := media.NewStorageManager(tempDir, maxAge, maxSize)
	if err != nil {
		return nil, fmt.Errorf("create storage manager: %w", err)
	}

	// Create downloader
	downloader := media.NewDownloader(tempDir, maxRetries, concurrency, storage)

	logger := zap.L().Named("telegram.media_downloader")

	return &MediaDownloader{
		client:     client,
		downloader: downloader,
		storage:    storage,
		logger:     logger,
		tempDir:    tempDir,
	}, nil
}

// DownloadPostMedia downloads all media from a post concurrently
func (md *MediaDownloader) DownloadPostMedia(ctx context.Context, post *Post) ([]*media.MediaFile, error) {
	if len(post.Media) == 0 {
		return nil, nil
	}

	// Convert Telegram media to download requests
	requests := make([]media.DownloadRequest, 0, len(post.Media))
	for _, m := range post.Media {
		req, err := md.createDownloadRequest(ctx, m)
		if err != nil {
			md.logger.Warn("Failed to create download request", zap.String("file_id", m.FileID), zap.Error(err))
			continue
		}
		requests = append(requests, req)
	}

	if len(requests) == 0 {
		return nil, nil
	}

	// Download concurrently
	results := md.downloader.DownloadMany(ctx, requests)

	// Collect successful downloads
	files := make([]*media.MediaFile, 0, len(results))
	for _, result := range results {
		if result.Error != nil {
			md.logger.Error("Failed to download media", zap.Error(result.Error))
			continue
		}
		files = append(files, result.File)
	}

	return files, nil
}

// DownloadMedia downloads a single media file
func (md *MediaDownloader) DownloadMedia(ctx context.Context, mediaObj Media) (*media.MediaFile, error) {
	req, err := md.createDownloadRequest(ctx, mediaObj)
	if err != nil {
		return nil, err
	}

	file, err := md.downloader.Download(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}

	return file, nil
}

// createDownloadRequest creates a DownloadRequest from Telegram Media
func (md *MediaDownloader) createDownloadRequest(ctx context.Context, m Media) (media.DownloadRequest, error) {
	// Get download URL from Telegram client
	url, err := md.client.GetMediaURL(ctx, m.FileID)
	if err != nil {
		return media.DownloadRequest{}, fmt.Errorf("get media URL: %w", err)
	}

	// Map MediaType to media.MediaType
	mediaType := md.mapMediaType(m.Type)

	return media.DownloadRequest{
		FileID:      m.FileID,
		URL:         url,
		Type:        mediaType,
		FileName:    m.FileName,
		Caption:     m.Caption,
		MaxFileSize: m.FileSize,
	}, nil
}

// mapMediaType converts Telegram MediaType to media.MediaType
func (md *MediaDownloader) mapMediaType(t MediaType) media.MediaType {
	switch t {
	case MediaTypePhoto:
		return media.TypePhoto
	case MediaTypeVideo:
		return media.TypeVideo
	case MediaTypeDocument:
		return media.TypeDocument
	case MediaTypeAudio:
		return media.TypeAudio
	case MediaTypeVoice:
		return media.TypeVoice
	case MediaTypeSticker:
		return media.TypeSticker
	case MediaTypeAnimation:
		return media.TypeAnimation
	default:
		return media.TypeDocument
	}
}

// Cleanup removes temporary files older than maxAge
func (md *MediaDownloader) Cleanup() error {
	return md.storage.Cleanup()
}

// GetStats returns storage statistics
func (md *MediaDownloader) GetStats() (fileCount int, totalSize int64) {
	return md.storage.GetStats()
}

// Close releases resources
func (md *MediaDownloader) Close() {
	md.storage.Close()
}