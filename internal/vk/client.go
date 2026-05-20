package vk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	vksdk "github.com/SevereCloud/vksdk/v2/api"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Client interface for VK operations
type Client interface {
	// Post creates a new wall post
	Post(ctx context.Context, post Post) (*PostResult, error)

	// UploadPhoto uploads photo to wall
	UploadPhoto(ctx context.Context, filePath string, groupID int) (*Attachment, error)

	// UploadVideo uploads video to wall
	UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*Attachment, error)

	// UploadDocument uploads document to wall
	UploadDocument(ctx context.Context, filePath, title string, groupID int) (*Attachment, error)

	// GetGroupInfo fetches group information
	GetGroupInfo(ctx context.Context, groupID int) (*GroupInfo, error)

	// Close releases resources
	Close() error
}

// Config holds VK client configuration
type Config struct {
	AccessToken      string
	APIURL           string
	APIVersion       string
	Timeout          time.Duration
	Language         string
	RequestsPerSecond float64
	MaxRetries       int
	GroupID          int
	FromGroup        bool
	Signed           bool
	FriendsOnly      bool
	PostDelay        time.Duration
	AlbumID          int
}

// SDKClient implements Client using vksdk
type SDKClient struct {
	api     *vksdk.VK
	config  Config
	limiter *rate.Limiter
	logger  *zap.Logger
	groupID int
}

// NewClient creates a new VK client
func NewClient(config Config) (*SDKClient, error) {
	api := vksdk.NewVK(config.AccessToken)

	// Set API version
	if config.APIVersion != "" {
		api.Version = config.APIVersion
	} else {
		api.Version = "5.199"
	}

	// Create HTTP client with timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	httpClient := &http.Client{
		Timeout: timeout,
	}
	api.Client = httpClient

	// Set custom API URL if provided
	if config.APIURL != "" {
		api.MethodURL = config.APIURL
	}

	logger := zap.L().Named("vk")

	// Rate limiter
	rps := config.RequestsPerSecond
	if rps <= 0 {
		rps = 3.0
	}
	limiter := rate.NewLimiter(rate.Limit(rps), 1)

	return &SDKClient{
		api:     api,
		config:  config,
		limiter: limiter,
		logger:  logger,
		groupID: config.GroupID,
	}, nil
}

// Close releases resources (currently nothing to close)
func (c *SDKClient) Close() error {
	return nil
}

// helper functions

func (c *SDKClient) getOwnerID(fromGroup bool) int {
	if fromGroup {
		return -c.groupID // Negative for groups
	}
	return c.groupID
}

func (c *SDKClient) formatAttachment(att Attachment) string {
	// Format: type{owner_id}_{media_id}_{access_key}
	// Example: photo-123456_7890123
	result := fmt.Sprintf("%s%d_%d", string(att.Type), att.OwnerID, att.MediaID)
	if att.AccessKey != "" {
		result += "_" + att.AccessKey
	}
	return result
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Post creates a new wall post
func (c *SDKClient) Post(ctx context.Context, post Post) (*PostResult, error) {
	// Rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Prepare parameters
	params := vksdk.Params{
		"owner_id":     c.getOwnerID(post.FromGroup),
		"message":      post.Text,
		"friends_only": boolToInt(post.FriendsOnly),
		"services":     post.Services,
		"signed":       boolToInt(post.Signed),
	}

	// Add location if provided
	if post.Lat != 0 || post.Long != 0 {
		params["lat"] = post.Lat
		params["long"] = post.Long
	}
	if post.PlaceID != 0 {
		params["place_id"] = post.PlaceID
	}

	// Add attachments
	if len(post.Attachments) > 0 {
		attachments := make([]string, 0, len(post.Attachments))
		for _, att := range post.Attachments {
			attachments = append(attachments, c.formatAttachment(att))
		}
		params["attachments"] = strings.Join(attachments, ",")
	}

	// Schedule post if publish date is in future
	if post.PublishDate != nil && post.PublishDate.After(time.Now()) {
		params["publish_date"] = post.PublishDate.Unix()
	}

	// Make API call
	response, err := c.api.WallPost(params)
	if err != nil {
		return nil, fmt.Errorf("wall.post failed: %w", err)
	}

	c.logger.Info("Post created",
		zap.Int("post_id", response.PostID),
		zap.Int("attachments", len(post.Attachments)))

	return &PostResult{
		PostID: response.PostID,
	}, nil
}

// UploadPhoto uploads photo to wall
func (c *SDKClient) UploadPhoto(ctx context.Context, filePath string, groupID int) (*Attachment, error) {
	// Rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// 1. Get upload server
	serverResp, err := c.api.PhotosGetWallUploadServer(vksdk.Params{
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get upload server: %w", err)
	}

	// 2. Upload file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("photo", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", serverResp.UploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	// 3. Parse upload response
	var uploadResult UploadResult
	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}

	// 4. Save photo
	saveResp, err := c.api.PhotosSaveWallPhoto(vksdk.Params{
		"group_id": groupID,
		"server":   uploadResult.Server,
		"photo":    uploadResult.Photo,
		"hash":     uploadResult.Hash,
	})
	if err != nil || len(saveResp) == 0 {
		return nil, fmt.Errorf("failed to save photo: %w", err)
	}

	photo := saveResp[0]
	return &Attachment{
		Type:      AttachmentTypePhoto,
		OwnerID:   photo.OwnerID,
		MediaID:   photo.ID,
		AccessKey: photo.AccessKey,
	}, nil
}

// UploadVideo uploads video to wall
func (c *SDKClient) UploadVideo(ctx context.Context, filePath, title, description string, groupID int) (*Attachment, error) {
	// Rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// 1. Get upload URL
	_, err := c.api.VideoSave(vksdk.Params{
		"name":        title,
		"description": description,
		"group_id":    groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get video upload URL: %w", err)
	}

	// 2. Upload file (similar to photo upload)
	// For simplicity, we'll reuse photo upload logic but with different endpoint
	// This is a placeholder - actual video upload requires multipart upload to resp.UploadURL
	// We'll implement a generic upload function later
	return nil, fmt.Errorf("video upload not yet implemented")
}

// UploadDocument uploads document to wall
func (c *SDKClient) UploadDocument(ctx context.Context, filePath, title string, groupID int) (*Attachment, error) {
	// Rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// 1. Get upload server
	_, err := c.api.DocsGetWallUploadServer(vksdk.Params{
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get upload server: %w", err)
	}

	// 2. Upload file (similar to photo upload)
	// Placeholder
	return nil, fmt.Errorf("document upload not yet implemented")
}