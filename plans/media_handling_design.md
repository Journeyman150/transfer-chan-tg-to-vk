# Media Handling Design

## Overview
System for downloading media from Telegram, processing it, and uploading to VK with proper format conversion and error handling.

## Media Types Support

### Telegram → VK Mapping
| Telegram Type | VK Type | Notes |
|---------------|---------|-------|
| `photo` | `photo` | JPEG, PNG, GIF, WebP |
| `video` | `video` | MP4, AVI, MOV, etc. |
| `document` | `doc` | PDF, DOC, ZIP, etc. |
| `audio` | `audio` | MP3, OGG, WAV |
| `voice` | `audio` | OGG files (need conversion) |
| `animation` (GIF) | `video` or `doc` | Convert GIF to MP4 for video |
| `sticker` | `photo` or `doc` | Convert to PNG or keep as document |

## Architecture

### Media Pipeline
```
Telegram Media → Downloader → Processor → VK Uploader → Attachment
     ↓              ↓            ↓            ↓
   Temp Storage  Format Check  Conversion  Rate Limiting
```

### Components
1. **Downloader**: Downloads media from Telegram to local storage
2. **Storage Manager**: Manages temporary files and cleanup
3. **Processor**: Converts formats, resizes, optimizes
4. **Uploader**: Uploads processed media to VK
5. **Cache**: Optional caching to avoid re-downloading

## Data Structures

```go
package media

import (
	"time"
	"path/filepath"
)

// MediaType represents supported media types
type MediaType string

const (
	TypePhoto    MediaType = "photo"
	TypeVideo    MediaType = "video"
	TypeDocument MediaType = "document"
	TypeAudio    MediaType = "audio"
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
```

## Downloader Implementation

### Telegram Media Download
```go
package media

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Downloader struct {
	client      *http.Client
	tempDir     string
	maxRetries  int
	concurrency int
	logger      *zap.Logger
}

func NewDownloader(tempDir string, maxRetries, concurrency int) *Downloader {
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
	}
}

func (d *Downloader) Download(ctx context.Context, req DownloadRequest) (*MediaFile, error) {
	// Create temp file path
	ext := d.guessExtension(req.Type, req.FileName)
	tempFile := filepath.Join(d.tempDir, fmt.Sprintf("%s_%d%s", 
		req.FileID, time.Now().UnixNano(), ext))
	
	// Download with retry
	var lastErr error
	for i := 0; i < d.maxRetries; i++ {
		if i > 0 {
			// Exponential backoff
			delay := time.Duration(1<<uint(i)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		
		err := d.downloadFile(ctx, req.URL, tempFile, req.MaxFileSize)
		if err == nil {
			// Success
			return d.createMediaFile(tempFile, req)
		}
		
		lastErr = err
		d.logger.Warn("Download failed, retrying",
			zap.String("file_id", req.FileID),
			zap.Int("attempt", i+1),
			zap.Error(err))
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", d.maxRetries, lastErr)
}

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
```

## Storage Manager

### Temporary File Management
```go
package media

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

type StorageManager struct {
	baseDir    string
	maxAge     time.Duration
	maxSize    int64
	files      map[string]*MediaFile
	mu         sync.RWMutex
	cleanupTicker *time.Ticker
}

func NewStorageManager(baseDir string, maxAge time.Duration, maxSize int64) (*StorageManager, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	
	sm := &StorageManager{
		baseDir: baseDir,
		maxAge:  maxAge,
		maxSize: maxSize,
		files:   make(map[string]*MediaFile),
	}
	
	// Start cleanup goroutine
	sm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go sm.cleanupLoop()
	
	return sm, nil
}

func (sm *StorageManager) Store(file *MediaFile) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Check if file already exists (by checksum)
	for _, existing := range sm.files {
		if existing.Checksum == file.Checksum {
			// Return existing file
			*file = *existing
			return nil
		}
	}
	
	// Add to tracking
	sm.files[file.Path] = file
	
	// Check storage limits
	if sm.maxSize > 0 {
		if err := sm.enforceSizeLimit(); err != nil {
			return err
		}
	}
	
	return nil
}

func (sm *StorageManager) enforceSizeLimit() error {
	var totalSize int64
	for _, file := range sm.files {
		totalSize += file.Size
	}
	
	// Remove oldest files if over limit
	if totalSize > sm.maxSize {
		// Sort by creation time
		files := make([]*MediaFile, 0, len(sm.files))
		for _, file := range sm.files {
			files = append(files, file)
		}
		
		sort.Slice(files, func(i, j int) bool {
			return files[i].CreatedAt.Before(files[j].CreatedAt)
		})
		
		// Remove until under limit
		for totalSize > sm.maxSize && len(files) > 0 {
			file := files[0]
			files = files[1:]
			
			if err := sm.Remove(file.Path); err != nil {
				sm.logger.Error("Failed to remove file", zap.Error(err))
			} else {
				totalSize -= file.Size
			}
		}
	}
	
	return nil
}

func (sm *StorageManager) cleanupLoop() {
	for range sm.cleanupTicker.C {
		sm.cleanupOldFiles()
	}
}

func (sm *StorageManager) cleanupOldFiles() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	cutoff := time.Now().Add(-sm.maxAge)
	for path, file := range sm.files {
		if file.CreatedAt.Before(cutoff) {
			os.Remove(path)
			delete(sm.files, path)
		}
	}
}
```

## Media Processor

### Format Conversion and Optimization
```go
package media

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Processor struct {
	options ProcessingOptions
	logger  *zap.Logger
}

func NewProcessor(options ProcessingOptions) *Processor {
	return &Processor{
		options: options,
		logger:  zap.L().Named("processor"),
	}
}

func (p *Processor) Process(ctx context.Context, file *MediaFile) (*MediaFile, error) {
	switch file.Type {
	case TypePhoto:
		return p.processPhoto(ctx, file)
	case TypeVideo:
		return p.processVideo(ctx, file)
	case TypeDocument:
		return p.processDocument(ctx, file)
	case TypeAudio:
		return p.processAudio(ctx, file)
	default:
		return file, nil // No processing for unknown types
	}
}

func (p *Processor) processPhoto(ctx context.Context, file *MediaFile) (*MediaFile, error) {
	// Check if processing is needed
	needsResize := p.options.MaxWidth > 0 && file.Width > p.options.MaxWidth ||
		p.options.MaxHeight > 0 && file.Height > p.options.MaxHeight
	
	needsConversion := p.options.ConvertToJPEG && 
		!strings.HasSuffix(strings.ToLower(file.Path), ".jpg") &&
		!strings.HasSuffix(strings.ToLower(file.Path), ".jpeg")
	
	if !needsResize && !needsConversion {
		return file, nil
	}
	
	// Use ImageMagick or similar for processing
	outputPath := filepath.Join(p.options.OutputDir, 
		filepath.Base(file.Path)+".processed.jpg")
	
	args := []string{file.Path, "-auto-orient"}
	
	if needsResize {
		args = append(args, "-resize", 
			fmt.Sprintf("%dx%d>", p.options.MaxWidth, p.options.MaxHeight))
	}
	
	if p.options.Quality > 0 {
		args = append(args, "-quality", fmt.Sprintf("%d", p.options.Quality))
	}
	
	args = append(args, outputPath)
	
	cmd := exec.CommandContext(ctx, "convert", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("image processing failed: %w", err)
	}
	
	// Create new MediaFile for processed image
	processedFile := &MediaFile{
		Path:        outputPath,
		Type:        TypePhoto,
		MimeType:    "image/jpeg",
		OriginalURL: file.OriginalURL,
		CreatedAt:   time.Now(),
	}
	
	// Get file info
	if info, err := os.Stat(outputPath); err == nil {
		processedFile.Size = info.Size()
	}
	
	return processedFile, nil
}

func (p *Processor) processVideo(ctx context.Context, file *MediaFile) (*MediaFile, error) {
	// Check if GIF needs conversion to MP4
	if p.options.ConvertGIFToMP4 && strings.HasSuffix(strings.ToLower(file.Path), ".gif") {
		return p.convertGIFToMP4(ctx, file)
	}
	
	// Check size/duration limits
	if p.options.MaxVideoSize > 0 && file.Size > p.options.MaxVideoSize {
		return nil, fmt.Errorf("video too large: %d > %d", 
			file.Size, p.options.MaxVideoSize)
	}
	
	if p.options.MaxDuration > 0 && file.Duration > p.options.MaxDuration {
		return nil, fmt.Errorf("video too long: %d > %d seconds", 
			file.Duration, p.options.MaxDuration)
	}
	
	return file, nil
}

func (p *Processor) convertGIFToMP4(ctx context.Context, file *MediaFile) (*MediaFile, error) {
	outputPath := strings.TrimSuffix(file.Path, filepath.Ext(file.Path)) + ".mp4"
	
	// Use ffmpeg to convert GIF to MP4
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", file.Path,
		"-movflags", "faststart",
		"-pix_fmt", "yuv420p",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		outputPath)
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("GIF to MP4 conversion failed: %w", err)
	}
	
	processedFile := &MediaFile{
		Path:        outputPath,
		Type:        TypeVideo,
		MimeType:    "video/mp4",
		OriginalURL: file.OriginalURL,
		CreatedAt:   time.Now(),
	}
	
	if info, err := os.Stat(outputPath); err == nil {
		processedFile.Size = info.Size()
	}
	
	return processedFile, nil
}
```

## Upload Coordinator

### Managing Uploads to VK
```go
package media

import (
	"context"
	"sync"
	"time"
)

type UploadCoordinator struct {
	vkClient    vk.Client
	downloader  *Downloader
	processor   *Processor
	storage     *StorageManager
	maxParallel int
	logger      *zap.Logger
}

func NewUploadCoordinator(vkClient vk.Client, downloader *Downloader, 
	processor *Processor, storage *StorageManager, maxParallel int) *UploadCoordinator {
	return &UploadCoordinator{
		vkClient:    vkClient,
		downloader:  downloader,
		processor:   processor,
		storage:     storage,
		maxParallel: maxParallel,
		logger:      zap.L().Named("upload-coordinator"),
	}
}

func (uc *UploadCoordinator) UploadMedia(ctx context.Context, 
	telegramMedia []telegram.Media, groupID int) ([]vk.Attachment, error) {
	
	// Create semaphore for limiting concurrent uploads
	sem := make(chan struct{}, uc.maxParallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var attachments []vk.Attachment
	var errors []error
	
	for _, media := range telegramMedia {
		wg.Add(1)
		
		go func(m telegram.Media) {
			defer wg.Done()
			
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			
			// Process single media item
			att, err := uc.processSingleMedia(ctx, m, groupID)
			
			mu.Lock()
			if err != nil {
				errors = append(errors, err)
				uc.logger.Error("Failed to process media",
					zap.String("type", string(m.Type)),
					zap.Error(err))
			} else if att != nil {
				attachments = append(attachments, *att)
			}
			mu.Unlock()
		}(media)
	}
	
	wg.Wait()
	
	// Return error if all failed
	if len(attachments) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all media uploads failed: %v", errors)
	}
	
	return attachments, nil
}

func (uc *UploadCoordinator) processSingleMedia(ctx context.Context, 
	m telegram.Media, groupID int) (*vk.Attachment, error) {
	
	// 1. Download from Telegram
	req := DownloadRequest{
		FileID:   m.FileID,
		URL:      m.URL,
		Type:     MediaType(m.Type),
		FileName: m.FileName,
		Caption:  m.Caption,
	}
	
	file, err := uc.downloader.Download(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	
	// 2. Store in temporary storage
	if err := uc.storage.Store(file); err != nil {
		return nil, fmt.Errorf("storage failed: %w", err)
	}
	
	// 3. Process if needed
	processedFile, err := uc.processor.Process(ctx, file)
	if err != nil {
		uc.logger.Warn("Processing failed, using original",
			zap.String("file", file.Path),
			zap.Error(err))
		processedFile = file
	}
	
	// 4. Upload to VK
	var attachment *vk.Attachment
	switch processedFile.Type {
	case TypePhoto:
		attachment, err = uc.vkClient.UploadPhoto(ctx, processedFile.Path, groupID)
	case TypeVideo:
		attachment, err = uc.vkClient.UploadVideo(ctx, processedFile.Path, 
			m.Caption, "", groupID)
	case TypeDocument, TypeAudio:
		attachment, err = uc.vkClient.UploadDocument(ctx, processedFile.Path, 
			m.Caption, groupID)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", processedFile.Type)
	}
	
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	
	return attachment, nil
}
```

## Error Handling

### Recovery Strategies
1. **Partial Failure**: Continue with remaining media if one fails
2. **Checkpointing**: Save progress to resume later
3. **Fallback Options**: 
   - Skip media if upload fails
   - Convert to link if file too large
   - Use lower quality if conversion fails

### Error Types
1. **Download Errors**: Network issues, file not found
2. **Processing Errors**: Conversion failures, unsupported formats
3. **Upload Errors**: VK API limits, file size limits
4. **Storage Errors**: Disk full, permission denied

## Configuration

```yaml
media:
  # Download settings
  temp_dir: "./temp/media"
  max_concurrent_downloads: 3
  download_timeout: 30s
  max_retries: 3
  
  # Storage settings
  max_file_age: 24h
  max_storage_size: 10GB
  cleanup_interval: 5m
  
  # Processing settings
  photos:
    max_width: 1920
    max_height: 1080
    quality: 85
    convert_to_jpeg: true
    
  videos:
    max_size: 2GB
    max_duration: 600 # 10 minutes
    convert_gif_to_mp4: true
    
  documents:
    max_size: 2GB
    rename: true
    
  # Upload settings
  max_concurrent_uploads: 2
  upload_timeout: 300s # 5 minutes for large files
```

## Dependencies
- **Image Processing**: ImageMagick (`convert`), `github.com/disintegration/imaging`
- **Video Processing**: FFmpeg (`ffmpeg`)
- **HTTP Client**: Standard library with custom transport
- **Concurrency**: `sync` package with worker pools

## Testing Strategy
1. **Unit Tests**: Mock HTTP responses, test individual components
2. **Integration Tests**: Test with small sample files
3. **Performance Tests**: Measure download/upload speeds
4. **Error Simulation**: Test recovery from various failures

## Next Steps
1. Implement basic downloader with retry logic
2. Add storage manager with cleanup
3. Implement simple processor (resize images)
4. Integrate with VK upload client
5. Add comprehensive error handling