package tts

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
)

// ParsedSentence represents a parsed sentence from markdown content
type ParsedSentence struct {
	// Text is the clean, speakable text for TTS synthesis
	Text string
	// Position is the index position in the document (0-based)
	Position int
	// Original contains the original markdown text
	Original string
	// Type indicates the markdown element type (paragraph, header, etc.)
	Type SentenceType
}

// SentenceType represents the type of markdown element
type SentenceType int

const (
	SentenceTypeParagraph SentenceType = iota
	SentenceTypeHeader
	SentenceTypeListItem
	SentenceTypeBlockquote
	SentenceTypeCodeBlock
	SentenceTypeLink
	SentenceTypeEmphasis
)

// ParserConfig contains configuration for the sentence parser
type ParserConfig struct {
	// IncludeCodeBlocks determines if code blocks should be included
	IncludeCodeBlocks bool
	// ExpandLinks determines if links should be expanded with "link to" prefix
	ExpandLinks bool
	// PreserveEmphasis determines if emphasis should be preserved in speech
	PreserveEmphasis bool
	// MinSentenceLength is the minimum length for a valid sentence
	MinSentenceLength int
	// MaxSentenceLength is the maximum length before splitting
	MaxSentenceLength int
}

// DefaultParserConfig returns a default configuration for the parser
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		IncludeCodeBlocks: false,
		ExpandLinks:       true,
		PreserveEmphasis:  true,
		MinSentenceLength: 3,
		MaxSentenceLength: 500,
	}
}

// SentenceParser extracts sentences from markdown content
type SentenceParser struct {
	config   *ParserConfig
	renderer *glamour.TermRenderer
	// Regex patterns for sentence detection
	sentenceEndPattern   *regexp.Regexp
	abbreviationPattern  *regexp.Regexp
	decimalPattern       *regexp.Regexp
	ellipsisPattern      *regexp.Regexp
}

// NewSentenceParser creates a new sentence parser with the given configuration
func NewSentenceParser(config *ParserConfig) (*SentenceParser, error) {
	if config == nil {
		config = DefaultParserConfig()
	}

	// For now, don't create a glamour renderer - it can hang in some environments
	// We'll use a simpler markdown stripping approach
	// TODO: Investigate glamour hanging issue and re-enable when fixed
	
	return &SentenceParser{
		config:   config,
		renderer: nil, // We'll handle markdown stripping without glamour for now
		// Compile regex patterns for sentence detection
		sentenceEndPattern:   regexp.MustCompile(`[.!?]+[\s\n]+|[.!?]+$`),
		abbreviationPattern:  regexp.MustCompile(`\b(?:Mr|Mrs|Ms|Dr|Prof|Sr|Jr|Inc|Ltd|Corp|Co|vs|etc|i\.e|e\.g|Ph\.D|M\.D|B\.A|M\.A|B\.S|M\.S)\.`),
		decimalPattern:       regexp.MustCompile(`\d+\.\d+`),
		ellipsisPattern:      regexp.MustCompile(`\.{3,}`),
	}, nil
}

// Parse extracts sentences from markdown content
func (p *SentenceParser) Parse(markdown string) ([]ParsedSentence, error) {
	if strings.TrimSpace(markdown) == "" {
		return []ParsedSentence{}, nil
	}

	var cleanText string
	
	// If we have a renderer, use it. Otherwise, use simple markdown stripping
	if p.renderer != nil {
		// Use glamour renderer
		rendered, err := p.renderer.Render(markdown)
		if err != nil {
			return nil, fmt.Errorf("failed to render markdown: %w", err)
		}
		cleanText = p.cleanRenderedText(rendered)
	} else {
		// Simple markdown stripping without glamour
		cleanText = p.stripMarkdownSimple(markdown)
	}

	// Extract sentences from the clean text
	sentences := p.extractSentences(cleanText, markdown)

	return sentences, nil
}

// stripMarkdownSimple removes basic markdown formatting without using glamour
func (p *SentenceParser) stripMarkdownSimple(markdown string) string {
	text := markdown
	
	// Remove HTML tags and their content (like badges and embedded images)
	text = regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(text, "")
	
	// Remove HTML comments
	text = regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(text, "")
	
	// Remove code blocks
	text = regexp.MustCompile("```[^`]*```").ReplaceAllString(text, "")
	text = regexp.MustCompile("`[^`]+`").ReplaceAllString(text, "")
	
	// Remove images (including ones with URLs)
	text = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(text, "")
	
	// Remove standalone URLs (http/https)
	text = regexp.MustCompile(`https?://[^\s\)]+`).ReplaceAllString(text, "")
	
	// Remove headers but keep the text
	text = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(text, "")
	
	// Remove bold and italic
	text = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`_{1,3}([^_]+)_{1,3}`).ReplaceAllString(text, "$1")
	
	// Remove links but keep text
	text = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(text, "$1")
	
	// Remove list markers
	text = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`).ReplaceAllString(text, "")
	
	// Remove blockquotes
	text = regexp.MustCompile(`(?m)^>\s+`).ReplaceAllString(text, "")
	
	// Remove horizontal rules
	text = regexp.MustCompile(`(?m)^[\s]*[-*_]{3,}[\s]*$`).ReplaceAllString(text, "")
	
	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	
	return text
}

// cleanRenderedText removes ANSI codes and extra whitespace from rendered text
func (p *SentenceParser) cleanRenderedText(text string) string {
	// Remove ANSI escape codes
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	text = ansiPattern.ReplaceAllString(text, "")

	// Normalize whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	return text
}

// extractSentences splits text into sentences with proper boundary detection
func (p *SentenceParser) extractSentences(text, originalMarkdown string) []ParsedSentence {
	var sentences []ParsedSentence
	position := 0

	// Handle empty text
	if strings.TrimSpace(text) == "" {
		return sentences
	}

	// Protect decimals and ellipsis from being split
	protectedText := p.protectSpecialPatterns(text)

	// Split by sentence boundaries
	parts := p.sentenceEndPattern.Split(protectedText, -1)
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		
		// Skip empty or too short sentences
		if len(part) < p.config.MinSentenceLength {
			continue
		}

		// Restore protected patterns
		part = p.restoreSpecialPatterns(part)

		// Handle sentences that are too long
		if len(part) > p.config.MaxSentenceLength {
			subSentences := p.splitLongSentence(part)
			for _, sub := range subSentences {
				sentences = append(sentences, ParsedSentence{
					Text:     sub,
					Position: position,
					Original: p.findOriginalText(sub, originalMarkdown),
					Type:     SentenceTypeParagraph,
				})
				position++
			}
		} else {
			sentences = append(sentences, ParsedSentence{
				Text:     part,
				Position: position,
				Original: p.findOriginalText(part, originalMarkdown),
				Type:     SentenceTypeParagraph,
			})
			position++
		}
	}

	return sentences
}

// protectSpecialPatterns temporarily replaces patterns that shouldn't be split
func (p *SentenceParser) protectSpecialPatterns(text string) string {
	// Protect decimal numbers
	text = p.decimalPattern.ReplaceAllStringFunc(text, func(match string) string {
		return strings.ReplaceAll(match, ".", "DECIMAL")
	})
	
	// Protect ellipsis
	text = p.ellipsisPattern.ReplaceAllString(text, "ELLIPSIS")
	
	// Protect common abbreviations
	text = p.abbreviationPattern.ReplaceAllStringFunc(text, func(match string) string {
		return strings.ReplaceAll(match, ".", "ABBREV")
	})
	
	return text
}

// restoreSpecialPatterns restores the protected patterns
func (p *SentenceParser) restoreSpecialPatterns(text string) string {
	text = strings.ReplaceAll(text, "DECIMAL", ".")
	text = strings.ReplaceAll(text, "ELLIPSIS", "...")
	text = strings.ReplaceAll(text, "ABBREV", ".")
	return text
}

// splitLongSentence breaks up sentences that are too long
func (p *SentenceParser) splitLongSentence(text string) []string {
	var parts []string
	
	// If text is already within limits, return as is
	if len(text) <= p.config.MaxSentenceLength {
		return []string{text}
	}
	
	// Try to split on commas first, then semicolons, then conjunctions
	splitPoints := []string{", ", "; ", " and ", " or ", " but "}
	
	for _, splitter := range splitPoints {
		candidates := strings.Split(text, splitter)
		if len(candidates) > 1 {
			current := ""
			for i, candidate := range candidates {
				candidate = strings.TrimSpace(candidate)
				if candidate == "" {
					continue
				}
				
				// Add the splitter back (except for the last part)
				if i < len(candidates)-1 && splitter != "; " {
					candidate += strings.TrimSpace(splitter)
				}
				
				if len(current)+len(candidate)+1 <= p.config.MaxSentenceLength {
					if current != "" {
						current += " " + candidate
					} else {
						current = candidate
					}
				} else {
					if current != "" {
						parts = append(parts, current)
					}
					current = candidate
				}
			}
			
			if current != "" {
				parts = append(parts, current)
			}
			
			// If we successfully split the text, return
			if len(parts) > 0 {
				// Check if any part is still too long and recursively split it
				var finalParts []string
				for _, part := range parts {
					if len(part) > p.config.MaxSentenceLength {
						subParts := p.splitLongSentence(part)
						finalParts = append(finalParts, subParts...)
					} else {
						finalParts = append(finalParts, part)
					}
				}
				return finalParts
			}
		}
	}
	
	// If we couldn't split naturally, break at word boundaries
	words := strings.Fields(text)
	current := ""
	for _, word := range words {
		if len(current)+len(word)+1 <= p.config.MaxSentenceLength {
			if current != "" {
				current += " " + word
			} else {
				current = word
			}
		} else {
			if current != "" {
				parts = append(parts, current)
			}
			current = word
		}
	}
	
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}

// findOriginalText attempts to locate the original markdown for a sentence
func (p *SentenceParser) findOriginalText(cleanText, markdown string) string {
	// This is a simplified implementation
	// In a production system, we'd maintain better mapping during parsing
	
	// Try to find the clean text in the markdown
	index := strings.Index(strings.ToLower(markdown), strings.ToLower(cleanText[:min(len(cleanText), 20)]))
	if index == -1 {
		return cleanText
	}
	
	// Extract a reasonable portion of the original
	endIndex := index + len(cleanText) + 20
	if endIndex > len(markdown) {
		endIndex = len(markdown)
	}
	
	original := markdown[index:endIndex]
	
	// Clean up the boundaries
	if idx := strings.Index(original, "\n"); idx > 0 && idx < len(cleanText) {
		original = original[:idx]
	}
	
	return strings.TrimSpace(original)
}

// ParseSentence parses a single sentence from text
func (p *SentenceParser) ParseSentence(text string) ParsedSentence {
	cleanText := p.cleanRenderedText(text)
	return ParsedSentence{
		Text:     cleanText,
		Position: 0,
		Original: text,
		Type:     SentenceTypeParagraph,
	}
}

// ParseSentences implements the TextParser interface
func (p *SentenceParser) ParseSentences(text string) ([]Sentence, error) {
	// Parse the markdown text
	parsed, err := p.Parse(text)
	if err != nil {
		return nil, err
	}
	
	// Convert ParsedSentence to Sentence
	sentences := make([]Sentence, 0, len(parsed))
	for i, ps := range parsed {
		// Skip empty sentences
		if ps.Text == "" {
			continue
		}
		
		sentences = append(sentences, Sentence{
			Text:     ps.Text,
			Position: i,
			Original: ps.Original,
		})
	}
	
	return sentences, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}