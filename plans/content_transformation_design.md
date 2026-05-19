# Content Transformation Logic Design

## Overview
Transforms Telegram posts into VK posts, handling text formatting, media mapping, and structural differences between platforms.

## Transformation Pipeline
```
Telegram Post → Text Transformer → Media Mapper → Post Assembler → VK Post
       ↓               ↓               ↓               ↓
  Parse Entities  Format Conversion Attachment Mapping Final Validation
```

## Text Transformation

### Telegram Text Features
1. **Entities**: Bold, italic, code, pre, text links, mentions, hashtags
2. **Markdown/MarkdownV2**: Optional formatting modes
3. **HTML**: Alternative formatting option
4. **Line breaks**: Preserved as `\n`
5. **Spoilers**: Hidden text (spoiler tags)

### VK Text Features
1. **Limited HTML**: `<b>`, `<i>`, `<s>`, `<u>`, `<code>`, `<pre>`
2. **No Markdown**: Only HTML tags
3. **Link format**: `[id|text]` for internal links, `<a href="url">text</a>` for external
4. **Line breaks**: Preserved
5. **No spoiler tags**: Use text indication instead

### Transformation Rules

#### Entity Mapping Table
| Telegram Entity | VK Equivalent | Notes |
|-----------------|---------------|-------|
| `bold` | `<b>text</b>` | Direct mapping |
| `italic` | `<i>text</i>` | Direct mapping |
| `code` | `<code>text</code>` | Direct mapping |
| `pre` | `<pre>text</pre>` | Preserve language if available |
| `text_link` | `<a href="url">text</a>` | External links |
| `text_mention` | `[id|text]` | User mention → VK user link |
| `mention` | `@username` → keep as text | No direct VK equivalent |
| `hashtag` | `#tag` → keep as text | VK doesn't support hashtags in same way |
| `url` | `<a href="url">url</a>` | Auto-detected URLs |
| `email` | Keep as text | No special formatting |
| `phone_number` | Keep as text | No special formatting |
| `spoiler` | `[SPOILER] text [/SPOILER]` | Custom notation |
| `strikethrough` | `<s>text</s>` | Direct mapping |
| `underline` | `<u>text</u>` | Direct mapping |

### Implementation

```go
package transformer

import (
	"fmt"
	"strings"
	"unicode/utf8"
	
	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
)

type TextTransformer struct {
	// Configuration
	preserveFormatting bool
	maxLength          int // VK limit: 10000 characters
	convertMentions    bool
	convertHashtags    bool
	spoilerFormat      string // e.g., "[SPOILER]%s[/SPOILER]"
}

func NewTextTransformer() *TextTransformer {
	return &TextTransformer{
		preserveFormatting: true,
		maxLength:          10000,
		convertMentions:    false, // VK doesn't have @mentions
		convertHashtags:    false, // Keep as plain text
		spoilerFormat:      "🚨 СПОЙЛЕР: %s 🚨",
	}
}

func (t *TextTransformer) Transform(text string, entities []tg.MessageEntity) (string, error) {
	if text == "" {
		return "", nil
	}
	
	// Convert Telegram entities to VK HTML
	result := t.applyEntities(text, entities)
	
	// Truncate if too long
	result = t.truncateText(result)
	
	// Clean up any invalid HTML
	result = t.cleanHTML(result)
	
	return result, nil
}

func (t *TextTransformer) applyEntities(text string, entities []tg.MessageEntity) string {
	if len(entities) == 0 {
		return text
	}
	
	// Sort entities by offset (ascending) and length (descending)
	// to handle nested entities properly
	sortedEntities := make([]tg.MessageEntity, len(entities))
	copy(sortedEntities, entities)
	sort.Slice(sortedEntities, func(i, j int) bool {
		if sortedEntities[i].Offset == sortedEntities[j].Offset {
			return sortedEntities[i].Length > sortedEntities[j].Length
		}
		return sortedEntities[i].Offset < sortedEntities[j].Offset
	})
	
	// Convert text to runes for proper Unicode handling
	runes := []rune(text)
	var result strings.Builder
	lastPos := 0
	
	for _, entity := range sortedEntities {
		// Append text before this entity
		if entity.Offset > lastPos {
			result.WriteString(string(runes[lastPos:entity.Offset]))
		}
		
		// Calculate end position
		endPos := entity.Offset + entity.Length
		if endPos > len(runes) {
			endPos = len(runes)
		}
		
		// Extract entity text
		entityText := string(runes[entity.Offset:endPos])
		
		// Apply formatting based on entity type
		formatted := t.formatEntity(entityText, entity)
		result.WriteString(formatted)
		
		lastPos = endPos
	}
	
	// Append remaining text
	if lastPos < len(runes) {
		result.WriteString(string(runes[lastPos:]))
	}
	
	return result.String()
}

func (t *TextTransformer) formatEntity(text string, entity tg.MessageEntity) string {
	if !t.preserveFormatting {
		return text
	}
	
	switch entity.Type {
	case "bold":
		return fmt.Sprintf("<b>%s</b>", text)
	case "italic":
		return fmt.Sprintf("<i>%s</i>", text)
	case "code":
		return fmt.Sprintf("<code>%s</code>", text)
	case "pre":
		lang := ""
		if entity.Language != "" {
			lang = fmt.Sprintf(" class=\"language-%s\"", entity.Language)
		}
		return fmt.Sprintf("<pre%s>%s</pre>", lang, text)
	case "text_link":
		return fmt.Sprintf("<a href=\"%s\">%s</a>", entity.URL, text)
	case "text_mention":
		if t.convertMentions && entity.User != nil {
			// Convert to VK user link format: [id|text]
			return fmt.Sprintf("[id%d|%s]", entity.User.ID, text)
		}
		return text
	case "mention":
		// Keep @username as plain text
		return text
	case "hashtag":
		// Keep hashtag as plain text
		return text
	case "url":
		return fmt.Sprintf("<a href=\"%s\">%s</a>", text, text)
	case "email":
		return text
	case "phone_number":
		return text
	case "spoiler":
		return fmt.Sprintf(t.spoilerFormat, text)
	case "strikethrough":
		return fmt.Sprintf("<s>%s</s>", text)
	case "underline":
		return fmt.Sprintf("<u>%s</u>", text)
	default:
		return text
	}
}

func (t *TextTransformer) truncateText(text string) string {
	if utf8.RuneCountInString(text) <= t.maxLength {
		return text
	}
	
	// Truncate to max length, preserving HTML tags if possible
	runes := []rune(text)
	truncated := string(runes[:t.maxLength-3]) + "..."
	
	// Simple fix for broken HTML tags (could be more sophisticated)
	return t.fixBrokenHTML(truncated)
}
```

## Media Transformation

### Mapping Telegram Media to VK Attachments
```go
package transformer

type MediaTransformer struct {
	// Configuration
	skipUnsupported bool
	maxAttachments  int // VK limit: 10 attachments per post
	videoConversion bool
}

func NewMediaTransformer() *MediaTransformer {
	return &MediaTransformer{
		skipUnsupported: true,
		maxAttachments:  10,
		videoConversion: true,
	}
}

func (m *MediaTransformer) Transform(media []tg.Media) ([]vk.Attachment, []string, error) {
	var attachments []vk.Attachment
	var skipped []string
	
	for i, item := range media {
		if i >= m.maxAttachments {
			skipped = append(skipped, fmt.Sprintf("%s: limit exceeded", item.Type))
			break
		}
		
		attachment, err := m.convertMedia(item)
		if err != nil {
			if m.skipUnsupported {
				skipped = append(skipped, fmt.Sprintf("%s: %v", item.Type, err))
				continue
			}
			return nil, skipped, err
		}
		
		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
	}
	
	return attachments, skipped, nil
}

func (m *MediaTransformer) convertMedia(media tg.Media) (*vk.Attachment, error) {
	// This is a placeholder - actual conversion happens in media uploader
	// Returns a "stub" attachment that will be filled during upload
	
	switch media.Type {
	case tg.MediaTypePhoto:
		return &vk.Attachment{
			Type: vk.AttachmentTypePhoto,
			// OwnerID and MediaID will be set after upload
		}, nil
	case tg.MediaTypeVideo:
		return &vk.Attachment{
			Type: vk.AttachmentTypeVideo,
		}, nil
	case tg.MediaTypeDocument:
		// Check if document is actually an image/video/audio
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
			return &vk.Attachment{
				Type: vk.AttachmentTypeDoc,
			}, nil
		}
	case tg.MediaTypeAudio:
		return &vk.Attachment{
			Type: vk.AttachmentTypeAudio,
		}, nil
	case tg.MediaTypeVoice:
		// Voice messages are audio in OGG format
		return &vk.Attachment{
			Type: vk.AttachmentTypeAudio,
		}, nil
	case tg.MediaTypeSticker:
		// Stickers can be uploaded as photos or documents
		return &vk.Attachment{
			Type: vk.AttachmentTypePhoto,
		}, nil
	case tg.MediaTypeAnimation: // GIF
		if m.videoConversion {
			return &vk.Attachment{
				Type: vk.AttachmentTypeVideo,
			}, nil
		} else {
			return &vk.Attachment{
				Type: vk.AttachmentTypeDoc,
			}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported media type: %s", media.Type)
	}
}
```

## Post Assembly

### Complete Post Transformation
```go
package transformer

import (
	"time"
	
	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
)

type PostTransformer struct {
	textTransformer   *TextTransformer
	mediaTransformer  *MediaTransformer
	config            Config
	logger            *zap.Logger
}

type Config struct {
	PreserveDates      bool
	FromGroup          bool
	AddSourceLink      bool
	SourceLinkFormat   string // e.g., "Source: %s"
	MaxPostsPerBatch   int
	DelayBetweenPosts  time.Duration
}

func NewPostTransformer(config Config) *PostTransformer {
	return &PostTransformer{
		textTransformer:  NewTextTransformer(),
		mediaTransformer: NewMediaTransformer(),
		config:           config,
		logger:           zap.L().Named("transformer"),
	}
}

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
	if tgPost.ForwardedFrom != nil {
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
	
	return fmt.Sprintf("↪️ Forwarded from %s", source)
}

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
```

## Special Cases Handling

### Long Posts
VK has a 10,000 character limit per post. Strategy:
1. Truncate with ellipsis
2. Split into multiple posts (with continuation markers)
3. Move excess text to document attachment

### Multiple Media Items
VK supports up to 10 attachments per post. Strategy:
1. Take first 10 media items
2. Skip the rest with warning
3. Create additional posts for excess media

### Unsupported Media Types
Strategy:
1. Skip with warning (default)
2. Convert to supported format if possible
3. Replace with text description

### Forwarded Messages
Options:
1. Preserve as "Forwarded from [source]" text
2. Try to fetch and include original content
3. Skip forwarding info

### Polls and Interactive Content
Telegram supports polls, quizzes. VK doesn't have direct equivalents:
1. Convert to text description
2. Include poll options as text
3. Skip interactive elements

## Configuration

```yaml
transformer:
  text:
    preserve_formatting: true
    max_length: 10000
    convert_mentions: false
    convert_hashtags: false
    spoiler_format: "🚨 СПОЙЛЕР: %s 🚨"
    
  media:
    skip_unsupported: true
    max_attachments: 10
    video_conversion: true
    gif_to_mp4: true
    
  posts:
    preserve_dates: false
    from_group: true
    add_source_link: true
    source_link_format: "Источник: %s"
    max_posts_per_batch: 100
    skip_failed_posts: true
    delay_between_posts: 3s
    
  forwarding:
    include_forward_info: true
    format: "↪️ Переслано из %s"
    
  splitting:
    split_long_posts: false
    max_post_length: 10000
    continuation_marker: "(продолжение)"
```

## Testing Strategy

### Unit Tests
1. Test individual entity transformations
2. Test text truncation
3. Test media type mapping
4. Test edge cases (empty text, no media, etc.)

### Integration Tests
1. Test with real Telegram post examples
2. Verify VK post creation
3. Test with various media combinations

### Sample Test Cases
```go
func TestTextTransformation(t *testing.T) {
	transformer := NewTextTransformer()
	
	// Test bold text
	text := "Hello world"
	entities := []tg.MessageEntity{
		{Type: "bold", Offset: 0, Length: 5},
	}
	
	result, err := transformer.Transform(text, entities)
	assert.NoError(t, err)
	assert.Equal(t, "<b>Hello</b> world", result)
	
	// Test nested entities
	// etc.
}
```

## Performance Considerations
1. **Batch Processing**: Transform multiple posts in parallel
2. **Caching**: Cache transformed text for identical posts
3. **Memory**: Stream large posts instead of loading all at once
4. **CPU**: Lazy processing of media until needed

## Error Recovery
1. **Partial Success**: Continue with remaining posts if one fails
2. **Fallback Formats**: Use plain text if HTML transformation fails
3. **Warning Collection**: Collect all warnings for user review
4. **Checkpointing**: Save transformation state for resumption

## Next Steps
1. Implement basic text transformer
2. Add media type mapping
3. Create post assembler
4. Add configuration support
5. Write comprehensive tests