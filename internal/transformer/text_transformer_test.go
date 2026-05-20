package transformer

import (
	"strings"
	"testing"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTextTransformer_Transform_EmptyText(t *testing.T) {
	tt := NewTextTransformer()
	result, err := tt.Transform("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTextTransformer_Transform_NoEntities(t *testing.T) {
	tt := NewTextTransformer()
	text := "Hello, world!"
	result, err := tt.Transform(text, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}

func TestTextTransformer_Transform_Bold(t *testing.T) {
	tt := NewTextTransformer()
	text := "Hello, world!"
	entities := []tgbotapi.MessageEntity{
		{Type: "bold", Offset: 0, Length: 5},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "<b>Hello</b>, world!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_Italic(t *testing.T) {
	tt := NewTextTransformer()
	text := "Hello, world!"
	entities := []tgbotapi.MessageEntity{
		{Type: "italic", Offset: 7, Length: 5},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Hello, <i>world</i>!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_Code(t *testing.T) {
	tt := NewTextTransformer()
	text := "Use `print()` function"
	entities := []tgbotapi.MessageEntity{
		{Type: "code", Offset: 4, Length: 7},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Use <code>`print(</code>)` function"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_TextLink(t *testing.T) {
	tt := NewTextTransformer()
	text := "Visit Google"
	entities := []tgbotapi.MessageEntity{
		{Type: "text_link", Offset: 6, Length: 6, URL: "https://google.com"},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Visit <a href=\"https://google.com\">Google</a>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_MultipleEntities(t *testing.T) {
	tt := NewTextTransformer()
	text := "Hello, world! This is a test."
	entities := []tgbotapi.MessageEntity{
		{Type: "bold", Offset: 0, Length: 5},
		{Type: "italic", Offset: 7, Length: 5},
		{Type: "code", Offset: 19, Length: 4},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "<b>Hello</b>, <i>world</i>! This <code>is a</code> test."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_NestedEntities(t *testing.T) {
	tt := NewTextTransformer()
	text := "Hello, world!"
	entities := []tgbotapi.MessageEntity{
		{Type: "bold", Offset: 0, Length: 13},
		{Type: "italic", Offset: 7, Length: 5},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce <b>Hello, world!</b> (nested entities not supported)
	expected := "<b>Hello, world!</b>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_Truncation(t *testing.T) {
	tt := NewTextTransformer()
	tt.maxLength = 20
	text := "This is a very long text that exceeds the limit."
	entities := []tgbotapi.MessageEntity{
		{Type: "bold", Offset: 0, Length: 4},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be truncated to 20 characters (including ellipsis)
	if utf8.RuneCountInString(result) > 20 {
		t.Errorf("result length %d exceeds max length %d", utf8.RuneCountInString(result), tt.maxLength)
	}
	// Should contain ellipsis
	if !strings.Contains(result, "...") {
		t.Errorf("truncated text should contain ellipsis, got %q", result)
	}
}

func TestTextTransformer_Transform_Spoiler(t *testing.T) {
	tt := NewTextTransformer()
	text := "The killer is John."
	entities := []tgbotapi.MessageEntity{
		{Type: "spoiler", Offset: 14, Length: 4},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "The killer is 🚨 СПОЙЛЕР: John 🚨."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_Strikethrough(t *testing.T) {
	tt := NewTextTransformer()
	text := "This is wrong."
	entities := []tgbotapi.MessageEntity{
		{Type: "strikethrough", Offset: 8, Length: 5},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "This is <s>wrong</s>."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTextTransformer_Transform_Underline(t *testing.T) {
	tt := NewTextTransformer()
	text := "Important note"
	entities := []tgbotapi.MessageEntity{
		{Type: "underline", Offset: 0, Length: 9},
	}
	result, err := tt.Transform(text, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "<u>Important</u> note"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}