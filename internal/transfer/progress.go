package transfer

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Progress represents the current progress of a transfer.
type Progress struct {
	TotalPosts     int       `json:"total_posts"`
	ProcessedPosts int       `json:"processed_posts"`
	FailedPosts    int       `json:"failed_posts"`
	SkippedPosts   int       `json:"skipped_posts"`
	StartTime      time.Time `json:"start_time"`
	LastUpdateTime time.Time `json:"last_update_time"`
	CurrentPostID  int64     `json:"current_post_id"`
	CurrentStatus  string    `json:"current_status"`
	ETA            time.Time `json:"eta,omitempty"`
}

// ProgressTracker tracks and reports transfer progress.
type ProgressTracker struct {
	progress Progress
	mu       sync.RWMutex
	logger   *zap.Logger
	started  bool
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(logger *zap.Logger) *ProgressTracker {
	return &ProgressTracker{
		progress: Progress{
			StartTime: time.Now(),
		},
		logger: logger,
	}
}

// Start initializes the progress tracker with total posts.
func (pt *ProgressTracker) Start(totalPosts int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.progress = Progress{
		TotalPosts:     totalPosts,
		ProcessedPosts: 0,
		FailedPosts:    0,
		SkippedPosts:   0,
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
		CurrentStatus:  "starting",
	}
	pt.started = true

	pt.logger.Info("Progress tracking started",
		zap.Int("total_posts", totalPosts),
	)
}

// UpdatePostProcessed increments the processed posts count.
func (pt *ProgressTracker) UpdatePostProcessed(postID int64, status string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.started {
		return
	}

	pt.progress.ProcessedPosts++
	pt.progress.CurrentPostID = postID
	pt.progress.CurrentStatus = status
	pt.progress.LastUpdateTime = time.Now()

	// Calculate ETA if we have processed at least one post
	if pt.progress.ProcessedPosts > 0 {
		elapsed := time.Since(pt.progress.StartTime)
		postsPerSecond := float64(pt.progress.ProcessedPosts) / elapsed.Seconds()
		if postsPerSecond > 0 {
			remainingPosts := pt.progress.TotalPosts - pt.progress.ProcessedPosts
			etaSeconds := float64(remainingPosts) / postsPerSecond
			pt.progress.ETA = time.Now().Add(time.Duration(etaSeconds) * time.Second)
		}
	}

	// Log progress every 10 posts or when percentage changes significantly
	if pt.progress.ProcessedPosts%10 == 0 || pt.progress.ProcessedPosts == pt.progress.TotalPosts {
		pt.logProgress()
	}
}

// UpdatePostFailed increments the failed posts count.
func (pt *ProgressTracker) UpdatePostFailed(postID int64, reason string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.started {
		return
	}

	pt.progress.FailedPosts++
	pt.progress.LastUpdateTime = time.Now()

	pt.logger.Warn("Post processing failed",
		zap.Int64("post_id", postID),
		zap.String("reason", reason),
		zap.Int("failed_count", pt.progress.FailedPosts),
	)
}

// UpdatePostSkipped increments the skipped posts count.
func (pt *ProgressTracker) UpdatePostSkipped(postID int64, reason string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.started {
		return
	}

	pt.progress.SkippedPosts++
	pt.progress.LastUpdateTime = time.Now()

	pt.logger.Info("Post skipped",
		zap.Int64("post_id", postID),
		zap.String("reason", reason),
		zap.Int("skipped_count", pt.progress.SkippedPosts),
	)
}

// UpdateStatus updates the current status message.
func (pt *ProgressTracker) UpdateStatus(status string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.progress.CurrentStatus = status
	pt.progress.LastUpdateTime = time.Now()

	pt.logger.Debug("Status updated", zap.String("status", status))
}

// GetProgress returns the current progress.
func (pt *ProgressTracker) GetProgress() Progress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.progress
}

// GetPercentage returns the completion percentage (0-100).
func (pt *ProgressTracker) GetPercentage() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.progress.TotalPosts == 0 {
		return 0
	}
	return float64(pt.progress.ProcessedPosts) / float64(pt.progress.TotalPosts) * 100
}

// GetETA returns the estimated time of completion.
func (pt *ProgressTracker) GetETA() time.Time {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.progress.ETA
}

// GetRemainingTime returns the estimated remaining time.
func (pt *ProgressTracker) GetRemainingTime() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.progress.ETA.IsZero() {
		return 0
	}
	return time.Until(pt.progress.ETA)
}

// logProgress logs the current progress.
func (pt *ProgressTracker) logProgress() {
	percentage := pt.GetPercentage()
	remaining := pt.GetRemainingTime()

	pt.logger.Info("Transfer progress",
		zap.Int("processed", pt.progress.ProcessedPosts),
		zap.Int("total", pt.progress.TotalPosts),
		zap.Int("failed", pt.progress.FailedPosts),
		zap.Int("skipped", pt.progress.SkippedPosts),
		zap.Float64("percentage", percentage),
		zap.String("remaining", formatDuration(remaining)),
		zap.String("status", pt.progress.CurrentStatus),
	)
}

// formatDuration formats a duration for human readability.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}