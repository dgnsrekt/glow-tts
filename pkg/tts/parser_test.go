package tts

import (
	"strings"
	"testing"
)

func TestNewSentenceParser(t *testing.T) {
	t.Run("creates parser with default config", func(t *testing.T) {
		parser, err := NewSentenceParser(nil)
		if err != nil {
			t.Fatalf("Failed to create parser: %v", err)
		}
		if parser == nil {
			t.Fatal("Expected parser to be created")
		}
		if parser.config == nil {
			t.Fatal("Expected parser to have config")
		}
	})

	t.Run("creates parser with custom config", func(t *testing.T) {
		config := &ParserConfig{
			IncludeCodeBlocks: true,
			ExpandLinks:       false,
			MinSentenceLength: 5,
			MaxSentenceLength: 100,
		}
		parser, err := NewSentenceParser(config)
		if err != nil {
			t.Fatalf("Failed to create parser: %v", err)
		}
		if parser.config.IncludeCodeBlocks != true {
			t.Error("Expected IncludeCodeBlocks to be true")
		}
		if parser.config.MinSentenceLength != 5 {
			t.Error("Expected MinSentenceLength to be 5")
		}
	})
}

func TestParseSentences(t *testing.T) {
	parser, err := NewSentenceParser(nil)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple sentence",
			input:    "This is a simple sentence.",
			expected: 1,
		},
		{
			name:     "multiple sentences",
			input:    "First sentence. Second sentence! Third sentence?",
			expected: 3,
		},
		{
			name:     "sentence with abbreviation",
			input:    "Dr. Smith works at Inc. Corporation.",
			expected: 1,
		},
		{
			name:     "sentence with decimal numbers",
			input:    "The price is 3.14 dollars.",
			expected: 1,
		},
		{
			name:     "sentence with ellipsis",
			input:    "Wait for it... and then continue.",
			expected: 1,
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
		{
			name:     "whitespace only",
			input:    "   \n\t  ",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(sentences) != tt.expected {
				t.Errorf("Expected %d sentences, got %d", tt.expected, len(sentences))
				for i, s := range sentences {
					t.Logf("Sentence %d: %q", i, s.Text)
				}
			}
		})
	}
}

func TestMarkdownParsing(t *testing.T) {
	parser, err := NewSentenceParser(nil)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "headers",
			input: `# Main Title
## Subtitle
Regular paragraph here.`,
			contains: []string{"Main Title", "Subtitle", "Regular paragraph here"},
		},
		{
			name: "lists",
			input: `- First item
- Second item
- Third item`,
			contains: []string{"First item", "Second item", "Third item"},
		},
		{
			name: "links",
			input: `Check out [this link](https://example.com) for more info.`,
			contains: []string{"Check out", "link", "for more info"},
		},
		{
			name: "blockquote",
			input: `> This is a quote
> that spans multiple lines`,
			contains: []string{"This is a quote", "that spans multiple lines"},
		},
		{
			name: "emphasis",
			input: `This is *italic* and this is **bold** text.`,
			contains: []string{"This is", "italic", "and this is", "bold", "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			
			allText := ""
			for _, s := range sentences {
				allText += " " + s.Text
			}
			
			for _, expected := range tt.contains {
				if !strings.Contains(allText, expected) {
					t.Errorf("Expected to find %q in parsed text, but didn't. Got: %q", expected, allText)
				}
			}
		})
	}
}

func TestLongSentenceSplitting(t *testing.T) {
	config := &ParserConfig{
		MinSentenceLength: 3,
		MaxSentenceLength: 50,
	}
	parser, err := NewSentenceParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	longText := "This is a very long sentence that contains many words and should be split into multiple parts because it exceeds the maximum sentence length configured in the parser settings."
	
	sentences, err := parser.Parse(longText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if len(sentences) < 2 {
		t.Errorf("Expected long sentence to be split, got %d sentences", len(sentences))
	}
	
	for i, s := range sentences {
		if len(s.Text) > config.MaxSentenceLength {
			t.Errorf("Sentence %d exceeds max length: %d > %d", i, len(s.Text), config.MaxSentenceLength)
		}
	}
}

func TestSpecialPatternProtection(t *testing.T) {
	parser, err := NewSentenceParser(nil)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "decimal protection",
			input:    "The value is 3.14159",
			expected: "The value is 3.14159",
		},
		{
			name:     "abbreviation protection",
			input:    "Contact Dr. Smith",
			expected: "Contact Dr. Smith",
		},
		{
			name:     "ellipsis protection",
			input:    "Wait for it...",
			expected: "Wait for it...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protected := parser.protectSpecialPatterns(tt.input)
			restored := parser.restoreSpecialPatterns(protected)
			
			if restored != tt.expected {
				t.Errorf("Pattern protection/restoration failed. Expected %q, got %q", tt.expected, restored)
			}
		})
	}
}

func TestMarkdownProcessor(t *testing.T) {
	config := DefaultParserConfig()
	processor := NewMarkdownProcessor(config)

	t.Run("process markdown elements", func(t *testing.T) {
		markdown := `# Title

This is a paragraph with **bold** and *italic* text.

## Subtitle

- List item 1
- List item 2

> A blockquote

[A link](https://example.com)
`

		elements, err := processor.ProcessMarkdown(markdown)
		if err != nil {
			t.Fatalf("Failed to process markdown: %v", err)
		}

		if len(elements) == 0 {
			t.Error("Expected to extract markdown elements")
		}

		// Check for different element types
		hasHeading := false
		hasParagraph := false
		hasList := false
		hasBlockquote := false
		hasLink := false

		for _, elem := range elements {
			switch elem.Type {
			case ElementHeading:
				hasHeading = true
			case ElementParagraph:
				hasParagraph = true
			case ElementListItem:
				hasList = true
			case ElementBlockquote:
				hasBlockquote = true
			case ElementLink:
				hasLink = true
			}
		}

		if !hasHeading {
			t.Error("Expected to find heading element")
		}
		if !hasParagraph {
			t.Error("Expected to find paragraph element")
		}
		if !hasList {
			t.Error("Expected to find list element")
		}
		if !hasBlockquote {
			t.Error("Expected to find blockquote element")
		}
		if !hasLink {
			t.Error("Expected to find link element")
		}
	})

	t.Run("convert to speech", func(t *testing.T) {
		markdown := `# Welcome

This is a test document.

## Features

- First feature
- Second feature

Visit [our website](https://example.com) for more.`

		elements, err := processor.ProcessMarkdown(markdown)
		if err != nil {
			t.Fatalf("Failed to process markdown: %v", err)
		}

		speeches := processor.ConvertToSpeech(elements)
		if len(speeches) == 0 {
			t.Error("Expected to generate speech text")
		}

		allSpeech := strings.Join(speeches, " ")
		
		// Check for expected content
		expectedPhrases := []string{
			"Welcome",
			"test document",
			"Features",
			"First feature",
			"Second feature",
		}

		for _, phrase := range expectedPhrases {
			if !strings.Contains(allSpeech, phrase) {
				t.Errorf("Expected speech to contain %q, got: %q", phrase, allSpeech)
			}
		}
	})

	t.Run("extract speakable sentences", func(t *testing.T) {
		markdown := `# Introduction

This is the first sentence. This is the second sentence! And here's the third?

## Details

Here are more details about the topic.`

		sentences, err := processor.ExtractSpeakableSentences(markdown)
		if err != nil {
			t.Fatalf("Failed to extract sentences: %v", err)
		}

		if len(sentences) < 4 {
			t.Errorf("Expected at least 4 sentences, got %d", len(sentences))
		}

		// Check that sentences have positions
		for i, s := range sentences {
			if s.Position != i {
				t.Errorf("Sentence %d has incorrect position %d", i, s.Position)
			}
		}
	})
}

func TestCleanTextForSpeech(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove multiple spaces",
			input:    "Too    many     spaces",
			expected: "Too many spaces",
		},
		{
			name:     "remove URLs in parentheses",
			input:    "Check this (https://example.com) out",
			expected: "Check this out",
		},
		{
			name:     "convert symbols",
			input:    "A & B @ home",
			expected: "A and B at home",
		},
		{
			name:     "clean punctuation",
			input:    "Really??? Yes!!!",
			expected: "Really? Yes!",
		},
		{
			name:     "spacing around punctuation",
			input:    "Hello , world . How are you ?",
			expected: "Hello, world. How are you?",
		},
		{
			name:     "programming symbols",
			input:    "if x >= 5 && y != 0",
			expected: "if x greater than or equal to 5 and y not equals 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanTextForSpeech(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCodeBlockHandling(t *testing.T) {
	t.Run("exclude code blocks", func(t *testing.T) {
		config := &ParserConfig{
			IncludeCodeBlocks: false,
		}
		processor := NewMarkdownProcessor(config)
		
		markdown := "Text before\n```go\nfunc main() {}\n```\nText after"
		
		sentences, err := processor.ExtractSpeakableSentences(markdown)
		if err != nil {
			t.Fatalf("Failed to extract sentences: %v", err)
		}
		
		allText := ""
		for _, s := range sentences {
			allText += s.Text + " "
		}
		
		if strings.Contains(allText, "func main") {
			t.Error("Code block should be excluded but was found in output")
		}
	})

	t.Run("include code blocks", func(t *testing.T) {
		config := &ParserConfig{
			IncludeCodeBlocks: true,
		}
		processor := NewMarkdownProcessor(config)
		
		markdown := "Text before\n```go\nfunc main() {}\n```\nText after"
		
		elements, err := processor.ProcessMarkdown(markdown)
		if err != nil {
			t.Fatalf("Failed to process markdown: %v", err)
		}
		
		hasCodeBlock := false
		for _, elem := range elements {
			if elem.Type == ElementCodeBlock {
				hasCodeBlock = true
				if elem.Language != "go" {
					t.Errorf("Expected language 'go', got %q", elem.Language)
				}
			}
		}
		
		if !hasCodeBlock {
			t.Error("Code block should be included but was not found")
		}
	})
}

func TestLinkExpansion(t *testing.T) {
	t.Run("expand links", func(t *testing.T) {
		config := &ParserConfig{
			ExpandLinks: true,
		}
		processor := NewMarkdownProcessor(config)
		
		markdown := "Check [this link](https://example.com)"
		
		elements, err := processor.ProcessMarkdown(markdown)
		if err != nil {
			t.Fatalf("Failed to process markdown: %v", err)
		}
		
		speeches := processor.ConvertToSpeech(elements)
		allSpeech := strings.Join(speeches, " ")
		
		if !strings.Contains(allSpeech, "link to") {
			t.Error("Expected expanded link format with 'link to' prefix")
		}
	})

	t.Run("don't expand links", func(t *testing.T) {
		config := &ParserConfig{
			ExpandLinks: false,
		}
		processor := NewMarkdownProcessor(config)
		
		markdown := "Check [this link](https://example.com)"
		
		elements, err := processor.ProcessMarkdown(markdown)
		if err != nil {
			t.Fatalf("Failed to process markdown: %v", err)
		}
		
		speeches := processor.ConvertToSpeech(elements)
		allSpeech := strings.Join(speeches, " ")
		
		if strings.Contains(allSpeech, "link to") {
			t.Error("Link should not be expanded")
		}
	})
}

func BenchmarkSentenceParsing(b *testing.B) {
	parser, _ := NewSentenceParser(nil)
	text := strings.Repeat("This is a sentence. ", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(text)
	}
}

func BenchmarkMarkdownProcessing(b *testing.B) {
	processor := NewMarkdownProcessor(nil)
	markdown := `# Title

This is a paragraph with **bold** and *italic* text.

## Subtitle

- List item 1
- List item 2

> A blockquote

[A link](https://example.com)`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessMarkdown(markdown)
	}
}