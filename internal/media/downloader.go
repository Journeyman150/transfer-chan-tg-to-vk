package media

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Downloader downloads media files with concurrency and retries
type Downloader struct {
	client      *http.Client
	tempDir     string
	maxRetries  int
	concurrency int
	logger      *zap.Logger
	storage     *StorageManager
	sem         chan struct{} // semaphore for concurrency control
}

// NewDownloader creates a new downloader
func NewDownloader(tempDir string, maxRetries, concurrency int, storage *StorageManager) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
				MaxConnsPerHost:     5,
				MaxIdleConnsPerHost: 5,
			},
		},
		tempDir:     tempDir,
		maxRetries:  maxRetries,
		concurrency: concurrency,
		logger:      zap.L().Named("downloader"),
		storage:     storage,
		sem:         make(chan struct{}, concurrency),
	}
}

// Download downloads a single file
func (d *Downloader) Download(ctx context.Context, req DownloadRequest) (*MediaFile, error) {
	// Create temp file path
	ext := d.guessExtension(req.Type, req.FileName)
	tempFile := filepath.Join(d.tempDir, fmt.Sprintf("%s_%d%s", 
		req.FileID, time.Now().UnixNano(), ext))
	
	// Download with retry
	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			d.logger.Warn("Retrying download",
				zap.String("file_id", req.FileID),
				zap.Int("attempt", attempt))
		}

		err := d.downloadFile(ctx, req.URL, tempFile, req.MaxFileSize)
		if err == nil {
			// Success
			return d.createMediaFile(tempFile, req)
		}
		lastErr = err
		d.logger.Warn("Download failed",
			zap.String("file_id", req.FileID),
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	return nil, fmt.Errorf("failed after %d retries: %w", d.maxRetries, lastErr)
}

// DownloadMany downloads multiple files concurrently
func (d *Downloader) DownloadMany(ctx context.Context, requests []DownloadRequest) []DownloadResult {
	results := make([]DownloadResult, len(requests))
	var wg sync.WaitGroup
	wg.Add(len(requests))

	for i, req := range requests {
		go func(idx int, req DownloadRequest) {
			defer wg.Done()
			
			// Acquire semaphore
			select {
			case d.sem <- struct{}{}:
				defer func() { <-d.sem }()
			case <-ctx.Done():
				results[idx] = DownloadResult{
					Error: ctx.Err(),
				}
				return
			}

			start := time.Now()
			file, err := d.Download(ctx, req)
			duration := time.Since(start)

			results[idx] = DownloadResult{
				File:     file,
				Error:    err,
				Retries:  0, // TODO: track retries
				Duration: duration,
			}
		}(i, req)
	}

	wg.Wait()
	return results
}

// downloadFile performs a single HTTP GET request to download a file
func (d *Downloader) downloadFile(ctx context.Context, url, destPath string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()
	
	// Copy with size limit
	var reader io.Reader = resp.Body
	if maxSize > 0 {
		reader = io.LimitReader(resp.Body, maxSize)
	}
	
	_, err = io.Copy(file, reader)
	if err != nil {
		// Clean up partial file
		os.Remove(destPath)
		return fmt.Errorf("copy data: %w", err)
	}
	
	return nil
}

// guessExtension determines file extension based on type and filename
func (d *Downloader) guessExtension(mediaType MediaType, filename string) string {
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			return ext
		}
	}
	
	// Default extensions by type
	switch mediaType {
	case TypePhoto:
		return ".jpg"
	case TypeVideo:
		return ".mp4"
	case TypeAudio, TypeVoice:
		return ".mp3"
	case TypeDocument:
		return ".bin"
	case TypeSticker:
		return ".webp"
	case TypeAnimation:
		return ".gif"
	default:
		return ".dat"
	}
}

// createMediaFile creates a MediaFile from downloaded file
func (d *Downloader) createMediaFile(path string, req DownloadRequest) (*MediaFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	
	// Compute checksum
	checksum, err := d.computeChecksum(path)
	if err != nil {
		d.logger.Warn("Failed to compute checksum", zap.String("path", path), zap.Error(err))
	}
	
	mediaFile := &MediaFile{
		Path:        path,
		Type:        req.Type,
		Size:        info.Size(),
		MimeType:    "", // TODO: detect mime type
		OriginalURL: req.URL,
		Checksum:    checksum,
		CreatedAt:   time.Now(),
	}
	
	// Store in storage manager if available
	if d.storage != nil {
		if err := d.storage.Store(mediaFile); err != nil {
			d.logger.Error("Failed to store media file", zap.Error(err))
		}
	}
	
	return mediaFile, nil
}

// computeChecksum computes MD5 checksum of file
func (d *Downloader) computeChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}