package tts

import (
	"strings"
	"testing"
)

func TestSentenceParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []string // Expected sentence texts
		wantErr  bool
	}{
		{
			name:     "Simple sentences",
			markdown: "This is a sentence. This is another sentence! And a third one?",
			want:     []string{"This is a sentence.", "This is another sentence!", "And a third one?"},
		},
		{
			name:     "Markdown with headers",
			markdown: "# Title\n\nThis is a paragraph. It has two sentences.",
			want:     []string{"Title.", "This is a paragraph.", "It has two sentences."},
		},
		{
			name:     "Markdown with lists",
			markdown: "Items:\n- First item\n- Second item\n- Third item",
			want:     []string{"Items:", "First item.", "Second item.", "Third item."},
		},
		{
			name:     "Markdown with code blocks",
			markdown: "Here is some text.\n\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\nMore text here.",
			want:     []string{"Here is some text.", "More text here."},
		},
		{
			name:     "Inline code",
			markdown: "Use the `fmt.Println()` function to print.",
			want:     []string{"Use the `fmt.Println()` function to print."},
		},
		{
			name:     "Links",
			markdown: "Visit [Google](https://google.com) for more info.",
			want:     []string{"Visit Google for more info."},
		},
		{
			name:     "Images",
			markdown: "![Alt text](image.png) This is an image.",
			want:     []string{"[Image: Alt text] This is an image."},
		},
		{
			name:     "Bold and italic",
			markdown: "This is **bold** and this is *italic* text.",
			want:     []string{"This is bold and this is italic text."},
		},
		{
			name:     "Blockquotes",
			markdown: "> This is a quote.\n> It spans multiple lines.",
			want:     []string{"Quote: This is a quote.", "It spans multiple lines."},
		},
		{
			name:     "Empty markdown",
			markdown: "",
			want:     []string{},
		},
		{
			name:     "Only whitespace",
			markdown: "   \n\n   \t   ",
			want:     []string{},
		},
	}

	parser := NewSentenceParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.markdown)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Parse() returned %d sentences, want %d", len(got), len(tt.want))
				for i, s := range got {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}

			for i, want := range tt.want {
				if got[i].Text != want {
					t.Errorf("Parse() sentence[%d] = %q, want %q", i, got[i].Text, want)
				}
			}
		})
	}
}

func TestSentenceParser_Abbreviations(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		want     []string
	}{
		{
			name: "Common abbreviations",
			text: "Dr. Smith and Mr. Jones met at 3 p.m. yesterday.",
			want: []string{"Dr. Smith and Mr. Jones met at 3 p.m. yesterday."},
		},
		{
			name: "Technical abbreviations",
			text: "The API uses HTTP. The SDK supports multiple languages.",
			want: []string{"The API uses HTTP.", "The SDK supports multiple languages."},
		},
		{
			name: "File extensions",
			text: "Edit the config.yml file. Then run main.go to start.",
			want: []string{"Edit the config.yml file.", "Then run main.go to start."},
		},
		{
			name: "Mixed abbreviations",
			text: "Prof. Johnson has a Ph.D. in computer science. She teaches at MIT.",
			want: []string{"Prof. Johnson has a Ph.D. in computer science.", "She teaches at MIT."},
		},
		{
			name: "Units of measurement",
			text: "The distance is 5 km. The weight is 10 kg.",
			want: []string{"The distance is 5 km.", "The weight is 10 kg."},
		},
	}

	parser := NewSentenceParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, _ := parser.Parse(tt.text)

			if len(sentences) != len(tt.want) {
				t.Errorf("Expected %d sentences, got %d", len(tt.want), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}

			for i, want := range tt.want {
				if sentences[i].Text != want {
					t.Errorf("Sentence[%d] = %q, want %q", i, sentences[i].Text, want)
				}
			}
		})
	}
}

func TestSentenceParser_Numbers(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Decimal numbers",
			text: "The value is 3.14. The price is $19.99.",
			want: []string{"The value is 3.14.", "The price is $19.99."},
		},
		{
			name: "Version numbers",
			text: "Version 2.0.1 is released. Update to v3.0 soon.",
			want: []string{"Version 2.0.1 is released.", "Update to v3.0 soon."},
		},
		{
			name: "Mixed numbers",
			text: "The temperature is 98.6 degrees. That's normal.",
			want: []string{"The temperature is 98.6 degrees.", "That's normal."},
		},
	}

	parser := NewSentenceParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, _ := parser.Parse(tt.text)

			if len(sentences) != len(tt.want) {
				t.Errorf("Expected %d sentences, got %d", len(tt.want), len(sentences))
				return
			}

			for i, want := range tt.want {
				if sentences[i].Text != want {
					t.Errorf("Sentence[%d] = %q, want %q", i, sentences[i].Text, want)
				}
			}
		})
	}
}

func TestSentenceParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Ellipsis",
			text: "Wait... I'm thinking. Let me see...",
			want: []string{"Wait... I'm thinking.", "Let me see..."},
		},
		{
			name: "URLs",
			text: "Visit https://example.com for info. Check the docs.",
			want: []string{"Visit https://example.com for info.", "Check the docs."},
		},
		{
			name: "Email addresses",
			text: "Contact user@example.com for help. We'll respond soon.",
			want: []string{"Contact user@example.com for help.", "We'll respond soon."},
		},
		{
			name: "Mixed punctuation",
			text: "Really?! That's amazing! Wow!!!",
			want: []string{"Really?!", "That's amazing!", "Wow!!!"},
		},
		{
			name: "Quotes",
			text: `He said "Hello." She replied "Hi there."`,
			want: []string{`He said "Hello."`, `She replied "Hi there."`},
		},
	}

	parser := NewSentenceParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, _ := parser.Parse(tt.text)

			if len(sentences) != len(tt.want) {
				t.Errorf("Expected %d sentences, got %d", len(tt.want), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}

			for i, want := range tt.want {
				if sentences[i].Text != want {
					t.Errorf("Sentence[%d] = %q, want %q", i, sentences[i].Text, want)
				}
			}
		})
	}
}

func TestSentenceParser_StripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "Headers",
			markdown: "# Title\n## Subtitle",
			want:     "Title. Subtitle.",
		},
		{
			name:     "Bold and italic",
			markdown: "**bold** and *italic*",
			want:     "bold and italic",
		},
		{
			name:     "Links",
			markdown: "[Link text](https://example.com)",
			want:     "Link text",
		},
		{
			name:     "Code blocks removed",
			markdown: "Text before\n```\ncode here\n```\nText after",
			want:     "Text before Text after",
		},
		{
			name:     "Lists",
			markdown: "- Item 1\n- Item 2",
			want:     "Item 1. Item 2.",
		},
	}

	parser := NewSentenceParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.StripMarkdown(tt.markdown)
			got = strings.TrimSpace(got)
			want := strings.TrimSpace(tt.want)

			if got != want {
				t.Errorf("StripMarkdown() = %q, want %q", got, want)
			}
		})
	}
}

func TestSentenceParser_PositionMapping(t *testing.T) {
	parser := NewSentenceParser()
	markdown := "First sentence. Second sentence. Third sentence."
	
	sentences, err := parser.Parse(markdown)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(sentences) != 3 {
		t.Fatalf("Expected 3 sentences, got %d", len(sentences))
	}

	// Check position indices
	for i, s := range sentences {
		if s.Position != i {
			t.Errorf("Sentence[%d].Position = %d, want %d", i, s.Position, i)
		}
	}

	// Check IDs are unique
	ids := make(map[string]bool)
	for _, s := range sentences {
		if ids[s.ID] {
			t.Errorf("Duplicate sentence ID: %s", s.ID)
		}
		ids[s.ID] = true
	}
}

func TestSentenceParser_LengthLimits(t *testing.T) {
	// Test minimum length filtering
	parser := NewSentenceParserWithOptions(WithMinLength(10))
	
	sentences, _ := parser.Parse("Hi. This is a longer sentence.")
	
	// "Hi." should be filtered out as it's less than 10 characters
	if len(sentences) != 1 {
		t.Errorf("Expected 1 sentence after min length filtering, got %d", len(sentences))
	}
	
	// Test maximum length truncation
	parser = NewSentenceParserWithOptions(WithMaxLength(20))
	
	longText := "This is a very long sentence that exceeds the maximum length limit."
	sentences, _ = parser.Parse(longText)
	
	if len(sentences) != 1 {
		t.Fatalf("Expected 1 sentence, got %d", len(sentences))
	}
	
	if len(sentences[0].Text) > 20 {
		t.Errorf("Sentence not truncated: length = %d, max = 20", len(sentences[0].Text))
	}
}

func TestSentenceParser_CodeBlockOption(t *testing.T) {
	markdown := "Text before.\n```go\ncode here\n```\nText after."
	
	// Test with code blocks skipped (default)
	parser := NewSentenceParser()
	sentences, _ := parser.Parse(markdown)
	
	if len(sentences) != 2 {
		t.Errorf("Expected 2 sentences with code skipped, got %d", len(sentences))
	}
	
	// Test with code blocks included
	parser = NewSentenceParserWithOptions(WithCodeBlocks(true))
	sentences, _ = parser.Parse(markdown)
	
	// Should have "Text before.", "[Code block omitted]", "Text after."
	hasCodeMarker := false
	for _, s := range sentences {
		if strings.Contains(s.Text, "Code block omitted") {
			hasCodeMarker = true
			break
		}
	}
	
	if !hasCodeMarker {
		t.Error("Expected code block marker when including code blocks")
	}
}

func TestSentenceParser_ComplexMarkdown(t *testing.T) {
	markdown := `# Glow TTS Documentation

## Introduction

Glow TTS adds **text-to-speech** capabilities to the popular [Glow](https://github.com/charmbracelet/glow) markdown reader.

### Features

- Offline TTS with Piper
- Online TTS with Google Cloud
- Smart sentence parsing

## Code Example

Here's how to use it:

` + "```bash" + `
glow --tts piper document.md
` + "```" + `

> Note: This requires Piper to be installed.

For more info, visit https://example.com or contact support@example.com.

Version 1.0.0 released!`

	parser := NewSentenceParser()
	sentences, err := parser.Parse(markdown)
	
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	
	// Should parse multiple sentences from the complex markdown
	if len(sentences) < 5 {
		t.Errorf("Expected at least 5 sentences from complex markdown, got %d", len(sentences))
		for i, s := range sentences {
			t.Logf("  [%d]: %q", i, s.Text)
		}
	}
	
	// Check that code block was skipped
	for _, s := range sentences {
		if strings.Contains(s.Text, "glow --tts") {
			t.Error("Code block content should be skipped")
		}
	}
	
	// Check that quote was included
	hasQuote := false
	for _, s := range sentences {
		if strings.Contains(s.Text, "Quote:") {
			hasQuote = true
			break
		}
	}
	if !hasQuote {
		t.Error("Expected quote to be included with 'Quote:' prefix")
	}
}

// Benchmark tests
func BenchmarkSentenceParser_Parse(b *testing.B) {
	parser := NewSentenceParser()
	markdown := strings.Repeat("This is a test sentence. ", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(markdown)
	}
}

func BenchmarkSentenceParser_LargeDocument(b *testing.B) {
	parser := NewSentenceParser()
	// Simulate a large markdown document
	sections := []string{
		"# Title\n\n",
		"## Introduction\n\n",
		strings.Repeat("This is a paragraph with multiple sentences. Each sentence should be parsed correctly. ", 50),
		"\n\n```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```\n\n",
		strings.Repeat("Another paragraph here. With more content. ", 50),
	}
	markdown := strings.Join(sections, "")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(markdown)
	}
}