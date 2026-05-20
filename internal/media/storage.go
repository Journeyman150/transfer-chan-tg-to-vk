package media

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// StorageManager manages temporary files and cleanup
type StorageManager struct {
	baseDir       string
	maxAge        time.Duration
	maxSize       int64
	files         map[string]*MediaFile
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	logger        *zap.Logger
}

// NewStorageManager creates a new storage manager
func NewStorageManager(baseDir string, maxAge time.Duration, maxSize int64) (*StorageManager, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	
	logger := zap.L().Named("storage")
	
	sm := &StorageManager{
		baseDir: baseDir,
		maxAge:  maxAge,
		maxSize: maxSize,
		files:   make(map[string]*MediaFile),
		logger:  logger,
	}
	
	// Start cleanup goroutine
	sm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go sm.cleanupLoop()
	
	return sm, nil
}

// Store adds a media file to storage tracking
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

// Get returns a media file by path
func (sm *StorageManager) Get(path string) (*MediaFile, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	file, ok := sm.files[path]
	return file, ok
}

// Remove removes a file from storage and deletes it from disk
func (sm *StorageManager) Remove(path string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Delete from disk
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	
	// Remove from tracking
	delete(sm.files, path)
	
	return nil
}

// Cleanup removes all files older than maxAge
func (sm *StorageManager) Cleanup() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	cutoff := time.Now().Add(-sm.maxAge)
	for path, file := range sm.files {
		if file.CreatedAt.Before(cutoff) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				sm.logger.Error("Failed to remove old file", zap.String("path", path), zap.Error(err))
			} else {
				delete(sm.files, path)
			}
		}
	}
	
	return nil
}

// Close stops the cleanup ticker
func (sm *StorageManager) Close() {
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
}

// enforceSizeLimit removes oldest files if total size exceeds maxSize
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

// cleanupLoop runs periodic cleanup
func (sm *StorageManager) cleanupLoop() {
	for range sm.cleanupTicker.C {
		sm.Cleanup()
	}
}

// GetStats returns storage statistics
func (sm *StorageManager) GetStats() (fileCount int, totalSize int64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	fileCount = len(sm.files)
	for _, file := range sm.files {
		totalSize += file.Size
	}
	return
}