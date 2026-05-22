package mocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	vk "ai-transfer-tg-to-vk/internal/vk"
)

// MockVKClient implements vk.Client for testing with configurable behavior
type MockVKClient struct {
	// Predefined data
	GroupInfo *vk.GroupInfo
	
	// Behavior control
	ShouldFail      bool
	FailOnCall      map[string]bool // method name -> should fail
	DelayOnCall     map[string]time.Duration // method name -> delay duration
	CallCount       map[string]int // method name -> call count
	
	// Call tracking
	PostedPosts      []vk.Post
	UploadedFiles    []string // paths of uploaded files
	UploadResults    []vk.UploadResult
	PostResults      []vk.PostResult
}

// NewMockVKClient creates a new mock VK client with default test data
func NewMockVKClient() *MockVKClient {
	return &MockVKClient{
		GroupInfo: &vk.GroupInfo{
			ID:          123456,
			Name:        "Test VK Group",
			ScreenName:  "test_group",
			Description: "This is a test VK group for integration testing",
			Members:     1000,
			PhotoURL:    "https://example.com/photo.jpg",
			Type:        "group",
		},
		FailOnCall:  make(map[string]bool),
		DelayOnCall: make(map[string]time.Duration),
		CallCount:   make(map[string]int),
		PostedPosts: []vk.Post{},
		UploadedFiles: []string{},
		UploadResults: []vk.UploadResult{},
		PostResults: []vk.PostResult{},
	}
}

// WithGroupInfo sets custom group info
func (m *MockVKClient) WithGroupInfo(info *vk.GroupInfo) *MockVKClient {
	m.GroupInfo = info
	return m
}

// WithFailure makes the mock fail all calls
func (m *MockVKClient) WithFailure(shouldFail bool) *MockVKClient {
	m.ShouldFail = shouldFail
	return m
}

// WithMethodFailure makes a specific method fail
func (m *MockVKClient) WithMethodFailure(method string, shouldFail bool) *MockVKClient {
	m.FailOnCall[method] = shouldFail
	return m
}

// WithDelay adds a delay to a specific method
func (m *MockVKClient) WithDelay(method string, delay time.Duration) *MockVKClient {
	m.DelayOnCall[method] = delay
	return m
}

// Post implements vk.Client.Post
func (m *MockVKClient) Post(ctx context.Context, post vk.Post) (*vk.PostResult, error) {
	m.incCallCount("Post")
	
	if delay := m.DelayOnCall["Post"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["Post"] {
		return nil, errors.New("mock: Post failed")
	}
	
	m.PostedPosts = append(m.PostedPosts, post)
	
	result := &vk.PostResult{
		PostID:   len(m.PostedPosts),
		PostHash: fmt.Sprintf("hash_%d_%d", time.Now().Unix(), len(m.PostedPosts)),
	}
	
	m.PostResults = append(m.PostResults, *result)
	return result, nil
}

// UploadPhoto implements vk.Client.UploadPhoto
func (m *MockVKClient) UploadPhoto(ctx context.Context, filePath string, groupID int) (*vk.Attachment, error) {
	m.incCallCount("UploadPhoto")
	m.UploadedFiles = append(m.UploadedFiles, filePath)
	
	if delay := m.DelayOnCall["UploadPhoto"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["UploadPhoto"] {
		return nil, errors.New("mock: UploadPhoto failed")
	}
	
	uploadID := len(m.UploadResults) + 1
	m.UploadResults = append(m.UploadResults, vk.UploadResult{
		Server:   12345,
		Photo:    fmt.Sprintf("photo_%d", uploadID),
		Hash:     fmt.Sprintf("hash_%d", uploadID),
		OwnerID:  -groupID,
		MediaID:  uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	})
	
	return &vk.Attachment{
		Type:      vk.AttachmentTypePhoto,
		OwnerID:   -groupID,
		MediaID:   uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	}, nil
}

// UploadVideo implements vk.Client.UploadVideo
func (m *MockVKClient) UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*vk.Attachment, error) {
	m.incCallCount("UploadVideo")
	m.UploadedFiles = append(m.UploadedFiles, filePath)
	
	if delay := m.DelayOnCall["UploadVideo"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["UploadVideo"] {
		return nil, errors.New("mock: UploadVideo failed")
	}
	
	uploadID := len(m.UploadResults) + 1
	m.UploadResults = append(m.UploadResults, vk.UploadResult{
		Server:   12345,
		Video:    fmt.Sprintf("video_%d", uploadID),
		Hash:     fmt.Sprintf("hash_%d", uploadID),
		OwnerID:  -groupID,
		MediaID:  uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	})
	
	return &vk.Attachment{
		Type:      vk.AttachmentTypeVideo,
		OwnerID:   -groupID,
		MediaID:   uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	}, nil
}

// UploadDocument implements vk.Client.UploadDocument
func (m *MockVKClient) UploadDocument(ctx context.Context, filePath, title string, groupID int) (*vk.Attachment, error) {
	m.incCallCount("UploadDocument")
	m.UploadedFiles = append(m.UploadedFiles, filePath)
	
	if delay := m.DelayOnCall["UploadDocument"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["UploadDocument"] {
		return nil, errors.New("mock: UploadDocument failed")
	}
	
	uploadID := len(m.UploadResults) + 1
	m.UploadResults = append(m.UploadResults, vk.UploadResult{
		Server:   12345,
		File:     fmt.Sprintf("doc_%d", uploadID),
		Hash:     fmt.Sprintf("hash_%d", uploadID),
		OwnerID:  -groupID,
		MediaID:  uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	})
	
	return &vk.Attachment{
		Type:      vk.AttachmentTypeDoc,
		OwnerID:   -groupID,
		MediaID:   uploadID,
		AccessKey: fmt.Sprintf("access_key_%d", uploadID),
	}, nil
}

// GetGroupInfo implements vk.Client.GetGroupInfo
func (m *MockVKClient) GetGroupInfo(ctx context.Context, groupID int) (*vk.GroupInfo, error) {
	m.incCallCount("GetGroupInfo")
	
	if delay := m.DelayOnCall["GetGroupInfo"]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.ShouldFail || m.FailOnCall["GetGroupInfo"] {
		return nil, errors.New("mock: GetGroupInfo failed")
	}
	
	return m.GroupInfo, nil
}

// Close implements vk.Client.Close
func (m *MockVKClient) Close() error {
	m.incCallCount("Close")
	return nil
}

// GetCallCount returns how many times a method was called
func (m *MockVKClient) GetCallCount(method string) int {
	return m.CallCount[method]
}

// ResetCallCount resets all call counters
func (m *MockVKClient) ResetCallCount() {
	for k := range m.CallCount {
		m.CallCount[k] = 0
	}
}

// GetPostedPosts returns all posts that were posted
func (m *MockVKClient) GetPostedPosts() []vk.Post {
	return m.PostedPosts
}

// GetUploadedFiles returns all file paths that were uploaded
func (m *MockVKClient) GetUploadedFiles() []string {
	return m.UploadedFiles
}

// Helper method to increment call count
func (m *MockVKClient) incCallCount(method string) {
	m.CallCount[method] = m.CallCount[method] + 1
}