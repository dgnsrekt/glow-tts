package tts

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// MarkdownProcessor handles advanced markdown parsing for TTS
type MarkdownProcessor struct {
	parser goldmark.Markdown
	config *ParserConfig
}

// NewMarkdownProcessor creates a new markdown processor
func NewMarkdownProcessor(config *ParserConfig) *MarkdownProcessor {
	if config == nil {
		config = DefaultParserConfig()
	}

	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return &MarkdownProcessor{
		parser: md,
		config: config,
	}
}

// ProcessMarkdown extracts structured content from markdown
func (mp *MarkdownProcessor) ProcessMarkdown(source string) ([]MarkdownElement, error) {
	reader := text.NewReader([]byte(source))
	doc := mp.parser.Parser().Parse(reader)
	
	elements := []MarkdownElement{}
	
	// Walk the AST and extract elements
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		
		element := mp.processNode(n, source)
		if element != nil {
			elements = append(elements, *element)
		}
		
		return ast.WalkContinue, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk markdown AST: %w", err)
	}
	
	return elements, nil
}

// MarkdownElement represents a parsed markdown element
type MarkdownElement struct {
	Type        ElementType
	Content     string
	Level       int // For headers
	Language    string // For code blocks
	URL         string // For links
	Alt         string // For images
	IsOrdered   bool // For lists
	Children    []MarkdownElement
}

// ElementType represents the type of markdown element
type ElementType int

const (
	ElementParagraph ElementType = iota
	ElementHeading
	ElementCodeBlock
	ElementInlineCode
	ElementLink
	ElementImage
	ElementList
	ElementListItem
	ElementBlockquote
	ElementEmphasis
	ElementStrong
	ElementLineBreak
	ElementHorizontalRule
	ElementTable
)

// processNode converts an AST node to a MarkdownElement
func (mp *MarkdownProcessor) processNode(node ast.Node, source string) *MarkdownElement {
	switch n := node.(type) {
	case *ast.Paragraph:
		return &MarkdownElement{
			Type:    ElementParagraph,
			Content: mp.extractText(n, source),
		}
		
	case *ast.Heading:
		return &MarkdownElement{
			Type:    ElementHeading,
			Level:   n.Level,
			Content: mp.extractText(n, source),
		}
		
	case *ast.CodeBlock:
		if !mp.config.IncludeCodeBlocks {
			return nil
		}
		return &MarkdownElement{
			Type:    ElementCodeBlock,
			Content: mp.extractCodeBlock(n, source),
		}
		
	case *ast.FencedCodeBlock:
		if !mp.config.IncludeCodeBlocks {
			return nil
		}
		lang := ""
		if n.Info != nil && n.Info.Segment.Len() > 0 {
			lang = string(n.Info.Segment.Value([]byte(source)))
		}
		return &MarkdownElement{
			Type:     ElementCodeBlock,
			Language: lang,
			Content:  mp.extractCodeBlock(n, source),
		}
		
	case *ast.Link:
		url := string(n.Destination)
		content := mp.extractText(n, source)
		if mp.config.ExpandLinks {
			content = fmt.Sprintf("link to %s", content)
		}
		return &MarkdownElement{
			Type:    ElementLink,
			Content: content,
			URL:     url,
		}
		
	case *ast.Image:
		alt := mp.extractText(n, source)
		url := string(n.Destination)
		return &MarkdownElement{
			Type:    ElementImage,
			Alt:     alt,
			URL:     url,
			Content: fmt.Sprintf("image: %s", alt),
		}
		
	case *ast.List:
		return &MarkdownElement{
			Type:      ElementList,
			IsOrdered: n.IsOrdered(),
		}
		
	case *ast.ListItem:
		return &MarkdownElement{
			Type:    ElementListItem,
			Content: mp.extractText(n, source),
		}
		
	case *ast.Blockquote:
		return &MarkdownElement{
			Type:    ElementBlockquote,
			Content: mp.extractText(n, source),
		}
		
	case *ast.Emphasis:
		content := mp.extractText(n, source)
		if mp.config.PreserveEmphasis {
			content = fmt.Sprintf("emphasized: %s", content)
		}
		return &MarkdownElement{
			Type:    ElementEmphasis,
			Content: content,
		}
		
	case *ast.CodeSpan:
		content := mp.extractText(n, source)
		if mp.config.IncludeCodeBlocks {
			content = fmt.Sprintf("code: %s", content)
		} else {
			return nil
		}
		return &MarkdownElement{
			Type:    ElementInlineCode,
			Content: content,
		}
		
	case *ast.ThematicBreak:
		return &MarkdownElement{
			Type:    ElementHorizontalRule,
			Content: "horizontal rule",
		}
	}
	
	return nil
}

// extractText extracts text content from a node
func (mp *MarkdownProcessor) extractText(node ast.Node, source string) string {
	var text strings.Builder
	
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			text.Write(c.Segment.Value([]byte(source)))
		case *ast.CodeSpan:
			if mp.config.IncludeCodeBlocks {
				text.WriteString(mp.extractText(c, source))
			}
		default:
			// Recursively extract text from child nodes
			text.WriteString(mp.extractText(c, source))
		}
	}
	
	return strings.TrimSpace(text.String())
}

// extractCodeBlock extracts code block content
func (mp *MarkdownProcessor) extractCodeBlock(node ast.Node, source string) string {
	var lines []string
	
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		lines = append(lines, string(line.Value([]byte(source))))
	}
	
	return strings.Join(lines, "\n")
}

// ConvertToSpeech converts markdown elements to speakable text
func (mp *MarkdownProcessor) ConvertToSpeech(elements []MarkdownElement) []string {
	var speeches []string
	
	for _, elem := range elements {
		speech := mp.elementToSpeech(elem)
		if speech != "" {
			speeches = append(speeches, speech)
		}
	}
	
	return speeches
}

// elementToSpeech converts a single element to speech
func (mp *MarkdownProcessor) elementToSpeech(elem MarkdownElement) string {
	switch elem.Type {
	case ElementParagraph:
		return elem.Content
		
	case ElementHeading:
		// Add pause indicators for headers
		pause := strings.Repeat(".", elem.Level)
		return fmt.Sprintf("%s %s %s", pause, elem.Content, pause)
		
	case ElementCodeBlock:
		if mp.config.IncludeCodeBlocks {
			if elem.Language != "" {
				return fmt.Sprintf("Code block in %s: %s", elem.Language, elem.Content)
			}
			return fmt.Sprintf("Code block: %s", elem.Content)
		}
		return ""
		
	case ElementLink:
		return elem.Content
		
	case ElementImage:
		return elem.Content
		
	case ElementListItem:
		return fmt.Sprintf("Item: %s", elem.Content)
		
	case ElementBlockquote:
		return fmt.Sprintf("Quote: %s", elem.Content)
		
	case ElementEmphasis, ElementStrong:
		return elem.Content
		
	case ElementInlineCode:
		return elem.Content
		
	case ElementHorizontalRule:
		return "... ... ..." // Pause for horizontal rule
		
	default:
		return elem.Content
	}
}

// ExtractSpeakableSentences combines markdown processing with sentence extraction
func (mp *MarkdownProcessor) ExtractSpeakableSentences(markdown string) ([]ParsedSentence, error) {
	// Process markdown to elements
	elements, err := mp.ProcessMarkdown(markdown)
	if err != nil {
		return nil, err
	}
	
	// Convert elements to speech
	speeches := mp.ConvertToSpeech(elements)
	
	// Create sentence parser for splitting
	parser, err := NewSentenceParser(mp.config)
	if err != nil {
		return nil, err
	}
	
	var allSentences []ParsedSentence
	position := 0
	
	// Process each speech element
	for _, speech := range speeches {
		if speech == "" {
			continue
		}
		
		// Split into sentences
		sentences := parser.extractSentences(speech, markdown)
		
		// Update positions
		for i := range sentences {
			sentences[i].Position = position
			position++
		}
		
		allSentences = append(allSentences, sentences...)
	}
	
	return allSentences, nil
}

// CleanTextForSpeech applies final cleaning for TTS synthesis
func CleanTextForSpeech(text string) string {
	// Remove URLs in parentheses first
	text = regexp.MustCompile(`\(https?://[^\)]+\)`).ReplaceAllString(text, "")
	
	// Convert common symbols to words (order matters for compound symbols)
	replacements := []struct{old, new string}{
		{">=", " greater than or equal to "},
		{"<=", " less than or equal to "},
		{"==", " equals "},
		{"!=", " not equals "},
		{"->", " arrow "},
		{"=>", " arrow "},
		{"&&", " and "},
		{"&", " and "},
		{"@", " at "},
		{"#", " hash "},
		{"%", " percent "},
		{"^", " caret "},
		{"*", " star "},
		{"~", " tilde "},
		{"|", " pipe "},
		{"<", " less than "},
		{">", " greater than "},
	}
	
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.old, r.new)
	}
	
	// Clean up punctuation
	text = regexp.MustCompile(`([.!?])+`).ReplaceAllString(text, "$1")
	
	// Ensure proper spacing around punctuation
	text = regexp.MustCompile(`\s+([,.!?;:])`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`([.!?])\s*([a-zA-Z])`).ReplaceAllString(text, "$1 $2")
	
	// Remove multiple spaces (must be last)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	return strings.TrimSpace(text)
}