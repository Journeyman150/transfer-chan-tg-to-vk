package transfer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
	"ai-transfer-tg-to-vk/internal/config"
	"ai-transfer-tg-to-vk/internal/media"
	"ai-transfer-tg-to-vk/internal/transformer"
	"go.uber.org/zap/zaptest"
)

// mockTelegramClient implements tg.Client for testing
type mockTelegramClient struct {
	posts      []tg.Post
	shouldFail bool
	failOnCall map[string]bool // map of method name to fail flag
}

func (m *mockTelegramClient) GetChannelInfo(ctx context.Context, channelID string) (*tg.ChannelInfo, error) {
	if m.shouldFail || m.failOnCall["GetChannelInfo"] {
		return nil, errors.New("mock: GetChannelInfo failed")
	}
	return &tg.ChannelInfo{
		ID:       123,
		Username: "test_channel",
		Title:    "Test Channel",
		IsPublic: true,
	}, nil
}

func (m *mockTelegramClient) GetPosts(ctx context.Context, channelID string, opts tg.GetPostsOptions) ([]tg.Post, error) {
	if m.shouldFail || m.failOnCall["GetPosts"] {
		return nil, errors.New("mock: GetPosts failed")
	}
	return m.posts, nil
}

func (m *mockTelegramClient) GetPostByID(ctx context.Context, channelID string, postID int64) (*tg.Post, error) {
	if m.shouldFail || m.failOnCall["GetPostByID"] {
		return nil, errors.New("mock: GetPostByID failed")
	}
	for _, post := range m.posts {
		if post.ID == postID {
			return &post, nil
		}
	}
	return nil, fmt.Errorf("post %d not found", postID)
}

func (m *mockTelegramClient) GetMediaURL(ctx context.Context, fileID string) (string, error) {
	if m.shouldFail || m.failOnCall["GetMediaURL"] {
		return "", errors.New("mock: GetMediaURL failed")
	}
	return fmt.Sprintf("https://example.com/file/%s", fileID), nil
}

func (m *mockTelegramClient) DownloadMedia(ctx context.Context, fileID, destPath string) error {
	if m.shouldFail || m.failOnCall["DownloadMedia"] {
		return errors.New("mock: DownloadMedia failed")
	}
	// Create a dummy file
	return os.WriteFile(destPath, []byte("mock media content"), 0644)
}

func (m *mockTelegramClient) Close() error {
	return nil
}

// mockVKClient implements vk.Client for testing
type mockVKClient struct {
	postedPosts []vk.Post
	shouldFail  bool
	failOnCall  map[string]bool
	uploaded    []string // paths of uploaded files
}

func (m *mockVKClient) Post(ctx context.Context, post vk.Post) (*vk.PostResult, error) {
	if m.shouldFail || m.failOnCall["Post"] {
		return nil, errors.New("mock: Post failed")
	}
	m.postedPosts = append(m.postedPosts, post)
	return &vk.PostResult{
		PostID:   len(m.postedPosts),
		PostHash: fmt.Sprintf("hash_%d", len(m.postedPosts)),
	}, nil
}

func (m *mockVKClient) UploadPhoto(ctx context.Context, filePath string, groupID int) (*vk.Attachment, error) {
	if m.shouldFail || m.failOnCall["UploadPhoto"] {
		return nil, errors.New("mock: UploadPhoto failed")
	}
	m.uploaded = append(m.uploaded, filePath)
	return &vk.Attachment{
		Type:      vk.AttachmentTypePhoto,
		OwnerID:   -groupID, // negative for groups
		MediaID:   len(m.uploaded),
		AccessKey: fmt.Sprintf("key_%d", len(m.uploaded)),
	}, nil
}

func (m *mockVKClient) UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*vk.Attachment, error) {
	if m.shouldFail || m.failOnCall["UploadVideo"] {
		return nil, errors.New("mock: UploadVideo failed")
	}
	m.uploaded = append(m.uploaded, filePath)
	return &vk.Attachment{
		Type:      vk.AttachmentTypeVideo,
		OwnerID:   -groupID,
		MediaID:   len(m.uploaded),
		AccessKey: fmt.Sprintf("key_%d", len(m.uploaded)),
	}, nil
}

func (m *mockVKClient) UploadDocument(ctx context.Context, filePath, title string, groupID int) (*vk.Attachment, error) {
	if m.shouldFail || m.failOnCall["UploadDocument"] {
		return nil, errors.New("mock: UploadDocument failed")
	}
	m.uploaded = append(m.uploaded, filePath)
	return &vk.Attachment{
		Type:      vk.AttachmentTypeDoc,
		OwnerID:   -groupID,
		MediaID:   len(m.uploaded),
		AccessKey: fmt.Sprintf("key_%d", len(m.uploaded)),
	}, nil
}

func (m *mockVKClient) GetGroupInfo(ctx context.Context, groupID int) (*vk.GroupInfo, error) {
	if m.shouldFail || m.failOnCall["GetGroupInfo"] {
		return nil, errors.New("mock: GetGroupInfo failed")
	}
	return &vk.GroupInfo{
		ID:          groupID,
		Name:        "Test Group",
		ScreenName:  "test_group",
		Description: "Test group description",
		Members:     100,
		PhotoURL:    "https://example.com/photo.jpg",
		Type:        "group",
	}, nil
}

func (m *mockVKClient) Close() error {
	return nil
}

// createTestConfig creates a minimal config for testing
func createTestConfig(t *testing.T) *config.Config {
	tempDir := t.TempDir()
	
	return &config.Config{
		Telegram: config.TelegramConfig{
			BotToken:         "test_token",
			ChannelID:        "@test_channel",
			APIURL:           "https://api.telegram.org",
			Timeout:          config.Duration(30 * time.Second),
			Debug:            false,
			RequestsPerSecond: 1,
			MaxRetries:       3,
			BatchSize:        10,
			MaxFileSize:      10 * 1024 * 1024, // 10MB
		},
		VK: config.VKConfig{
			AccessToken:      "test_vk_token",
			GroupID:          123456,
			APIURL:           "https://api.vk.com",
			APIVersion:       "5.199",
			Timeout:          config.Duration(30 * time.Second),
			RequestsPerSecond: 3.0,
			FromGroup:        true,
			PostDelay:        config.Duration(0),
		},
		Media: config.MediaConfig{
			TempDir:                tempDir,
			DownloadRetries:        2,
			MaxConcurrentDownloads: 2,
		},
		Transfer: config.TransferConfig{
			DryRun:             false,
			IncludeMedia:       true,
			PreserveDates:      false,
			AddSourceLink:      true,
			SourceLinkFormat:   "Original: {link}",
			IncludeForwardInfo: false,
			SkipFailedPosts:    false,
			MaxPostsPerBatch:   100,
		},
	}
}

// createTestPosts creates sample Telegram posts for testing
func createTestPosts() []tg.Post {
	now := time.Now()
	return []tg.Post{
		{
			ID:      1001,
			Date:    now.Add(-24 * time.Hour),
			Text:    "First post with text",
			Media:   []tg.Media{},
			ForwardedFrom: nil,
		},
		{
			ID:    1002,
			Date:  now.Add(-12 * time.Hour),
			Text:  "Second post with photo",
			Media: []tg.Media{
				{
					Type:     tg.MediaTypePhoto,
					FileID:   "photo_123",
					FileSize: 1024 * 1024,
					Width:    800,
					Height:   600,
				},
			},
			ForwardedFrom: nil,
		},
		{
			ID:    1003,
			Date:  now.Add(-6 * time.Hour),
			Text:  "Third post with video",
			Media: []tg.Media{
				{
					Type:     tg.MediaTypeVideo,
					FileID:   "video_456",
					FileSize: 5 * 1024 * 1024,
					Width:    1280,
					Height:   720,
					Duration: 60,
				},
			},
			ForwardedFrom: nil,
		},
	}
}

// TestTransferIntegration tests the transfer process with mocked clients
func TestTransferIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := createTestConfig(t)
	
	// Create mocks
	tgMock := &mockTelegramClient{
		posts: createTestPosts(),
	}
	vkMock := &mockVKClient{
		postedPosts: []vk.Post{},
		uploaded:    []string{},
	}
	
	// Create transfer with dependency injection
	// We need to modify NewTransfer to accept mock clients, but it's not designed for that.
	// Instead, we'll create a Transfer struct manually.
	tempDir := cfg.Media.TempDir
	
	// Initialize media storage
	storage, err := media.NewStorageManager(tempDir, 24*time.Hour, 1*1024*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create media storage: %v", err)
	}
	
	// Initialize downloader
	downloader := media.NewDownloader(tempDir, cfg.Media.DownloadRetries, 
		cfg.Media.MaxConcurrentDownloads, storage)
	
	// Initialize transformer
	transformerConfig := transformer.Config{
		PreserveDates:    cfg.Transfer.PreserveDates,
		FromGroup:        cfg.VK.FromGroup,
		AddSourceLink:    cfg.Transfer.AddSourceLink,
		SourceLinkFormat: cfg.Transfer.SourceLinkFormat,
		IncludeForwardInfo: cfg.Transfer.IncludeForwardInfo,
		ForwardFormat:    cfg.Transfer.ForwardFormat,
		MaxPostsPerBatch: cfg.Transfer.MaxPostsPerBatch,
		SkipFailedPosts:  cfg.Transfer.SkipFailedPosts,
	}
	postTransformer := transformer.NewPostTransformer(transformerConfig)
	
	// Initialize checkpoint manager
	checkpoint := NewCheckpointManager(filepath.Join(tempDir, "checkpoint.json"), logger.Named("checkpoint"))
	
	// Initialize progress tracker
	progress := NewProgressTracker(logger.Named("progress"))
	
	transfer := &Transfer{
		config:         cfg,
		logger:         logger.Named("transfer"),
		telegramClient: tgMock,
		vkClient:       vkMock,
		mediaStorage:   storage,
		downloader:     downloader,
		transformer:    postTransformer,
		checkpoint:     checkpoint,
		progress:       progress,
	}
	
	// Run the transfer
	ctx := context.Background()
	err = transfer.Run(ctx)
	if err != nil {
		t.Fatalf("Transfer.Run failed: %v", err)
	}
	
	// Verify results
	if len(vkMock.postedPosts) != len(tgMock.posts) {
		t.Errorf("Expected %d posts to be posted to VK, got %d", len(tgMock.posts), len(vkMock.postedPosts))
	}
	
	// Check that media files were uploaded (for posts with media)
	expectedUploads := 0
	for _, post := range tgMock.posts {
		if len(post.Media) > 0 {
			expectedUploads++
		}
	}
	if len(vkMock.uploaded) != expectedUploads {
		t.Errorf("Expected %d media uploads, got %d", expectedUploads, len(vkMock.uploaded))
	}
	
	// Cleanup
	transfer.Close()
}

// TestTransferDryRun tests dry-run mode where no actual posting happens
func TestTransferDryRun(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := createTestConfig(t)
	cfg.Transfer.DryRun = true
	
	tgMock := &mockTelegramClient{
		posts: createTestPosts(),
	}
	vkMock := &mockVKClient{
		shouldFail: true, // Should not be called in dry-run
	}
	
	// Create transfer manually (same as above)
	tempDir := cfg.Media.TempDir
	storage, err := media.NewStorageManager(tempDir, 24*time.Hour, 1*1024*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create media storage: %v", err)
	}
	downloader := media.NewDownloader(tempDir, cfg.Media.DownloadRetries, 
		cfg.Media.MaxConcurrentDownloads, storage)
	transformerConfig := transformer.Config{
		PreserveDates:    cfg.Transfer.PreserveDates,
		FromGroup:        cfg.VK.FromGroup,
		AddSourceLink:    cfg.Transfer.AddSourceLink,
		SourceLinkFormat: cfg.Transfer.SourceLinkFormat,
		IncludeForwardInfo: cfg.Transfer.IncludeForwardInfo,
		ForwardFormat:    cfg.Transfer.ForwardFormat,
		MaxPostsPerBatch: cfg.Transfer.MaxPostsPerBatch,
		SkipFailedPosts:  cfg.Transfer.SkipFailedPosts,
	}
	postTransformer := transformer.NewPostTransformer(transformerConfig)
	checkpoint := NewCheckpointManager(filepath.Join(tempDir, "checkpoint.json"), logger.Named("checkpoint"))
	progress := NewProgressTracker(logger.Named("progress"))
	
	transfer := &Transfer{
		config:         cfg,
		logger:         logger.Named("transfer"),
		telegramClient: tgMock,
		vkClient:       vkMock,
		mediaStorage:   storage,
		downloader:     downloader,
		transformer:    postTransformer,
		checkpoint:     checkpoint,
		progress:       progress,
	}
	
	ctx := context.Background()
	err = transfer.Run(ctx)
	if err != nil {
		t.Fatalf("Transfer.Run in dry-run mode failed: %v", err)
	}
	
	// In dry-run mode, VK client should not be called
	if len(vkMock.postedPosts) > 0 {
		t.Errorf("Expected no posts in dry-run mode, got %d", len(vkMock.postedPosts))
	}
	
	transfer.Close()
}

// TestTransferWithCheckpoint tests resuming from a checkpoint
func TestTransferWithCheckpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := createTestConfig(t)
	
	posts := createTestPosts()
	tgMock := &mockTelegramClient{
		posts: posts,
	}
	vkMock := &mockVKClient{
		postedPosts: []vk.Post{},
	}
	
	// Create transfer
	tempDir := cfg.Media.TempDir
	storage, err := media.NewStorageManager(tempDir, 24*time.Hour, 1*1024*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create media storage: %v", err)
	}
	downloader := media.NewDownloader(tempDir, cfg.Media.DownloadRetries, 
		cfg.Media.MaxConcurrentDownloads, storage)
	transformerConfig := transformer.Config{
		PreserveDates:    cfg.Transfer.PreserveDates,
		FromGroup:        cfg.VK.FromGroup,
		AddSourceLink:    cfg.Transfer.AddSourceLink,
		SourceLinkFormat: cfg.Transfer.SourceLinkFormat,
		IncludeForwardInfo: cfg.Transfer.IncludeForwardInfo,
		ForwardFormat:    cfg.Transfer.ForwardFormat,
		MaxPostsPerBatch: cfg.Transfer.MaxPostsPerBatch,
		SkipFailedPosts:  cfg.Transfer.SkipFailedPosts,
	}
	postTransformer := transformer.NewPostTransformer(transformerConfig)
	checkpoint := NewCheckpointManager(filepath.Join(tempDir, "checkpoint.json"), logger.Named("checkpoint"))
	progress := NewProgressTracker(logger.Named("progress"))
	
	transfer := &Transfer{
		config:         cfg,
		logger:         logger.Named("transfer"),
		telegramClient: tgMock,
		vkClient:       vkMock,
		mediaStorage:   storage,
		downloader:     downloader,
		transformer:    postTransformer,
		checkpoint:     checkpoint,
		progress:       progress,
	}
	
	// Create a checkpoint after first post
	checkpointState := &CheckpointState{
		LastProcessedIndex: 0, // First post processed
		LastProcessedID:    posts[0].ID,
		TotalPosts:         len(posts),
	}
	if err := checkpoint.Save(checkpointState); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}
	
	// Run transfer - should process only remaining posts
	ctx := context.Background()
	err = transfer.Run(ctx)
	if err != nil {
		t.Fatalf("Transfer.Run with checkpoint failed: %v", err)
	}
	
	// Should have processed 2 posts (posts[1] and posts[2]) since first was already processed
	if len(vkMock.postedPosts) != 2 {
		t.Errorf("Expected 2 posts to be posted (skipping first), got %d", len(vkMock.postedPosts))
	}
	
	transfer.Close()
}

// TestTransferTelegramFailure tests handling of Telegram API failures
func TestTransferTelegramFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := createTestConfig(t)
	
	tgMock := &mockTelegramClient{
		shouldFail: true,
	}
	vkMock := &mockVKClient{}
	
	// Create transfer
	tempDir := cfg.Media.TempDir
	storage, err := media.NewStorageManager(tempDir, 24*time.Hour, 1*1024*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create media storage: %v", err)
	}
	downloader := media.NewDownloader(tempDir, cfg.Media.DownloadRetries, 
		cfg.Media.MaxConcurrentDownloads, storage)
	transformerConfig := transformer.Config{
		PreserveDates:    cfg.Transfer.PreserveDates,
		FromGroup:        cfg.VK.FromGroup,
		AddSourceLink:    cfg.Transfer.AddSourceLink,
		SourceLinkFormat: cfg.Transfer.SourceLinkFormat,
		IncludeForwardInfo: cfg.Transfer.IncludeForwardInfo,
		ForwardFormat:    cfg.Transfer.ForwardFormat,
		MaxPostsPerBatch: cfg.Transfer.MaxPostsPerBatch,
		SkipFailedPosts:  cfg.Transfer.SkipFailedPosts,
	}
	postTransformer := transformer.NewPostTransformer(transformerConfig)
	checkpoint := NewCheckpointManager(filepath.Join(tempDir, "checkpoint.json"), logger.Named("checkpoint"))
	progress := NewProgressTracker(logger.Named("progress"))
	
	transfer := &Transfer{
		config:         cfg,
		logger:         logger.Named("transfer"),
		telegramClient: tgMock,
		vkClient:       vkMock,
		mediaStorage:   storage,
		downloader:     downloader,
		transformer:    postTransformer,
		checkpoint:     checkpoint,
		progress:       progress,
	}
	
	ctx := context.Background()
	err = transfer.Run(ctx)
	if err == nil {
		t.Error("Expected error when Telegram client fails, got nil")
	}
	
	transfer.Close()
}