package transfer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewCheckpointManager(t *testing.T) {
	logger := zap.NewNop()
	cm := NewCheckpointManager("/tmp/test.json", logger)
	if cm == nil {
		t.Fatal("expected checkpoint manager not to be nil")
	}
}

func TestCheckpointManager_Load_NoFile(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")
	cm := NewCheckpointManager(path, logger)
	state := cm.Load()
	if state == nil {
		t.Fatal("expected state not to be nil")
	}
	if state.LastProcessedIndex != -1 {
		t.Errorf("expected LastProcessedIndex -1, got %d", state.LastProcessedIndex)
	}
	if state.Metadata == nil {
		t.Error("expected Metadata map to be initialized")
	}
}

func TestCheckpointManager_Load_ValidFile(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "checkpoint.json")
	// Write a valid checkpoint file
	expectedState := CheckpointState{
		LastProcessedIndex: 42,
		LastProcessedID:    12345,
		LastProcessedTime:  time.Now().UTC(),
		TotalPosts:         100,
		Metadata:           map[string]interface{}{"key": "value"},
	}
	data, err := json.Marshal(expectedState)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cm := NewCheckpointManager(path, logger)
	state := cm.Load()
	if state == nil {
		t.Fatal("expected state not to be nil")
	}
	if state.LastProcessedIndex != expectedState.LastProcessedIndex {
		t.Errorf("expected LastProcessedIndex %d, got %d", expectedState.LastProcessedIndex, state.LastProcessedIndex)
	}
	if state.LastProcessedID != expectedState.LastProcessedID {
		t.Errorf("expected LastProcessedID %d, got %d", expectedState.LastProcessedID, state.LastProcessedID)
	}
	if state.TotalPosts != expectedState.TotalPosts {
		t.Errorf("expected TotalPosts %d, got %d", expectedState.TotalPosts, state.TotalPosts)
	}
	if val, ok := state.Metadata["key"]; !ok || val != "value" {
		t.Errorf("metadata mismatch")
	}
}

func TestCheckpointManager_Load_InvalidFile(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	// Write invalid JSON
	if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cm := NewCheckpointManager(path, logger)
	state := cm.Load()
	if state == nil {
		t.Fatal("expected state not to be nil")
	}
	if state.LastProcessedIndex != -1 {
		t.Errorf("expected LastProcessedIndex -1, got %d", state.LastProcessedIndex)
	}
}

func TestCheckpointManager_Save(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "checkpoint.json")
	cm := NewCheckpointManager(path, logger)
	state := &CheckpointState{
		LastProcessedIndex: 99,
		LastProcessedID:    999,
		TotalPosts:         500,
		Metadata:           map[string]interface{}{"test": true},
	}
	err := cm.Save(state)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	// Verify file exists and can be loaded
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	var loaded CheckpointState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal saved file: %v", err)
	}
	if loaded.LastProcessedIndex != state.LastProcessedIndex {
		t.Errorf("saved LastProcessedIndex mismatch: expected %d, got %d", state.LastProcessedIndex, loaded.LastProcessedIndex)
	}
	if loaded.LastProcessedID != state.LastProcessedID {
		t.Errorf("saved LastProcessedID mismatch: expected %d, got %d", state.LastProcessedID, loaded.LastProcessedID)
	}
	if loaded.TotalPosts != state.TotalPosts {
		t.Errorf("saved TotalPosts mismatch: expected %d, got %d", state.TotalPosts, loaded.TotalPosts)
	}
	// LastProcessedTime should be set by Save
	if loaded.LastProcessedTime.IsZero() {
		t.Error("LastProcessedTime should be set")
	}
}

func TestCheckpointManager_Save_CreateDirectory(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "nested", "checkpoint.json")
	cm := NewCheckpointManager(path, logger)
	state := &CheckpointState{
		LastProcessedIndex: 0,
	}
	err := cm.Save(state)
	if err != nil {
		t.Fatalf("failed to save with directory creation: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file not created")
	}
}

func TestCheckpointManager_LoadAfterSave(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "checkpoint.json")
	cm := NewCheckpointManager(path, logger)
	state := &CheckpointState{
		LastProcessedIndex: 77,
		LastProcessedID:    7777,
		TotalPosts:         200,
		Metadata:           map[string]interface{}{"foo": "bar"},
	}
	if err := cm.Save(state); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded := cm.Load()
	if loaded.LastProcessedIndex != state.LastProcessedIndex {
		t.Errorf("loaded index mismatch: expected %d, got %d", state.LastProcessedIndex, loaded.LastProcessedIndex)
	}
	if loaded.LastProcessedID != state.LastProcessedID {
		t.Errorf("loaded id mismatch: expected %d, got %d", state.LastProcessedID, loaded.LastProcessedID)
	}
	if loaded.TotalPosts != state.TotalPosts {
		t.Errorf("loaded total posts mismatch: expected %d, got %d", state.TotalPosts, loaded.TotalPosts)
	}
	if val, ok := loaded.Metadata["foo"]; !ok || val != "bar" {
		t.Errorf("metadata mismatch")
	}
}