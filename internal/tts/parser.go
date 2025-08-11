package tts

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/glow/v2/internal/ttypes"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// SentenceParser extracts speakable sentences from markdown content.
type SentenceParser struct {
	stripMarkdown  bool
	skipCodeBlocks bool
	minLength      int
	maxLength      int
	abbreviations  map[string]bool
	titleAbbrevs   map[string]bool
}

// NewSentenceParser creates a new sentence parser with default settings.
func NewSentenceParser() *SentenceParser {
	return &SentenceParser{
		stripMarkdown:  true,
		skipCodeBlocks: true,
		minLength:      3,    // Minimum sentence length in characters
		maxLength:      1000, // Maximum sentence length for TTS
		abbreviations:  defaultAbbreviations(),
		titleAbbrevs:   defaultTitleAbbreviations(),
	}
}

// Parse extracts speakable sentences from markdown content.
func (p *SentenceParser) Parse(markdown string) ([]ttypes.Sentence, error) {
	// First, extract plain text from markdown
	plainText := p.extractPlainText(markdown)

	// Then split into sentences
	sentences := p.splitIntoSentences(plainText)

	// Create Sentence objects with position tracking
	result := make([]ttypes.Sentence, 0, len(sentences))
	offset := 0

	for i, text := range sentences {
		// Skip empty or too short sentences
		trimmed := strings.TrimSpace(text)
		if len(trimmed) < p.minLength {
			continue
		}

		// Truncate if too long
		if len(trimmed) > p.maxLength {
			trimmed = trimmed[:p.maxLength]
		}

		sentence := ttypes.Sentence{
			ID:          fmt.Sprintf("s%d", i),
			Text:        trimmed,
			Position:    len(result),
			StartOffset: offset,
			EndOffset:   offset + len(text),
			Priority:    ttypes.PriorityNormal,
			CacheKey:    "", // Will be generated when needed
		}

		result = append(result, sentence)
		offset += len(text) + 1 // +1 for space/newline
	}

	return result, nil
}

// StripMarkdown removes markdown formatting from text.
func (p *SentenceParser) StripMarkdown(text string) string {
	return p.extractPlainText(text)
}

// extractPlainText extracts plain text from markdown using goldmark.
func (p *SentenceParser) extractPlainText(markdown string) string {
	md := goldmark.New()
	reader := text.NewReader([]byte(markdown))
	doc := md.Parser().Parse(reader)

	var buf strings.Builder
	p.walkNode(doc, reader.Source(), &buf)

	return buf.String()
}

// walkNode recursively walks the AST and extracts text content.
func (p *SentenceParser) walkNode(node ast.Node, source []byte, buf *strings.Builder) {
	switch n := node.(type) {
	case *ast.CodeBlock:
		// Skip code blocks for TTS
		if p.skipCodeBlocks {
			return
		}
		// Otherwise add code block content with marker
		buf.WriteString("[Code block omitted]")
		buf.WriteString(" ")

	case *ast.FencedCodeBlock:
		// Skip fenced code blocks for TTS
		if p.skipCodeBlocks {
			return
		}
		buf.WriteString("[Code block omitted]")
		buf.WriteString(" ")

	case *ast.HTMLBlock:
		// Skip HTML blocks
		return

	case *ast.Text:
		// Add text content
		buf.Write(n.Segment.Value(source))

	case *ast.CodeSpan:
		// Include inline code with backticks for TTS clarity
		buf.WriteString("`")
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if text, ok := c.(*ast.Text); ok {
				buf.Write(text.Segment.Value(source))
			}
		}
		buf.WriteString("`")
		return // Don't process children again in default case

	case *ast.Heading:
		// Add heading text with a period for better sentence breaks
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		buf.WriteString(". ")
		return

	case *ast.Paragraph:
		// Process paragraph content
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		// Check if next character is not punctuation, then add space
		content := buf.String()
		if len(content) > 0 && content[len(content)-1] != '.' && content[len(content)-1] != '!' && content[len(content)-1] != '?' && content[len(content)-1] != ':' {
			buf.WriteString(". ")
		} else {
			buf.WriteString(" ")
		}
		return

	case *ast.List:
		// Process list items
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		return

	case *ast.ListItem:
		// Add list item content
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		buf.WriteString(". ")
		return

	case *ast.Link:
		// Include link text but not URL
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		return

	case *ast.Image:
		// Describe image
		buf.WriteString("[Image:")
		if n.Title != nil {
			buf.WriteString(" ")
			buf.Write(n.Title)
		} else {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if text, ok := c.(*ast.Text); ok {
					buf.WriteString(" ")
					buf.Write(text.Segment.Value(source))
				}
			}
		}
		buf.WriteString("]")
		return

	case *ast.Emphasis:
		// Include emphasized text
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walkNode(c, source, buf)
		}
		return

	// Note: ast.Strong might not exist in some goldmark versions
	// We handle it in the default case

	case *ast.Blockquote:
		// Include blockquote content with "Quote:" prefix
		// Each text node in blockquote should be treated as a separate sentence
		isFirst := true
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if paragraph, ok := c.(*ast.Paragraph); ok {
				// Process each text node within the paragraph as a separate sentence
				for textNode := paragraph.FirstChild(); textNode != nil; textNode = textNode.NextSibling() {
					if text, ok := textNode.(*ast.Text); ok {
						if isFirst {
							buf.WriteString("Quote: ")
							isFirst = false
						}

						// Add the text content
						buf.Write(text.Segment.Value(source))

						// Ensure sentence ending
						content := buf.String()
						if len(content) > 0 {
							lastChar := content[len(content)-1]
							if lastChar != '.' && lastChar != '!' && lastChar != '?' {
								buf.WriteString(".")
							}
							// Add space for next sentence (if any)
							if textNode.NextSibling() != nil {
								buf.WriteString(" ")
							}
						}
					} else {
						// Handle other node types within paragraph
						p.walkNode(textNode, source, buf)
					}
				}
			} else {
				// Handle other node types within blockquote
				p.walkNode(c, source, buf)
			}
		}
		return

	case *ast.ThematicBreak:
		// Add a pause for thematic breaks
		buf.WriteString(". ")
		return
	}

	// Process children for any other node types
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		p.walkNode(c, source, buf)
	}
}

// splitIntoSentences splits text into sentences with proper boundary detection.
func (p *SentenceParser) splitIntoSentences(text string) []string {
	// Clean up the text first
	text = p.cleanText(text)

	// Use character-by-character processing for more control
	// This handles abbreviations, numbers, and various punctuation

	var sentences []string
	var currentSentence strings.Builder

	// Process character by character for more control
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		currentSentence.WriteRune(runes[i])

		// Check for sentence boundary
		if p.isSentenceBoundary(runes, i) {
			sentence := strings.TrimSpace(currentSentence.String())
			if len(sentence) > 0 {
				sentences = append(sentences, sentence)
				currentSentence.Reset()
			}
		}
	}

	// Add any remaining text as a sentence
	if currentSentence.Len() > 0 {
		sentence := strings.TrimSpace(currentSentence.String())
		if len(sentence) > 0 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// isSentenceBoundary checks if the current position is a sentence boundary.
func (p *SentenceParser) isSentenceBoundary(runes []rune, pos int) bool {
	if pos >= len(runes)-1 {
		// At end of text, it's a boundary
		return true
	}

	current := runes[pos]

	// Check for sentence-ending punctuation
	if current != '.' && current != '!' && current != '?' && current != ':' {
		return false
	}

	// Check for ellipsis first
	if current == '.' && p.isEllipsis(runes, pos) {
		return false
	}

	// Check for decimal numbers (e.g., 3.14)
	if current == '.' && p.isDecimalNumber(runes, pos) {
		return false
	}

	// Check for URLs (e.g., https://example.com)
	if current == ':' && p.isURL(runes, pos) {
		return false
	}

	// Look back to check for abbreviations
	if current == '.' && p.isAbbreviation(runes, pos) {
		// Check if it's a title abbreviation (never sentence boundaries when followed by names)
		if p.isTitleAbbreviation(runes, pos) {
			return false
		}

		// For other abbreviations, check if followed by capital letter indicating new sentence
		nextPos := pos + 1
		for nextPos < len(runes) && unicode.IsSpace(runes[nextPos]) {
			nextPos++
		}
		// If followed by capital letter, treat as sentence boundary
		if nextPos < len(runes) && unicode.IsUpper(runes[nextPos]) {
			return true
		}
		return false
	}

	// Special handling for colons - treat as sentence boundary except for quote prefixes and image descriptions
	if current == ':' {
		// Look back to see if this is a "Quote:" prefix (don't split these)
		if pos >= 5 { // "Quote" is 5 characters
			prevText := string(runes[pos-5 : pos+1]) // Get "Quote:"
			if prevText == "Quote:" {
				return false
			}
		}

		// Look back to see if this is an "[Image:" prefix (don't split these)
		if pos >= 6 { // "[Image" is 6 characters
			prevText := string(runes[pos-6 : pos+1]) // Get "[Image:"
			if prevText == "[Image:" {
				return false
			}
		}

		// For other colons (like "Items:"), treat as sentence boundary
		return true
	}

	// Special case: period inside quotes
	// "Hello." She said - should break after the quote, not before
	if pos+1 < len(runes) && runes[pos+1] == '"' {
		// Don't break here, wait for the end of the quote
		return false
	}

	// Special case: closing quote followed by space and capital letter
	if current == '"' && pos+1 < len(runes) && unicode.IsSpace(runes[pos+1]) {
		// Check if next non-space character is capital letter
		nextPos := pos + 1
		for nextPos < len(runes) && unicode.IsSpace(runes[nextPos]) {
			nextPos++
		}
		if nextPos < len(runes) && unicode.IsUpper(runes[nextPos]) {
			return true
		}
	}

	// Look ahead for whitespace followed by capital letter (strong indicator of new sentence)
	if pos+1 < len(runes) {
		if !unicode.IsSpace(runes[pos+1]) {
			return false
		}
		// Skip whitespace and check for capital letter
		nextPos := pos + 1
		for nextPos < len(runes) && unicode.IsSpace(runes[nextPos]) {
			nextPos++
		}
		if nextPos < len(runes) && unicode.IsUpper(runes[nextPos]) {
			return true
		}
		// If we have whitespace but no capital letter, it's probably not a sentence boundary
		return false
	}

	return true
}

// isAbbreviation checks if a period is part of an abbreviation.
func (p *SentenceParser) isAbbreviation(runes []rune, pos int) bool {
	// Extract the word before the period
	start := pos - 1
	for start >= 0 && !unicode.IsSpace(runes[start]) {
		start--
	}
	start++

	if start >= pos {
		return false
	}

	word := string(runes[start:pos])
	word = strings.ToLower(word)

	// Check against known abbreviations
	return p.abbreviations[word]
}

// isTitleAbbreviation checks if a period is part of a title abbreviation.
func (p *SentenceParser) isTitleAbbreviation(runes []rune, pos int) bool {
	// Extract the word before the period
	start := pos - 1
	for start >= 0 && !unicode.IsSpace(runes[start]) {
		start--
	}
	start++

	if start >= pos {
		return false
	}

	word := string(runes[start:pos])
	word = strings.ToLower(word)

	// Check against known title abbreviations
	return p.titleAbbrevs[word]
}

// isDecimalNumber checks if a period is part of a decimal number.
func (p *SentenceParser) isDecimalNumber(runes []rune, pos int) bool {
	// Check if there's a digit before the period
	if pos > 0 && unicode.IsDigit(runes[pos-1]) {
		// Check if there's a digit after the period
		if pos+1 < len(runes) && unicode.IsDigit(runes[pos+1]) {
			return true
		}
	}
	return false
}

// isEllipsis checks if a period is part of an ellipsis.
func (p *SentenceParser) isEllipsis(runes []rune, pos int) bool {
	// Check for ... pattern
	if pos > 0 && runes[pos-1] == '.' {
		return true
	}
	if pos+1 < len(runes) && runes[pos+1] == '.' {
		return true
	}
	return false
}

// isURL checks if a colon is part of a URL scheme.
func (p *SentenceParser) isURL(runes []rune, pos int) bool {
	// Check if colon is preceded by http or https
	if pos >= 4 { // minimum for "http"
		preceding := string(runes[pos-4 : pos])
		if preceding == "http" {
			return true
		}
	}
	if pos >= 5 { // for "https"
		preceding := string(runes[pos-5 : pos])
		if preceding == "https" {
			return true
		}
	}
	if pos >= 3 { // for "ftp"
		preceding := string(runes[pos-3 : pos])
		if preceding == "ftp" {
			return true
		}
	}
	return false
}

// cleanText removes excessive whitespace and normalizes the text.
func (p *SentenceParser) cleanText(text string) string {
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace multiple newlines with period
	newlineRegex := regexp.MustCompile(`\n{2,}`)
	text = newlineRegex.ReplaceAllString(text, ". ")

	// Replace single newlines with space
	text = strings.ReplaceAll(text, "\n", " ")

	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// defaultAbbreviations returns common English abbreviations.
func defaultAbbreviations() map[string]bool {
	return map[string]bool{
		// Titles
		"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true,
		"sr": true, "jr": true, "ph.d": true, "m.d": true, "b.a": true,
		"m.a": true, "b.s": true, "m.s": true,

		// Common abbreviations
		"etc": true, "vs": true, "v": true, "e.g": true, "i.e": true,
		"inc": true, "ltd": true, "co": true, "corp": true,
		"jan": true, "feb": true, "mar": true, "apr": true, "jun": true,
		"jul": true, "aug": true, "sep": true, "sept": true, "oct": true,
		"nov": true, "dec": true,

		// Technical abbreviations
		"api": true, "url": true, "uri": true, "http": true, "https": true,
		"ftp": true, "ssh": true, "tcp": true, "ip": true, "dns": true,
		"cpu": true, "gpu": true, "ram": true, "ssd": true, "hdd": true,
		"os": true, "ui": true, "ux": true, "cli": true, "gui": true,
		"sdk": true, "ide": true, "sql": true, "nosql": true,

		// Units
		"ft": true, "in": true, "yd": true, "mi": true,
		"mm": true, "cm": true, "m": true, "km": true,
		"oz": true, "lb": true, "kg": true, "g": true,
		"sec": true, "min": true, "hr": true,

		// File extensions (common in documentation)
		"md": true, "txt": true, "pdf": true, "doc": true,
		"js": true, "ts": true, "go": true, "py": true,
		"html": true, "css": true, "json": true, "xml": true,
		"yml": true, "yaml": true, "toml": true,
	}
}

// defaultTitleAbbreviations returns abbreviations that are typically used as titles/prefixes.
func defaultTitleAbbreviations() map[string]bool {
	return map[string]bool{
		// Personal titles
		"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true,
		"sr": true, "jr": true,

		// Academic degrees that are commonly used as prefixes
		"ph.d": true, "m.d": true, "b.a": true, "m.a": true, "b.s": true, "m.s": true,
	}
}

// ParserOption is a functional option for configuring the parser.
type ParserOption func(*SentenceParser)

// WithMinLength sets the minimum sentence length.
func WithMinLength(length int) ParserOption {
	return func(p *SentenceParser) {
		p.minLength = length
	}
}

// WithMaxLength sets the maximum sentence length.
func WithMaxLength(length int) ParserOption {
	return func(p *SentenceParser) {
		p.maxLength = length
	}
}

// WithCodeBlocks enables or disables code block inclusion.
func WithCodeBlocks(include bool) ParserOption {
	return func(p *SentenceParser) {
		p.skipCodeBlocks = !include
	}
}

// NewSentenceParserWithOptions creates a parser with custom options.
func NewSentenceParserWithOptions(opts ...ParserOption) *SentenceParser {
	p := NewSentenceParser()
	for _, opt := range opts {
		opt(p)
	}
	return p
}
