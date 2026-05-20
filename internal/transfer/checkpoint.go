package transfer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// CheckpointState represents the state of a transfer that can be resumed.
type CheckpointState struct {
	// Last processed post index (0-based)
	LastProcessedIndex int `json:"last_processed_index"`
	// Last processed post ID
	LastProcessedID int64 `json:"last_processed_id"`
	// Timestamp of last checkpoint update
	LastProcessedTime time.Time `json:"last_processed_time"`
	// Total posts fetched in last run
	TotalPosts int `json:"total_posts"`
	// Any additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CheckpointManager handles saving and loading checkpoint state.
type CheckpointManager struct {
	filePath string
	logger   *zap.Logger
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(filePath string, logger *zap.Logger) *CheckpointManager {
	return &CheckpointManager{
		filePath: filePath,
		logger:   logger,
	}
}

// Load loads the checkpoint state from disk.
// Returns empty state if file doesn't exist or is invalid.
func (cm *CheckpointManager) Load() *CheckpointState {
	cm.logger.Debug("Loading checkpoint", zap.String("path", cm.filePath))

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.logger.Debug("Checkpoint file does not exist, starting fresh")
			return &CheckpointState{
				LastProcessedIndex: -1,
				Metadata:           make(map[string]interface{}),
			}
		}
		cm.logger.Warn("Failed to read checkpoint file", zap.Error(err))
		return &CheckpointState{
			LastProcessedIndex: -1,
			Metadata:           make(map[string]interface{}),
		}
	}

	var state CheckpointState
	if err := json.Unmarshal(data, &state); err != nil {
		cm.logger.Warn("Failed to parse checkpoint file", zap.Error(err))
		return &CheckpointState{
			LastProcessedIndex: -1,
			Metadata:           make(map[string]interface{}),
		}
	}

	cm.logger.Info("Checkpoint loaded",
		zap.Int("last_processed_index", state.LastProcessedIndex),
		zap.Int64("last_processed_id", state.LastProcessedID),
		zap.Time("last_processed_time", state.LastProcessedTime),
	)
	return &state
}

// Save saves the checkpoint state to disk.
func (cm *CheckpointManager) Save(state *CheckpointState) error {
	cm.logger.Debug("Saving checkpoint",
		zap.Int("last_processed_index", state.LastProcessedIndex),
		zap.Int64("last_processed_id", state.LastProcessedID),
	)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cm.filePath), 0755); err != nil {
		return fmt.Errorf("create checkpoint directory: %w", err)
	}

	state.LastProcessedTime = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	// Write to temporary file first, then rename (atomic write)
	tmpPath := cm.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write checkpoint temp file: %w", err)
	}

	if err := os.Rename(tmpPath, cm.filePath); err != nil {
		return fmt.Errorf("rename checkpoint file: %w", err)
	}

	return nil
}

// Clear removes the checkpoint file.
func (cm *CheckpointManager) Clear() error {
	cm.logger.Info("Clearing checkpoint")
	if err := os.Remove(cm.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove checkpoint file: %w", err)
	}
	return nil
}