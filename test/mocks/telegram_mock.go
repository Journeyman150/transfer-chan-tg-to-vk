package mocks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MockTelegramClient implements tg.Client for testing with configurable behavior
type MockTelegramClient struct {
	// Predefined data
	ChannelInfo *tg.ChannelInfo
	Posts       []tg.Post
	
	// Behavior control
	ShouldFail      bool
	FailOnCall      map[string]bool // method name -> should fail
	DelayOnCall     map[string]time.Duration // method name -> delay duration
	CallCount       map[string]int // method name -> call count
	
	// Call tracking
	DownloadedFiles []string
	GetMediaURLCalls []string
}

// NewMockTelegramClient creates a new mock Telegram client with default test data
func NewMockTelegramClient() *MockTelegramClient {
	return &MockTelegramClient{
		ChannelInfo: &tg.ChannelInfo{
			ID:       123456789,
			Username: "test_channel",
			Title:    "Test Telegram Channel",
			IsPublic: true,
		},
		Posts: []tg.Post{
			{
				ID:   1,
				Date: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Text: "First test post with **bold** text",
				Entities: []tgbotapi.MessageEntity{
					{Type: "bold", Offset: 22, Length: 4},
				},
				Media: []tg.Media{
					{
						Type:        tg.MediaTypePhoto,
						FileID:      "photo_123",
						FileUniqueID: "unique_photo_123",
						FileSize:    1024 * 1024,
						Width:       1920,
						Height:      1080,
					},
				},
				Link: "t.me/test_channel/1",
			},
			{
				ID:   2,
				Date: time.Date(2024, 1, 2, 14, 30, 0, 0, time.UTC),
				Text: "Second post with a video",
				Media: []tg.Media{
					{
						Type:        tg.MediaTypeVideo,
						FileID:      "video_456",
						FileUniqueID: "unique_video_456",
						FileSize:    5 * 1024 * 1024,
						Width:       1280,
						Height:      720,
						Duration:    60,
					},
				},
				Link: "t.me/test_channel/2",
			},
		},
		FailOnCall:  make(map[string]bool),
		DelayOnCall: make(map[string]time.Duration),
		CallCount:   make(map[string]int),
		DownloadedFiles: []string{},
		GetMediaURLCalls: []string{},
	}
}

// WithChannelInfo sets custom channel info
func (m *MockTelegramClient) WithChannelInfo(info *tg.ChannelInfo) *MockTelegramClient {
	m.ChannelInfo = info
	return m
}

// WithPosts sets custom posts
func (m *MockTelegramClient) WithPosts(posts []tg.Post) *MockTelegramClient {
	m.Posts = posts
	return m
}

// WithFailure makes the mock fail all calls
func (m *MockTelegramClient) WithFailure(shouldFail bool) *MockTelegramClient {
	m.ShouldFail = shouldFail
	return m
}

// WithMethodFailure makes a specific method fail
func (m *MockTelegramClient) WithMethodFailure(method string, shouldFail bool) *MockTelegramClient {
	m.FailOnCall[method] = shouldFail
	return m
}

// WithDelay adds a delay to a specific method
func (m *MockTelegramClient) WithDelay(method string, delay time.Duration) *MockTelegramClient {
	m.DelayOnCall[method] = delay
	return m
}

// GetChannelInfo implements tg.Client.GetChannelInfo
func (m *MockTelegramClient) GetChannelInfo(ctx context.Context, channelID string) (*tg.ChannelInfo, error) {
	m.incCallCount("GetChannelInfo")
	
	if delay := m.DelayOnCall["GetChannelInfo"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["GetChannelInfo"] {
		return nil, errors.New("mock: GetChannelInfo failed")
	}
	
	return m.ChannelInfo, nil
}

// GetPosts implements tg.Client.GetPosts
func (m *MockTelegramClient) GetPosts(ctx context.Context, channelID string, opts tg.GetPostsOptions) ([]tg.Post, error) {
	m.incCallCount("GetPosts")
	
	if delay := m.DelayOnCall["GetPosts"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["GetPosts"] {
		return nil, errors.New("mock: GetPosts failed")
	}
	
	// Apply pagination simulation
	start := 0
	if opts.OffsetID > 0 {
		for i, post := range m.Posts {
			if post.ID < opts.OffsetID {
				start = i + 1
			}
		}
	}
	
	end := len(m.Posts)
	if opts.Limit > 0 && start+opts.Limit < end {
		end = start + opts.Limit
	}
	
	if start >= len(m.Posts) {
		return []tg.Post{}, nil
	}
	
	return m.Posts[start:end], nil
}

// GetPostByID implements tg.Client.GetPostByID
func (m *MockTelegramClient) GetPostByID(ctx context.Context, channelID string, postID int64) (*tg.Post, error) {
	m.incCallCount("GetPostByID")
	
	if delay := m.DelayOnCall["GetPostByID"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["GetPostByID"] {
		return nil, errors.New("mock: GetPostByID failed")
	}
	
	for _, post := range m.Posts {
		if post.ID == postID {
			return &post, nil
		}
	}
	
	return nil, fmt.Errorf("post %d not found", postID)
}

// GetMediaURL implements tg.Client.GetMediaURL
func (m *MockTelegramClient) GetMediaURL(ctx context.Context, fileID string) (string, error) {
	m.incCallCount("GetMediaURL")
	m.GetMediaURLCalls = append(m.GetMediaURLCalls, fileID)
	
	if delay := m.DelayOnCall["GetMediaURL"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["GetMediaURL"] {
		return "", errors.New("mock: GetMediaURL failed")
	}
	
	return fmt.Sprintf("https://api.telegram.org/file/bot_token/%s", fileID), nil
}

// DownloadMedia implements tg.Client.DownloadMedia
func (m *MockTelegramClient) DownloadMedia(ctx context.Context, fileID, destPath string) error {
	m.incCallCount("DownloadMedia")
	m.DownloadedFiles = append(m.DownloadedFiles, destPath)
	
	if delay := m.DelayOnCall["DownloadMedia"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["DownloadMedia"] {
		return errors.New("mock: DownloadMedia failed")
	}
	
	// Create a dummy file with some content
	content := fmt.Sprintf("Mock media content for file ID: %s", fileID)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// Close implements tg.Client.Close
func (m *MockTelegramClient) Close() error {
	m.incCallCount("Close")
	return nil
}

// GetCallCount returns how many times a method was called
func (m *MockTelegramClient) GetCallCount(method string) int {
	return m.CallCount[method]
}

// ResetCallCount resets all call counters
func (m *MockTelegramClient) ResetCallCount() {
	for k := range m.CallCount {
		m.CallCount[k] = 0
	}
}

// Helper method to increment call count
func (m *MockTelegramClient) incCallCount(method string) {
	m.CallCount[method] = m.CallCount[method] + 1
}