package transformer

import (
	"fmt"
	"strings"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
)

// MediaTransformer converts Telegram media attachments to VK attachments.
// It maps media types and handles unsupported formats according to configuration.
type MediaTransformer struct {
	// Configuration
	skipUnsupported bool // If true, skip unsupported media types with warning
	maxAttachments  int  // VK limit: 10 attachments per post
	videoConversion bool // Whether to attempt video conversion (not implemented yet)
	gifToMP4        bool // Whether to convert GIFs to MP4 (not implemented yet)
}

// NewMediaTransformer creates a new MediaTransformer with default settings.
func NewMediaTransformer() *MediaTransformer {
	return &MediaTransformer{
		skipUnsupported: true,
		maxAttachments:  10,
		videoConversion: false,
		gifToMP4:        false,
	}
}

// Transform converts a slice of Telegram media items to VK attachments.
// Returns:
//   - attachments: VK attachments that can be uploaded
//   - warnings: descriptive messages about skipped or unsupported media
//   - error: only returned if skipUnsupported is false and an unsupported type is encountered
func (m *MediaTransformer) Transform(media []tg.Media) ([]vk.Attachment, []string, error) {
	var attachments []vk.Attachment
	var warnings []string

	for i, item := range media {
		if i >= m.maxAttachments {
			warnings = append(warnings, fmt.Sprintf("%s: attachment limit exceeded (max %d)", item.Type, m.maxAttachments))
			break
		}

		attachment, err := m.convertMedia(item)
		if err != nil {
			if m.skipUnsupported {
				warnings = append(warnings, fmt.Sprintf("%s: %v", item.Type, err))
				continue
			}
			return nil, warnings, err
		}

		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
	}

	return attachments, warnings, nil
}

// convertMedia maps a single Telegram media item to a VK attachment.
// This creates a stub attachment; actual upload will fill OwnerID and MediaID.
func (m *MediaTransformer) convertMedia(media tg.Media) (*vk.Attachment, error) {
	switch media.Type {
	case tg.MediaTypePhoto:
		return &vk.Attachment{
			Type: vk.AttachmentTypePhoto,
		}, nil
	case tg.MediaTypeVideo:
		return &vk.Attachment{
			Type: vk.AttachmentTypeVideo,
		}, nil
	case tg.MediaTypeDocument:
		// Determine actual type by MIME type
		mimeType := strings.ToLower(media.MimeType)
		if strings.HasPrefix(mimeType, "image/") {
			return &vk.Attachment{
				Type: vk.AttachmentTypePhoto,
			}, nil
		} else if strings.HasPrefix(mimeType, "video/") {
			return &vk.Attachment{
				Type: vk.AttachmentTypeVideo,
			}, nil
		} else if strings.HasPrefix(mimeType, "audio/") {
			return &vk.Attachment{
				Type: vk.AttachmentTypeAudio,
			}, nil
		} else {
			// Generic document
			return &vk.Attachment{
				Type: vk.AttachmentTypeDoc,
			}, nil
		}
	case tg.MediaTypeAudio:
		return &vk.Attachment{
			Type: vk.AttachmentTypeAudio,
		}, nil
	case tg.MediaTypeVoice:
		// Voice messages are audio in OGG format, treat as audio
		return &vk.Attachment{
			Type: vk.AttachmentTypeAudio,
		}, nil
	case tg.MediaTypeSticker:
		// Stickers can be uploaded as photos
		return &vk.Attachment{
			Type: vk.AttachmentTypePhoto,
		}, nil
	case tg.MediaTypeAnimation:
		// GIFs: if conversion enabled, treat as video; otherwise as document
		if m.gifToMP4 {
			return &vk.Attachment{
				Type: vk.AttachmentTypeVideo,
			}, nil
		}
		return &vk.Attachment{
			Type: vk.AttachmentTypeDoc,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported media type: %s", media.Type)
	}
}

// WithSkipUnsupported sets whether unsupported media types should be skipped.
func (m *MediaTransformer) WithSkipUnsupported(skip bool) *MediaTransformer {
	m.skipUnsupported = skip
	return m
}

// WithMaxAttachments sets the maximum number of attachments per post.
func (m *MediaTransformer) WithMaxAttachments(max int) *MediaTransformer {
	m.maxAttachments = max
	return m
}

// WithVideoConversion enables or disables video conversion (placeholder).
func (m *MediaTransformer) WithVideoConversion(enable bool) *MediaTransformer {
	m.videoConversion = enable
	return m
}

// WithGifToMP4 enables or disables GIF to MP4 conversion (placeholder).
func (m *MediaTransformer) WithGifToMP4(enable bool) *MediaTransformer {
	m.gifToMP4 = enable
	return m
}