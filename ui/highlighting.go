package ui

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/lipgloss"
)

// HighlightSentence applies highlighting to the current sentence in the content.
// This is a simple implementation that highlights based on sentence index.
func HighlightSentence(content string, sentenceIndex int) string {
	if sentenceIndex < 0 {
		return content
	}
	
	// Define highlight style (yellow background, black text)
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("226")). // Yellow
		Foreground(lipgloss.Color("0")).   // Black
		Bold(true)
	
	// Simple sentence splitting (this should match the sentence parser logic)
	sentences := splitIntoSentences(content)
	if sentenceIndex >= len(sentences) {
		return content
	}
	
	// Build the highlighted content
	var result strings.Builder
	currentPos := 0
	
	for i, sentence := range sentences {
		// Find the sentence in the original content
		sentenceStart := strings.Index(content[currentPos:], sentence.text)
		if sentenceStart == -1 {
			continue
		}
		sentenceStart += currentPos
		
		// Add content before the sentence
		result.WriteString(content[currentPos:sentenceStart])
		
		// Add the sentence (highlighted or not)
		if i == sentenceIndex {
			result.WriteString(highlightStyle.Render(sentence.text))
		} else {
			result.WriteString(sentence.text)
		}
		
		currentPos = sentenceStart + len(sentence.text)
	}
	
	// Add any remaining content
	if currentPos < len(content) {
		result.WriteString(content[currentPos:])
	}
	
	return result.String()
}

type sentenceInfo struct {
	text  string
	start int
	end   int
}

// splitIntoSentences splits content into sentences.
// This is a simplified version - the real implementation should use the sentence parser.
func splitIntoSentences(content string) []sentenceInfo {
	var sentences []sentenceInfo
	var current strings.Builder
	start := 0
	
	for i, r := range content {
		current.WriteRune(r)
		
		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if this is really the end of a sentence
			if i+1 < len(content) {
				next := content[i+1]
				if next == ' ' || next == '\n' || next == '\t' {
					// End of sentence
					text := current.String()
					if strings.TrimSpace(text) != "" {
						sentences = append(sentences, sentenceInfo{
							text:  text,
							start: start,
							end:   i + 1,
						})
					}
					current.Reset()
					start = i + 1
				}
			} else {
				// End of content
				text := current.String()
				if strings.TrimSpace(text) != "" {
					sentences = append(sentences, sentenceInfo{
						text:  text,
						start: start,
						end:   i + 1,
					})
				}
			}
		}
	}
	
	// Add any remaining content as a sentence
	if current.Len() > 0 {
		text := current.String()
		if strings.TrimSpace(text) != "" {
			sentences = append(sentences, sentenceInfo{
				text:  text,
				start: start,
				end:   len(content),
			})
		}
	}
	
	return sentences
}

// ApplyTTSHighlighting applies TTS sentence highlighting to the rendered content.
func ApplyTTSHighlighting(content string, currentSentence int, enabled bool) string {
	if !enabled || currentSentence < 0 {
		return content
	}
	
	// For now, we'll add a visual indicator at the top
	indicator := fmt.Sprintf("🔊 TTS: Sentence %d", currentSentence+1)
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")). // Yellow
		Bold(true).
		MarginBottom(1)
	
	// TODO: Implement actual sentence highlighting in the viewport
	// This requires more complex integration with glamour rendering
	
	return indicatorStyle.Render(indicator) + "\n" + content
}