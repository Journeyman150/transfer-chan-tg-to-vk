package transformer

import (
	"fmt"
	"time"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
	"go.uber.org/zap"
)

// PostTransformer combines text and media transformations to produce VK posts.
type PostTransformer struct {
	textTransformer  *TextTransformer
	mediaTransformer *MediaTransformer
	config           Config
	logger           *zap.Logger
}

// Config holds transformation configuration.
type Config struct {
	PreserveDates      bool
	FromGroup          bool
	AddSourceLink      bool
	SourceLinkFormat   string // e.g., "Source: %s"
	IncludeForwardInfo bool
	ForwardFormat      string // e.g., "↪️ Forwarded from %s"
	MaxPostsPerBatch   int
	SkipFailedPosts    bool
}

// NewPostTransformer creates a new PostTransformer with default components.
func NewPostTransformer(config Config) *PostTransformer {
	return &PostTransformer{
		textTransformer:  NewTextTransformer(),
		mediaTransformer: NewMediaTransformer(),
		config:           config,
		logger:           zap.L().Named("transformer"),
	}
}

// TransformPost converts a single Telegram post to a VK post.
// Returns the VK post, warnings (skipped media, etc.), and any error.
func (pt *PostTransformer) TransformPost(tgPost tg.Post) (*vk.Post, []string, error) {
	var warnings []string

	// 1. Transform text
	text, err := pt.textTransformer.Transform(tgPost.Text, tgPost.Entities)
	if err != nil {
		return nil, warnings, fmt.Errorf("text transformation failed: %w", err)
	}

	// 2. Add source link if configured
	if pt.config.AddSourceLink && tgPost.Link != "" {
		sourceText := fmt.Sprintf(pt.config.SourceLinkFormat, tgPost.Link)
		if text == "" {
			text = sourceText
		} else {
			text = text + "\n\n" + sourceText
		}
	}

	// 3. Transform media
	attachments, skippedMedia, err := pt.mediaTransformer.Transform(tgPost.Media)
	if err != nil {
		return nil, warnings, fmt.Errorf("media transformation failed: %w", err)
	}

	// Add media warnings
	for _, skipped := range skippedMedia {
		warnings = append(warnings, fmt.Sprintf("Skipped media: %s", skipped))
	}

	// 4. Handle forwarded posts
	if pt.config.IncludeForwardInfo && tgPost.ForwardedFrom != nil {
		forwardText := pt.formatForwardInfo(tgPost.ForwardedFrom)
		if text == "" {
			text = forwardText
		} else {
			text = forwardText + "\n\n" + text
		}
	}

	// 5. Set publish date
	var publishDate *time.Time
	if pt.config.PreserveDates {
		publishDate = &tgPost.Date
	}

	// 6. Create VK post
	vkPost := &vk.Post{
		Text:        text,
		Attachments: attachments,
		PublishDate: publishDate,
		FromGroup:   pt.config.FromGroup,
		Signed:      true, // Show group signature
		FriendsOnly: false,
	}

	return vkPost, warnings, nil
}

// formatForwardInfo creates a formatted string describing the forwarded source.
func (pt *PostTransformer) formatForwardInfo(info *tg.ForwardInfo) string {
	if info == nil {
		return ""
	}

	var source string
	if info.FromChatName != "" {
		source = info.FromChatName
	} else {
		source = fmt.Sprintf("chat %d", info.FromChatID)
	}

	format := pt.config.ForwardFormat
	if format == "" {
		format = "↪️ Forwarded from %s"
	}
	return fmt.Sprintf(format, source)
}

// TransformPosts converts multiple Telegram posts to VK posts.
// Stops early if a post fails and SkipFailedPosts is false.
func (pt *PostTransformer) TransformPosts(tgPosts []tg.Post) ([]vk.Post, []string, error) {
	var vkPosts []vk.Post
	var allWarnings []string

	for i, tgPost := range tgPosts {
		pt.logger.Info("Transforming post",
			zap.Int("index", i),
			zap.Int64("id", tgPost.ID),
			zap.Time("date", tgPost.Date))

		vkPost, warnings, err := pt.TransformPost(tgPost)
		if err != nil {
			pt.logger.Error("Failed to transform post",
				zap.Int64("id", tgPost.ID),
				zap.Error(err))

			// Decide whether to skip or fail
			if pt.config.SkipFailedPosts {
				allWarnings = append(allWarnings,
					fmt.Sprintf("Skipped post %d: %v", tgPost.ID, err))
				continue
			} else {
				return nil, allWarnings, fmt.Errorf("failed to transform post %d: %w",
					tgPost.ID, err)
			}
		}

		vkPosts = append(vkPosts, *vkPost)
		allWarnings = append(allWarnings, warnings...)

		// Check batch limit
		if pt.config.MaxPostsPerBatch > 0 &&
			len(vkPosts) >= pt.config.MaxPostsPerBatch {
			pt.logger.Info("Reached batch limit",
				zap.Int("limit", pt.config.MaxPostsPerBatch))
			break
		}
	}

	return vkPosts, allWarnings, nil
}