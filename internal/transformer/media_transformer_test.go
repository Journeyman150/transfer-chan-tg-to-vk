package transformer

import (
	"testing"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
)

func TestMediaTransformer_Transform_Empty(t *testing.T) {
	mt := NewMediaTransformer()
	attachments, warnings, err := mt.Transform(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected no attachments, got %d", len(attachments))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestMediaTransformer_Transform_Photo(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypePhoto, FileID: "photo1"},
	}
	attachments, warnings, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Type != vk.AttachmentTypePhoto {
		t.Errorf("expected photo attachment, got %v", attachments[0].Type)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestMediaTransformer_Transform_Video(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeVideo, FileID: "video1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Type != vk.AttachmentTypeVideo {
		t.Errorf("expected video attachment, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Audio(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeAudio, FileID: "audio1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Type != vk.AttachmentTypeAudio {
		t.Errorf("expected audio attachment, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Voice(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeVoice, FileID: "voice1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Type != vk.AttachmentTypeAudio {
		t.Errorf("expected audio attachment for voice, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Sticker(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeSticker, FileID: "sticker1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].Type != vk.AttachmentTypePhoto {
		t.Errorf("expected photo attachment for sticker, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Animation(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeAnimation, FileID: "gif1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// By default gifToMP4 is false, so should be doc
	if attachments[0].Type != vk.AttachmentTypeDoc {
		t.Errorf("expected doc attachment for animation (default), got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Animation_WithConversion(t *testing.T) {
	mt := NewMediaTransformer().WithGifToMP4(true)
	media := []tg.Media{
		{Type: tg.MediaTypeAnimation, FileID: "gif1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachments[0].Type != vk.AttachmentTypeVideo {
		t.Errorf("expected video attachment for animation with conversion, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Document_Image(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeDocument, MimeType: "image/jpeg", FileID: "doc1"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachments[0].Type != vk.AttachmentTypePhoto {
		t.Errorf("expected photo attachment for image document, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Document_Video(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeDocument, MimeType: "video/mp4", FileID: "doc2"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachments[0].Type != vk.AttachmentTypeVideo {
		t.Errorf("expected video attachment for video document, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Document_Audio(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeDocument, MimeType: "audio/mpeg", FileID: "doc3"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachments[0].Type != vk.AttachmentTypeAudio {
		t.Errorf("expected audio attachment for audio document, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_Document_Generic(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypeDocument, MimeType: "application/pdf", FileID: "doc4"},
	}
	attachments, _, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachments[0].Type != vk.AttachmentTypeDoc {
		t.Errorf("expected doc attachment for generic document, got %v", attachments[0].Type)
	}
}

func TestMediaTransformer_Transform_MaxAttachments(t *testing.T) {
	mt := NewMediaTransformer().WithMaxAttachments(2)
	media := []tg.Media{
		{Type: tg.MediaTypePhoto, FileID: "p1"},
		{Type: tg.MediaTypePhoto, FileID: "p2"},
		{Type: tg.MediaTypePhoto, FileID: "p3"},
	}
	attachments, warnings, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments due to limit, got %d", len(attachments))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning about limit, got %d", len(warnings))
	}
	expectedWarning := "photo: attachment limit exceeded (max 2)"
	if warnings[0] != expectedWarning {
		t.Errorf("expected warning %q, got %q", expectedWarning, warnings[0])
	}
}

func TestMediaTransformer_Transform_UnsupportedType(t *testing.T) {
	mt := NewMediaTransformer()
	// Create a media with unknown type (not in enum)
	media := []tg.Media{
		{Type: tg.MediaType("unknown"), FileID: "unknown1"},
	}
	attachments, warnings, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected no attachments for unsupported type, got %d", len(attachments))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
	expectedWarningPrefix := "unknown: unsupported media type: unknown"
	if !containsPrefix(warnings[0], expectedWarningPrefix) {
		t.Errorf("expected warning starting with %q, got %q", expectedWarningPrefix, warnings[0])
	}
}

func TestMediaTransformer_Transform_UnsupportedType_NoSkip(t *testing.T) {
	mt := NewMediaTransformer().WithSkipUnsupported(false)
	media := []tg.Media{
		{Type: tg.MediaType("unknown"), FileID: "unknown1"},
	}
	attachments, warnings, err := mt.Transform(media)
	if err == nil {
		t.Fatal("expected error for unsupported type when skipUnsupported is false")
	}
	if len(attachments) != 0 {
		t.Errorf("expected no attachments, got %d", len(attachments))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestMediaTransformer_Transform_Mixed(t *testing.T) {
	mt := NewMediaTransformer()
	media := []tg.Media{
		{Type: tg.MediaTypePhoto, FileID: "p1"},
		{Type: tg.MediaTypeVideo, FileID: "v1"},
		{Type: tg.MediaTypeAudio, FileID: "a1"},
		{Type: tg.MediaTypeDocument, MimeType: "image/png", FileID: "d1"},
	}
	attachments, warnings, err := mt.Transform(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 4 {
		t.Errorf("expected 4 attachments, got %d", len(attachments))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	// Check order preserved
	expectedTypes := []vk.AttachmentType{
		vk.AttachmentTypePhoto,
		vk.AttachmentTypeVideo,
		vk.AttachmentTypeAudio,
		vk.AttachmentTypePhoto,
	}
	for i, att := range attachments {
		if att.Type != expectedTypes[i] {
			t.Errorf("attachment %d: expected %v, got %v", i, expectedTypes[i], att.Type)
		}
	}
}

// Helper function to check if string contains prefix
func containsPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}