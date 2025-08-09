package sentence

import (
	"strings"
	"testing"
	"time"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser returned nil")
	}
	
	if parser.minLength != 3 {
		t.Errorf("Expected minLength=3, got %d", parser.minLength)
	}
	
	if !parser.skipCodeBlocks {
		t.Error("Expected skipCodeBlocks to be true by default")
	}
}

func TestParsePlainText(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "simple sentences",
			input: "Hello world. How are you? I'm fine!",
			expected: []string{
				"Hello world.",
				"How are you?",
				"I'm fine!",
			},
		},
		{
			name:  "sentences with newlines",
			input: "First sentence.\nSecond sentence.\nThird sentence.",
			expected: []string{
				"First sentence.",
				"Second sentence.",
				"Third sentence.",
			},
		},
		{
			name:  "sentences with multiple spaces",
			input: "First.  Second.   Third.",
			expected: []string{
				"First.",
				"Second.",
				"Third.",
			},
		},
		{
			name:  "sentence with ellipsis",
			input: "Wait... I'm thinking. Done!",
			expected: []string{
				"Wait... I'm thinking.",
				"Done!",
			},
		},
		{
			name:  "mixed punctuation",
			input: "Really? Yes! Of course. Why not?!",
			expected: []string{
				"Really?",
				"Yes!",
				"Of course.",
				"Why not?!",
			},
		},
		{
			name:  "quoted sentences",
			input: `She said "Hello." Then she left.`,
			expected: []string{
				`She said "Hello."`,
				"Then she left.",
			},
		},
		{
			name:  "parenthetical sentences",
			input: "Main point (see appendix). Next point.",
			expected: []string{
				"Main point (see appendix).",
				"Next point.",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := parser.Parse(tt.input)
			
			if len(sentences) != len(tt.expected) {
				t.Errorf("Expected %d sentences, got %d", len(tt.expected), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}
			
			for i, expected := range tt.expected {
				if sentences[i].Text != expected {
					t.Errorf("Sentence %d: expected %q, got %q", i, expected, sentences[i].Text)
				}
			}
		})
	}
}

func TestParseMarkdown(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "headings",
			input: `# Title
This is a paragraph. It has sentences.

## Subtitle
Another paragraph here.`,
			expected: []string{
				"Title This is a paragraph.",
				"It has sentences.",
				"Subtitle Another paragraph here.",
			},
		},
		{
			name:  "bold and italic",
			input: "**Bold text.** *Italic text.* Normal text.",
			expected: []string{
				"Bold text.",
				"Italic text.",
				"Normal text.",
			},
		},
		{
			name:  "inline code",
			input: "Use `fmt.Println()` to print. It's simple.",
			expected: []string{
				"Use to print.",
				"It's simple.",
			},
		},
		{
			name: "lists",
			input: `- First item.
- Second item.
- Third item.`,
			expected: []string{
				"First item.",
				"Second item.",
				"Third item.",
			},
		},
		{
			name: "numbered lists",
			input: `1. First step.
2. Second step.
3. Third step.`,
			expected: []string{
				"First step.",
				"Second step.",
				"Third step.",
			},
		},
		{
			name:  "links",
			input: "Check [this link](https://example.com). It's useful.",
			expected: []string{
				"Check this link.",
				"It's useful.",
			},
		},
		{
			name: "blockquotes",
			input: `> This is a quote.
> It continues here.

Regular text.`,
			expected: []string{
				"This is a quote.",
				"It continues here.",
				"Regular text.",
			},
		},
		{
			name: "code blocks excluded",
			input: `Text before.
` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `
Text after.`,
			expected: []string{
				"Text before.",
				"Text after.",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := parser.Parse(tt.input)
			
			if len(sentences) != len(tt.expected) {
				t.Errorf("Expected %d sentences, got %d", len(tt.expected), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}
			
			for i, expected := range tt.expected {
				if sentences[i].Text != expected {
					t.Errorf("Sentence %d: expected %q, got %q", i, expected, sentences[i].Text)
				}
			}
		})
	}
}

func TestParseAbbreviations(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "common titles",
			input: "Dr. Smith arrived. Mr. Jones left.",
			expected: []string{
				"Dr. Smith arrived.",
				"Mr. Jones left.",
			},
		},
		{
			name:  "academic degrees",
			input: "Jane Doe, Ph.D. teaches here. John has a B.S. degree.",
			expected: []string{
				"Jane Doe, Ph.D. teaches here.",
				"John has a B.S. degree.",
			},
		},
		{
			name:  "business abbreviations",
			input: "Apple Inc. is large. Microsoft Corp. too.",
			expected: []string{
				"Apple Inc. is large.",
				"Microsoft Corp. too.",
			},
		},
		{
			name:  "latin abbreviations",
			input: "Many reasons, e.g. cost. Also consider efficiency, i.e. speed.",
			expected: []string{
				"Many reasons, e.g. cost.",
				"Also consider efficiency, i.e. speed.",
			},
		},
		{
			name:  "months",
			input: "Meeting on Jan. 5th. Deadline is Dec. 31st.",
			expected: []string{
				"Meeting on Jan. 5th.",
				"Deadline is Dec. 31st.",
			},
		},
		{
			name:  "addresses",
			input: "Located at 123 Main St. near Park Ave. intersection.",
			expected: []string{
				"Located at 123 Main St. near Park Ave. intersection.",
			},
		},
		{
			name:  "measurements",
			input: "The box is 10 ft. tall. It weighs 50 lbs. total.",
			expected: []string{
				"The box is 10 ft. tall.",
				"It weighs 50 lbs. total.",
			},
		},
		{
			name:  "countries",
			input: "U.S. policy changed. U.K. followed suit.",
			expected: []string{
				"U.S. policy changed.",
				"U.K. followed suit.",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := parser.Parse(tt.input)
			
			if len(sentences) != len(tt.expected) {
				t.Errorf("Expected %d sentences, got %d", len(tt.expected), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}
			
			for i, expected := range tt.expected {
				if sentences[i].Text != expected {
					t.Errorf("Sentence %d: expected %q, got %q", i, expected, sentences[i].Text)
				}
			}
		})
	}
}

func TestParseNumbers(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "decimal numbers",
			input: "Pi is 3.14159. E is 2.71828.",
			expected: []string{
				"Pi is 3.14159.",
				"E is 2.71828.",
			},
		},
		{
			name:  "versions",
			input: "Version 1.2.3 released. Update from 1.0.0 required.",
			expected: []string{
				"Version 1.2.3 released.",
				"Update from 1.0.0 required.",
			},
		},
		{
			name:  "money",
			input: "Cost is $19.99. Save $5.00 today!",
			expected: []string{
				"Cost is $19.99.",
				"Save $5.00 today!",
			},
		},
		{
			name:  "percentages",
			input: "Growth of 12.5% expected. Current rate is 3.2%.",
			expected: []string{
				"Growth of 12.5% expected.",
				"Current rate is 3.2%.",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := parser.Parse(tt.input)
			
			if len(sentences) != len(tt.expected) {
				t.Errorf("Expected %d sentences, got %d", len(tt.expected), len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
				return
			}
			
			for i, expected := range tt.expected {
				if sentences[i].Text != expected {
					t.Errorf("Sentence %d: expected %q, got %q", i, expected, sentences[i].Text)
				}
			}
		})
	}
}

func TestEstimateDuration(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name             string
		text             string
		expectedMin      time.Duration
		expectedMax      time.Duration
	}{
		{
			name:        "short sentence",
			text:        "Hello world.",
			expectedMin: 500 * time.Millisecond,
			expectedMax: 1500 * time.Millisecond,
		},
		{
			name:        "medium sentence",
			text:        "This is a medium length sentence with several words.",
			expectedMin: 2 * time.Second,
			expectedMax: 5 * time.Second,
		},
		{
			name:        "long sentence",
			text:        strings.Repeat("word ", 30) + "end.",
			expectedMin: 10 * time.Second,
			expectedMax: 15 * time.Second,
		},
		{
			name:        "complex sentence with numbers",
			text:        "The value increased by 23.5% from $1,234.56 to $1,524.89 in Q3 2024.",
			expectedMin: 3 * time.Second,
			expectedMax: 7 * time.Second,
		},
		{
			name:        "sentence with long words",
			text:        "Pneumonoultramicroscopicsilicovolcanoconiosis is extraordinarily complicated.",
			expectedMin: 1500 * time.Millisecond,
			expectedMax: 5 * time.Second,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := parser.EstimateDuration(tt.text)
			
			if duration < tt.expectedMin {
				t.Errorf("Duration %v is less than expected minimum %v", duration, tt.expectedMin)
			}
			if duration > tt.expectedMax {
				t.Errorf("Duration %v is greater than expected maximum %v", duration, tt.expectedMax)
			}
		})
	}
}

func TestParseEdgeCases(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		input    string
		expected int // expected number of sentences
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "only whitespace",
			input:    "   \n\t  ",
			expected: 0,
		},
		{
			name:     "no punctuation",
			input:    "This text has no ending punctuation",
			expected: 1,
		},
		{
			name:     "only punctuation",
			input:    "...!!!???",
			expected: 1, // Gets parsed as one segment
		},
		{
			name:     "very short sentences",
			input:    "A. B. C.",
			expected: 0, // All too short
		},
		{
			name:     "single word sentences",
			input:    "Yes. No. Maybe.",
			expected: 3,
		},
		{
			name:     "URLs not split",
			input:    "Visit https://example.com/page.html for info. Check it out.",
			expected: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := parser.Parse(tt.input)
			
			if len(sentences) != tt.expected {
				t.Errorf("Expected %d sentences, got %d", tt.expected, len(sentences))
				for i, s := range sentences {
					t.Logf("  [%d]: %q", i, s.Text)
				}
			}
		})
	}
}

func TestSentenceMetadata(t *testing.T) {
	parser := NewParser()
	
	input := "First sentence. Second sentence. Third sentence."
	sentences := parser.Parse(input)
	
	if len(sentences) != 3 {
		t.Fatalf("Expected 3 sentences, got %d", len(sentences))
	}
	
	// Check indices
	for i, s := range sentences {
		if s.Index != i {
			t.Errorf("Sentence %d has incorrect index %d", i, s.Index)
		}
	}
	
	// Check positions are valid
	for i, s := range sentences {
		if s.Start < 0 || s.End > len(input) {
			t.Errorf("Sentence %d has invalid positions: start=%d, end=%d (input len=%d)",
				i, s.Start, s.End, len(input))
		}
		if s.Start >= s.End {
			t.Errorf("Sentence %d has invalid range: start=%d >= end=%d",
				i, s.Start, s.End)
		}
	}
	
	// Check durations are positive
	for i, s := range sentences {
		if s.Duration <= 0 {
			t.Errorf("Sentence %d has non-positive duration: %v", i, s.Duration)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkParsePlainText(b *testing.B) {
	parser := NewParser()
	text := strings.Repeat("This is a test sentence. ", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(text)
	}
}

func BenchmarkParseMarkdown(b *testing.B) {
	parser := NewParser()
	markdown := `
# Heading

This is a **paragraph** with *emphasis*. It has [links](http://example.com).

## Another Section

- List item one.
- List item two.
- List item three.

> A blockquote with text.

` + "```code" + `
Some code here
` + "```"
	
	// Repeat to make it larger
	markdown = strings.Repeat(markdown, 10)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(markdown)
	}
}

func BenchmarkParse10KB(b *testing.B) {
	parser := NewParser()
	
	// Generate approximately 10KB of markdown
	var builder strings.Builder
	for builder.Len() < 10*1024 {
		builder.WriteString("# Section\n\n")
		builder.WriteString("This is a paragraph with multiple sentences. ")
		builder.WriteString("Each sentence should be properly detected. ")
		builder.WriteString("The parser needs to handle this efficiently.\n\n")
		builder.WriteString("- List item one.\n")
		builder.WriteString("- List item two.\n\n")
		builder.WriteString("> A quoted section.\n\n")
	}
	text := builder.String()[:10*1024] // Exactly 10KB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(text)
	}
}

// TestPerformanceRequirement validates <100ms for 10KB
func TestPerformanceRequirement(t *testing.T) {
	parser := NewParser()
	
	// Generate 10KB of realistic markdown
	var builder strings.Builder
	for builder.Len() < 10*1024 {
		builder.WriteString("# Documentation Section\n\n")
		builder.WriteString("This is a typical paragraph in documentation. ")
		builder.WriteString("It contains multiple sentences with various formatting. ")
		builder.WriteString("**Bold text** and *italic text* are common. ")
		builder.WriteString("Links like [this](http://example.com) appear frequently.\n\n")
		builder.WriteString("## Subsection\n\n")
		builder.WriteString("- First list item with text.\n")
		builder.WriteString("- Second list item with more text.\n")
		builder.WriteString("- Third list item.\n\n")
		builder.WriteString("> Blockquotes are used for emphasis.\n\n")
		builder.WriteString("Code examples use `inline code` formatting.\n\n")
	}
	text := builder.String()[:10*1024]
	
	// Warm up
	_ = parser.Parse(text)
	
	// Measure performance
	start := time.Now()
	sentences := parser.Parse(text)
	elapsed := time.Since(start)
	
	// Check performance requirement
	if elapsed > 100*time.Millisecond {
		t.Errorf("Performance requirement failed: parsing 10KB took %v (>100ms)", elapsed)
	}
	
	// Verify we got reasonable output
	if len(sentences) < 10 {
		t.Errorf("Expected at least 10 sentences from 10KB text, got %d", len(sentences))
	}
	
	t.Logf("Parsed 10KB in %v, found %d sentences", elapsed, len(sentences))
}