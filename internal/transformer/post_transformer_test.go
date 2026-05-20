package transformer

import (
	"testing"
	"time"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
)

func TestPostTransformer_TransformPost_EmptyPost(t *testing.T) {
	pt := NewPostTransformer(Config{})
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "",
	}
	vkPost, warnings, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if vkPost.Text != "" {
		t.Errorf("expected empty text, got %q", vkPost.Text)
	}
	if len(vkPost.Attachments) != 0 {
		t.Errorf("expected no attachments, got %v", vkPost.Attachments)
	}
}

func TestPostTransformer_TransformPost_TextOnly(t *testing.T) {
	pt := NewPostTransformer(Config{})
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "Hello world",
	}
	vkPost, warnings, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if vkPost.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", vkPost.Text)
	}
}

func TestPostTransformer_TransformPost_WithSourceLink(t *testing.T) {
	pt := NewPostTransformer(Config{
		AddSourceLink:    true,
		SourceLinkFormat: "Source: %s",
	})
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "Hello",
		Link: "https://t.me/channel/123",
	}
	vkPost, warnings, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	expected := "Hello\n\nSource: https://t.me/channel/123"
	if vkPost.Text != expected {
		t.Errorf("expected text %q, got %q", expected, vkPost.Text)
	}
}

func TestPostTransformer_TransformPost_WithForwardInfo(t *testing.T) {
	pt := NewPostTransformer(Config{
		IncludeForwardInfo: true,
		ForwardFormat:      "Forwarded from %s",
	})
	forwardTime := time.Now().Add(-time.Hour)
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "Hello",
		ForwardedFrom: &tg.ForwardInfo{
			FromChatID:   123,
			FromChatName: "Test Channel",
			MessageID:    456,
			Date:         forwardTime,
		},
	}
	vkPost, warnings, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	expected := "Forwarded from Test Channel\n\nHello"
	if vkPost.Text != expected {
		t.Errorf("expected text %q, got %q", expected, vkPost.Text)
	}
}

func TestPostTransformer_TransformPost_WithMedia(t *testing.T) {
	pt := NewPostTransformer(Config{})
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "Photo",
		Media: []tg.Media{
			{
				Type:   tg.MediaTypePhoto,
				FileID: "photo1",
			},
		},
	}
	vkPost, warnings, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Media transformer returns stub attachments without owner/media IDs
	if len(vkPost.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(vkPost.Attachments))
	}
	if vkPost.Attachments[0].Type != vk.AttachmentTypePhoto {
		t.Errorf("expected photo attachment, got %v", vkPost.Attachments[0].Type)
	}
	// warnings about skipped media? none
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestPostTransformer_TransformPost_PreserveDates(t *testing.T) {
	postTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	pt := NewPostTransformer(Config{
		PreserveDates: true,
	})
	post := tg.Post{
		ID:   1,
		Date: postTime,
		Text: "Hello",
	}
	vkPost, _, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vkPost.PublishDate == nil {
		t.Fatal("expected publish date to be set")
	}
	if !vkPost.PublishDate.Equal(postTime) {
		t.Errorf("expected publish date %v, got %v", postTime, vkPost.PublishDate)
	}
}

func TestPostTransformer_TransformPost_FromGroup(t *testing.T) {
	pt := NewPostTransformer(Config{
		FromGroup: true,
	})
	post := tg.Post{
		ID:   1,
		Date: time.Now(),
		Text: "Hello",
	}
	vkPost, _, err := pt.TransformPost(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !vkPost.FromGroup {
		t.Error("expected FromGroup to be true")
	}
	if !vkPost.Signed {
		t.Error("expected Signed to be true")
	}
}

func TestPostTransformer_TransformPosts_BatchLimit(t *testing.T) {
	pt := NewPostTransformer(Config{
		MaxPostsPerBatch: 2,
	})
	posts := []tg.Post{
		{ID: 1, Date: time.Now(), Text: "First"},
		{ID: 2, Date: time.Now(), Text: "Second"},
		{ID: 3, Date: time.Now(), Text: "Third"},
	}
	vkPosts, warnings, err := pt.TransformPosts(posts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vkPosts) != 2 {
		t.Errorf("expected 2 posts due to batch limit, got %d", len(vkPosts))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestPostTransformer_TransformPosts_SkipFailedPosts(t *testing.T) {
	pt := NewPostTransformer(Config{
		SkipFailedPosts: true,
	})
	// We cannot easily cause a transformation failure without mocking.
	// This test is a placeholder; we'll rely on integration tests.
	// For now, just ensure it doesn't panic.
	posts := []tg.Post{
		{ID: 1, Date: time.Now(), Text: "OK"},
	}
	_, _, err := pt.TransformPosts(posts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}