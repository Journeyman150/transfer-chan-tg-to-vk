package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Client interface for Telegram operations
type Client interface {
	// GetChannelInfo fetches channel metadata
	GetChannelInfo(ctx context.Context, channelID string) (*ChannelInfo, error)

	// GetPosts fetches posts from channel with pagination
	GetPosts(ctx context.Context, channelID string, opts GetPostsOptions) ([]Post, error)

	// GetPostByID fetches a specific post
	GetPostByID(ctx context.Context, channelID string, postID int64) (*Post, error)

	// GetMediaURL generates download URL for media file
	GetMediaURL(ctx context.Context, fileID string) (string, error)

	// DownloadMedia downloads media to local file
	DownloadMedia(ctx context.Context, fileID, destPath string) error

	// Close releases resources
	Close() error
}

// BotAPIClient implements Client using go-telegram-bot-api
type BotAPIClient struct {
	bot     *tgbotapi.BotAPI
	config  Config
	limiter *rate.Limiter
	logger  *zap.Logger
}

// Config holds Telegram client configuration
type Config struct {
	BotToken         string
	APIURL           string
	Timeout          time.Duration
	Debug            bool
	RequestsPerSecond int
	MaxRetries       int
}

// NewBotAPIClient creates a new Telegram client
func NewBotAPIClient(config Config) (*BotAPIClient, error) {
	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint(config.BotToken, config.APIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = config.Debug

	// Set custom HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}
	bot.Client = client

	logger := zap.L().Named("telegram")

	return &BotAPIClient{
		bot:     bot,
		config:  config,
		limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 1),
		logger:  logger,
	}, nil
}

// createChatConfig creates a ChatConfig from a channel identifier string.
func createChatConfig(channelID string) tgbotapi.ChatConfig {
	// Check if channelID is a numeric ID
	if id, err := strconv.ParseInt(channelID, 10, 64); err == nil {
		return tgbotapi.ChatConfig{
			ChatID: id,
		}
	}
	// Check if it's a username starting with '@'
	if strings.HasPrefix(channelID, "@") {
		return tgbotapi.ChatConfig{
			SuperGroupUsername: channelID,
		}
	}
	// Assume it's a username without '@' (add it)
	if len(channelID) > 0 && channelID[0] != '@' {
		return tgbotapi.ChatConfig{
			SuperGroupUsername: "@" + channelID,
		}
	}
	// Fallback: treat as ChatID (will likely fail)
	return tgbotapi.ChatConfig{
		ChatID: 0,
	}
}

// GetChannelInfo fetches channel metadata
func (c *BotAPIClient) GetChannelInfo(ctx context.Context, channelID string) (*ChannelInfo, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	chatConfig := createChatConfig(channelID)
	chat, err := c.bot.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: chatConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chat info: %w", err)
	}

	// Determine member count (if available)
	members := 0
	// Note: The Chat struct may have MembersCount field; we'll skip for now.

	info := &ChannelInfo{
		ID:          chat.ID,
		Username:    chat.UserName,
		Title:       chat.Title,
		Description: chat.Description,
		Members:     members,
		PhotoURL:    "", // TODO: extract photo URL if available
		IsPublic:    chat.UserName != "",
	}

	return info, nil
}

// GetPosts fetches posts from channel with pagination
func (c *BotAPIClient) GetPosts(ctx context.Context, channelID string, opts GetPostsOptions) ([]Post, error) {
	// TODO: Implement post fetching
	c.logger.Warn("GetPosts not yet implemented", zap.String("channel", channelID))
	return nil, fmt.Errorf("not implemented")
}

// GetPostByID fetches a specific post
func (c *BotAPIClient) GetPostByID(ctx context.Context, channelID string, postID int64) (*Post, error) {
	// TODO: Implement
	return nil, fmt.Errorf("not implemented")
}

// GetMediaURL generates download URL for media file
func (c *BotAPIClient) GetMediaURL(ctx context.Context, fileID string) (string, error) {
	// Telegram Bot API provides file download via getFile
	// URL format: https://api.telegram.org/file/bot<token>/<file_path>
	// We need to call getFile to get file_path
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait: %w", err)
	}

	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	url := file.Link(c.bot.Token)
	return url, nil
}

// DownloadMedia downloads media to local file
func (c *BotAPIClient) DownloadMedia(ctx context.Context, fileID, destPath string) error {
	_, err := c.GetMediaURL(ctx, fileID)
	if err != nil {
		return err
	}

	// TODO: implement download with retry and progress
	return fmt.Errorf("download not implemented")
}

// Close releases resources
func (c *BotAPIClient) Close() error {
	// Nothing to close for now
	return nil
}

// convertMessage converts tgbotapi.Message to our Post type
func (c *BotAPIClient) convertMessage(msg tgbotapi.Message) Post {
	// TODO: implement conversion
	return Post{}
}