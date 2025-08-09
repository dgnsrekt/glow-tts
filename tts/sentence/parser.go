// Package sentence provides sentence extraction and parsing for TTS.
package sentence

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/glow/v2/tts"
)

// Parser extracts sentences from markdown content.
type Parser struct {
	// Sentence detection patterns
	sentenceEndRegex   *regexp.Regexp
	abbreviationRegex  *regexp.Regexp
	
	// Markdown patterns
	codeBlockRegex     *regexp.Regexp
	inlineCodeRegex    *regexp.Regexp
	linkRegex          *regexp.Regexp
	emphasisRegex      *regexp.Regexp
	strongRegex        *regexp.Regexp
	headingRegex       *regexp.Regexp
	listItemRegex      *regexp.Regexp
	blockquoteRegex    *regexp.Regexp
	htmlTagRegex       *regexp.Regexp
	
	// Options
	skipCodeBlocks bool
	skipURLs       bool
	minLength      int
	
	// Common abbreviations that don't end sentences
	abbreviations map[string]bool
}

// NewParser creates a new sentence parser with optimized regex patterns.
func NewParser() *Parser {
	return &Parser{
		// Enhanced sentence ending pattern - handles combinations like ?!
		sentenceEndRegex: regexp.MustCompile(
			`([.!?]+[!?]*)(\s+|$|["')\]]*\s*)`,
		),
		
		// Common abbreviations pattern - more comprehensive
		abbreviationRegex: regexp.MustCompile(
			`(?i)\b(mr|mrs|ms|dr|prof|sr|jr|ph\.?d|m\.?d|b\.?a|m\.?a|b\.?s|` +
			`i\.?e|e\.?g|etc|vs|inc|ltd|co|corp|pg|pp|ed|eds|vol|vols|no|nos|` +
			`jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec|` +
			`mon|tue|wed|thu|fri|sat|sun|` +
			`st|rd|nd|th|ave|blvd|dept|div|est|ft|hr|hrs|min|mins|sec|secs|` +
			`lb|lbs|oz|kg|km|cm|mm|mi|yd|in|` +
			`u\.?s|u\.?k|u\.?n|e\.?u|n\.?y|l\.?a)\.$`,
		),
		
		// Markdown patterns
		codeBlockRegex:  regexp.MustCompile("(?s)```[^`]*```|~~~[^~]*~~~"),
		inlineCodeRegex: regexp.MustCompile("`[^`]+`"),
		linkRegex:       regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`),
		emphasisRegex:   regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`),
		strongRegex:     regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`),
		headingRegex:    regexp.MustCompile(`^#{1,6}\s+(.+)$`),
		listItemRegex:   regexp.MustCompile(`^[\s]*[-*+]\s+(.+)$|^[\s]*\d+\.\s+(.+)$`),
		blockquoteRegex: regexp.MustCompile(`^>\s*(.+)$`),
		htmlTagRegex:    regexp.MustCompile(`<[^>]+>`),
		
		skipCodeBlocks: true,
		skipURLs:       true,
		minLength:      3,
		
		abbreviations: makeAbbreviationMap(),
	}
}

// Parse extracts sentences from markdown content.
func (p *Parser) Parse(markdown string) []tts.Sentence {
	// Strip markdown to plain text while tracking positions
	plainText, positionMap := p.stripMarkdown(markdown)
	
	// Find sentence boundaries
	boundaries := p.findSentenceBoundaries(plainText)
	
	// Create sentences
	sentences := make([]tts.Sentence, 0, len(boundaries))
	
	for i := range boundaries {
		start := boundaries[i].start
		end := boundaries[i].end
		
		text := strings.TrimSpace(plainText[start:end])
		if len(text) < p.minLength {
			continue
		}
		
		// Map back to original markdown positions
		mdStart := start
		mdEnd := end
		if len(positionMap) > 0 && len(positionMap) == len(plainText) {
			if start < len(positionMap) {
				mdStart = positionMap[start]
			}
			if end > 0 && end <= len(positionMap) {
				mdEnd = positionMap[end-1] + 1
			}
		}
		
		sentence := tts.Sentence{
			Index:    len(sentences),
			Text:     text,
			Markdown: markdown[mdStart:mdEnd],
			Start:    mdStart,
			End:      mdEnd,
			Duration: p.EstimateDuration(text),
		}
		
		sentences = append(sentences, sentence)
	}
	
	return sentences
}

// EstimateDuration estimates the speaking duration for text.
func (p *Parser) EstimateDuration(text string) time.Duration {
	// More accurate estimation based on word count and complexity
	words := len(strings.Fields(text))
	if words == 0 {
		words = 1
	}
	
	// Base rate: 150 words per minute
	baseRate := 150.0
	
	// Adjust for complexity (punctuation, numbers, etc.)
	complexity := p.calculateComplexity(text)
	adjustedRate := baseRate * (1.0 - complexity*0.2) // Slow down for complex text
	
	seconds := float64(words) * 60.0 / adjustedRate
	return time.Duration(seconds * float64(time.Second))
}

// stripMarkdown removes markdown formatting while tracking positions.
func (p *Parser) stripMarkdown(markdown string) (string, []int) {
	// First pass: remove code blocks if requested
	processed := markdown
	if p.skipCodeBlocks {
		processed = p.codeBlockRegex.ReplaceAllString(processed, " ")
	}
	
	// Process line by line for better control
	lines := strings.Split(processed, "\n")
	var plainBuilder strings.Builder
	positionMap := make([]int, 0, len(markdown))
	
	originalPos := 0
	for lineNum, line := range lines {
		// Handle headings (add space after heading text)
		if matches := p.headingRegex.FindStringSubmatch(line); len(matches) > 1 {
			line = matches[1] + " "
		}
		
		// Handle list items
		if matches := p.listItemRegex.FindStringSubmatch(line); len(matches) > 0 {
			for _, match := range matches[1:] {
				if match != "" {
					line = match
					break
				}
			}
		}
		
		// Handle blockquotes
		if matches := p.blockquoteRegex.FindStringSubmatch(line); len(matches) > 1 {
			line = matches[1]
		}
		
		// Remove inline formatting
		line = p.stripInlineFormatting(line)
		
		// Add to plain text
		for _, ch := range line {
			plainBuilder.WriteRune(ch)
			positionMap = append(positionMap, originalPos)
		}
		
		// Add space instead of newline for better sentence flow
		if lineNum < len(lines)-1 && len(strings.TrimSpace(line)) > 0 {
			plainBuilder.WriteRune(' ')
			positionMap = append(positionMap, originalPos)
		}
		
		// Update original position
		originalPos += len(lines[lineNum])
		if lineNum < len(lines)-1 {
			originalPos++ // Account for newline
		}
	}
	
	return plainBuilder.String(), positionMap
}

// stripInlineFormatting removes inline markdown formatting.
func (p *Parser) stripInlineFormatting(text string) string {
	// Remove HTML tags
	text = p.htmlTagRegex.ReplaceAllString(text, "")
	
	// Remove inline code (keep empty space for readability)
	text = p.inlineCodeRegex.ReplaceAllString(text, "")
	
	// Remove links but keep text
	text = p.linkRegex.ReplaceAllString(text, "$1")
	
	// Remove strong emphasis (order matters - do strong before emphasis)
	text = p.strongRegex.ReplaceAllString(text, "$1$2")
	
	// Remove emphasis
	text = p.emphasisRegex.ReplaceAllString(text, "$1$2")
	
	// Clean up extra spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	return strings.TrimSpace(text)
}

// findSentenceBoundaries finds sentence boundaries with improved detection.
func (p *Parser) findSentenceBoundaries(text string) []boundary {
	var boundaries []boundary
	
	// Simple approach: look for sentence endings followed by space and uppercase
	runes := []rune(text)
	lastStart := 0
	
	for i := 0; i < len(runes); i++ {
		// Check for sentence ending punctuation
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// Collect all punctuation
			punctEnd := i + 1
			for punctEnd < len(runes) && (runes[punctEnd] == '!' || runes[punctEnd] == '?' || runes[punctEnd] == '.') {
				punctEnd++
			}
			
			// Handle quotes/parens after punctuation
			if punctEnd < len(runes) && (runes[punctEnd] == '"' || runes[punctEnd] == '\'' || runes[punctEnd] == ')' || runes[punctEnd] == ']') {
				punctEnd++
			}
			
			// Check if this is a real sentence end
			if p.isRealSentenceEndRunes(runes, i) {
				// Found a sentence boundary
				boundaries = append(boundaries, boundary{
					start: lastStart,
					end:   punctEnd,
				})
				
				// Skip whitespace
				for punctEnd < len(runes) && unicode.IsSpace(runes[punctEnd]) {
					punctEnd++
				}
				lastStart = punctEnd
				i = punctEnd - 1
			}
		}
	}
	
	// Add final boundary if there's remaining text
	if lastStart < len(runes) {
		remaining := strings.TrimSpace(string(runes[lastStart:]))
		if len(remaining) > 0 {
			boundaries = append(boundaries, boundary{
				start: lastStart,
				end:   len(runes),
			})
		}
	}
	
	// If no boundaries found, treat entire text as one sentence
	if len(boundaries) == 0 && len(strings.TrimSpace(text)) > 0 {
		boundaries = append(boundaries, boundary{
			start: 0,
			end:   len(text),
		})
	}
	
	// Convert rune positions to byte positions
	for i := range boundaries {
		boundaries[i].start = len(string(runes[:boundaries[i].start]))
		boundaries[i].end = len(string(runes[:boundaries[i].end]))
	}
	
	return boundaries
}

// isRealSentenceEndRunes checks if a position is a real sentence ending using runes.
func (p *Parser) isRealSentenceEndRunes(runes []rune, pos int) bool {
	if pos < 0 || pos >= len(runes) {
		return false
	}
	
	punct := runes[pos]
	
	// Check what comes before
	wordBefore := ""
	start := pos - 1
	for start >= 0 && !unicode.IsSpace(runes[start]) {
		start--
	}
	if start < pos {
		wordBefore = strings.ToLower(string(runes[start+1 : pos+1]))
	}
	
	// Check for known abbreviations
	if punct == '.' && len(wordBefore) > 0 {
		// Remove the period from the word for checking
		wordNoPeriod := strings.TrimSuffix(wordBefore, ".")
		if _, ok := p.abbreviations[wordNoPeriod]; ok {
			return false
		}
		if _, ok := p.abbreviations[wordBefore]; ok {
			return false
		}
		
		// Check for multi-part abbreviations like "Ph.D." or "U.S."
		if strings.Count(wordBefore, ".") > 1 {
			return false
		}
		
		// Special case for single letters followed by period
		if len(wordNoPeriod) == 1 && pos+1 < len(runes) {
			// Check if next char is also a period (like in "U.S.")
			if pos+2 < len(runes) && runes[pos+1] == ' ' && unicode.IsUpper(runes[pos-1]) {
				nextWord := ""
				end := pos + 2
				for end < len(runes) && !unicode.IsSpace(runes[end]) {
					nextWord += string(runes[end])
					end++
				}
				// If next word is also a single letter with period, it's likely an abbreviation
				if len(nextWord) == 2 && nextWord[1] == '.' && unicode.IsUpper(rune(nextWord[0])) {
					return false
				}
			}
		}
	}
	
	// Check for decimal numbers (need space between number and next sentence)
	// TODO: Known issue - doesn't correctly handle sentences like "Pi is 3.14159. E is 2.71828."
	// See KNOWN_ISSUES.md for details and potential solutions
	if punct == '.' && pos > 0 && pos < len(runes)-1 {
		if unicode.IsDigit(runes[pos-1]) {
			// Check if immediately followed by digit (decimal number)
			if unicode.IsDigit(runes[pos+1]) {
				return false
			}
			// Check if followed by space then uppercase (likely new sentence)
			if pos+2 < len(runes) && unicode.IsSpace(runes[pos+1]) && unicode.IsUpper(runes[pos+2]) {
				return true
			}
		}
	}
	
	// Check for ellipsis
	if punct == '.' && pos+2 < len(runes) && runes[pos+1] == '.' && runes[pos+2] == '.' {
		return false
	}
	
	// Check what comes after - needs space and capital letter or end of text
	if pos+1 >= len(runes) {
		return true // End of text
	}
	
	// Skip any closing quotes or brackets
	nextPos := pos + 1
	for nextPos < len(runes) && (runes[nextPos] == '"' || runes[nextPos] == '\'' || runes[nextPos] == ')' || runes[nextPos] == ']') {
		nextPos++
	}
	
	if nextPos >= len(runes) {
		return true // End of text
	}
	
	// Must have whitespace after punctuation
	if !unicode.IsSpace(runes[nextPos]) {
		return false
	}
	
	// Skip whitespace
	for nextPos < len(runes) && unicode.IsSpace(runes[nextPos]) {
		nextPos++
	}
	
	// Check if next word starts with uppercase (typical sentence start)
	if nextPos < len(runes) && unicode.IsUpper(runes[nextPos]) {
		return true
	}
	
	// For exclamation and question marks, be more lenient
	if punct == '!' || punct == '?' {
		return true
	}
	
	return false
}

// isRealSentenceEnd checks if a position is a real sentence ending.
func (p *Parser) isRealSentenceEnd(text string, pos int) bool {
	if pos < 0 || pos >= len(text) {
		return false
	}
	
	// Get the punctuation character
	punct := text[pos]
	
	// Get context before the punctuation
	contextStart := pos - 20
	if contextStart < 0 {
		contextStart = 0
	}
	contextBefore := text[contextStart : pos+1]
	
	// Check for abbreviations
	if punct == '.' {
		// Get the word before the period
		words := strings.Fields(contextBefore[:len(contextBefore)-1])
		if len(words) > 0 {
			lastWord := strings.ToLower(words[len(words)-1])
			// Check various forms of the abbreviation
			if _, ok := p.abbreviations[lastWord]; ok {
				return false
			}
			if _, ok := p.abbreviations[lastWord+"."]; ok {
				return false
			}
			// Check for multi-dot abbreviations like Ph.D.
			if strings.Contains(lastWord, ".") {
				return false
			}
		}
		
		// Check with regex for more complex patterns
		if p.abbreviationRegex.MatchString(contextBefore) {
			return false
		}
	}
	
	// Check for decimal numbers (e.g., "3.14")
	if punct == '.' && pos > 0 && pos < len(text)-1 {
		if unicode.IsDigit(rune(text[pos-1])) && unicode.IsDigit(rune(text[pos+1])) {
			return false
		}
	}
	
	// Check for ellipsis
	if punct == '.' && pos+2 < len(text) && text[pos:pos+3] == "..." {
		return false
	}
	
	// Check for URLs (basic check)
	if p.skipURLs && strings.Contains(contextBefore, "://") {
		return false
	}
	
	return true
}

// calculateComplexity estimates text complexity for duration adjustment.
func (p *Parser) calculateComplexity(text string) float64 {
	complexity := 0.0
	
	// Count numbers (slower to read)
	numbers := regexp.MustCompile(`\d+`).FindAllString(text, -1)
	complexity += float64(len(numbers)) * 0.02
	
	// Count punctuation (requires pauses)
	punctuation := regexp.MustCompile(`[,;:\-()]`).FindAllString(text, -1)
	complexity += float64(len(punctuation)) * 0.01
	
	// Long words (harder to pronounce)
	words := strings.Fields(text)
	longWords := 0
	for _, word := range words {
		if len(word) > 10 {
			longWords++
		}
	}
	complexity += float64(longWords) / float64(len(words)+1) * 0.1
	
	// Cap complexity at 0.5 (max 50% slowdown)
	if complexity > 0.5 {
		complexity = 0.5
	}
	
	return complexity
}

// makeAbbreviationMap creates a map of common abbreviations.
func makeAbbreviationMap() map[string]bool {
	abbrevs := []string{
		"mr", "mrs", "ms", "dr", "prof", "sr", "jr",
		"ph.d", "m.d", "b.a", "m.a", "b.s", "ph", "d",
		"llc", "inc", "ltd", "co", "corp",
		"i.e", "e.g", "etc", "vs", "cf", "al",
		"jan", "feb", "mar", "apr", "jun", "jul", "aug", "sep", "sept", "oct", "nov", "dec",
		"mon", "tue", "wed", "thu", "fri", "sat", "sun",
		"st", "rd", "ave", "blvd", "ln", "ct",
		"u.s", "u.k", "u.n", "e.u", "n.y", "l.a",
		"ft", "lbs", "oz", "kg", "km", "cm", "mm", "mi", "yd", "in",
		"hr", "hrs", "min", "mins", "sec", "secs",
	}
	
	m := make(map[string]bool)
	for _, abbrev := range abbrevs {
		m[abbrev] = true
		// Also add versions with periods
		if !strings.Contains(abbrev, ".") {
			m[abbrev+"."] = true
		}
	}
	return m
}

// boundary represents a sentence boundary.
type boundary struct {
	start int
	end   int
}