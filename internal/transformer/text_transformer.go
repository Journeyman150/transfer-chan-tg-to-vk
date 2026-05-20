package transformer

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TextTransformer converts Telegram formatted text with entities to VK HTML.
type TextTransformer struct {
	// Configuration
	preserveFormatting bool
	maxLength          int // VK limit: 10000 characters
	convertMentions    bool
	convertHashtags    bool
	spoilerFormat      string // e.g., "[SPOILER]%s[/SPOILER]"
}

// NewTextTransformer creates a new TextTransformer with default settings.
func NewTextTransformer() *TextTransformer {
	return &TextTransformer{
		preserveFormatting: true,
		maxLength:          10000,
		convertMentions:    false, // VK doesn't have @mentions
		convertHashtags:    false, // Keep as plain text
		spoilerFormat:      "🚨 СПОЙЛЕР: %s 🚨",
	}
}

// Transform converts Telegram text with entities to VK HTML.
// If entities is nil or empty, returns the original text (after truncation).
func (t *TextTransformer) Transform(text string, entities []tgbotapi.MessageEntity) (string, error) {
	if text == "" {
		return "", nil
	}

	// Convert Telegram entities to VK HTML
	result := t.applyEntities(text, entities)

	// Truncate if too long
	result = t.truncateText(result)

	// Clean up any invalid HTML (simple implementation)
	result = t.cleanHTML(result)

	return result, nil
}

// applyEntities applies Telegram entities to text, producing HTML.
func (t *TextTransformer) applyEntities(text string, entities []tgbotapi.MessageEntity) string {
	if len(entities) == 0 {
		return text
	}

	// Sort entities by offset (ascending) and length (descending)
	// to handle nested entities properly
	sortedEntities := make([]tgbotapi.MessageEntity, len(entities))
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
		// Skip if entity is completely before lastPos (should not happen due to sorting)
		if entity.Offset+entity.Length <= lastPos {
			continue
		}

		// Append text between lastPos and entity start
		if entity.Offset > lastPos {
			result.WriteString(string(runes[lastPos:entity.Offset]))
		}

		// Calculate end position
		endPos := entity.Offset + entity.Length
		if endPos > len(runes) {
			endPos = len(runes)
		}

		// If entity starts before lastPos (overlap), adjust start
		startPos := entity.Offset
		if startPos < lastPos {
			startPos = lastPos
		}
		if startPos >= endPos {
			// No text left for this entity
			lastPos = endPos
			continue
		}

		// Extract entity text
		entityText := string(runes[startPos:endPos])

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

// formatEntity wraps entity text in appropriate HTML tags.
func (t *TextTransformer) formatEntity(text string, entity tgbotapi.MessageEntity) string {
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
		if entity.URL == "" {
			return text
		}
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

// truncateText ensures text does not exceed maxLength characters.
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

// cleanHTML performs basic HTML sanitization (placeholder).
func (t *TextTransformer) cleanHTML(text string) string {
	// For now, just return the text.
	// In a real implementation, we would ensure tags are properly closed,
	// escape unsafe characters, etc.
	return text
}

// fixBrokenHTML attempts to close any unclosed HTML tags in truncated text.
func (t *TextTransformer) fixBrokenHTML(text string) string {
	// Simple implementation: if we detect an opening tag without closing,
	// we append the closing tag.
	// This is a naive approach; a robust solution would require a proper parser.
	// For now, we just return the text as-is.
	return text
}