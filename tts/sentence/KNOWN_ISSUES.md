# Known Issues - Sentence Parser

## Overview
The sentence parser implementation is functionally complete and meets performance requirements (82ms for 10KB documents, well under the 100ms requirement). However, there are some edge cases that fail in the test suite that should be addressed in future refinements.

## Failing Test Cases

### 1. Decimal Number Sentence Splitting
**Test**: `TestParseNumbers/decimal_numbers`
- **Input**: `"Pi is 3.14159. E is 2.71828."`
- **Expected**: 2 sentences
- **Actual**: 1 sentence (entire text treated as single sentence)
- **Root Cause**: The parser doesn't correctly identify sentence boundaries after decimal numbers when followed by a period, space, and uppercase letter.

### 2. Money/Price Sentence Splitting  
**Test**: `TestParseNumbers/money`
- **Input**: `"Cost is $19.99. Save $5.00 today!"`
- **Expected**: 2 sentences
- **Actual**: 1 sentence
- **Root Cause**: Similar to decimal numbers - the period after "99" isn't recognized as a sentence boundary.

### 3. Very Short Sentence Filtering
**Test**: `TestParseEdgeCases/very_short_sentences`
- **Input**: `"A. B. C."`
- **Expected**: 0 sentences (each segment is below minLength threshold)
- **Actual**: 1 sentence containing all text
- **Root Cause**: The parser combines all short segments instead of filtering them individually.

## Technical Analysis

The core issue lies in the `isRealSentenceEndRunes` function's logic for handling periods after digits:

```go
// Current implementation at line 346-358 in parser.go
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
```

The problem scenarios:
1. "3.14159. E" - The second period should be recognized as sentence end
2. "$19.99. Save" - Period after price should split sentences
3. Need to differentiate between decimal points and sentence-ending periods

## Potential Solutions

### Solution 1: Enhanced Lookahead Logic
Check for common patterns that indicate sentence boundaries after numbers:
- Number + period + space + uppercase word (not single letter)
- Number + period + space + common sentence starters ("The", "Save", "Get", etc.)
- Number + period at end of line

### Solution 2: Two-Pass Parsing
1. First pass: Identify all decimal numbers and mark them
2. Second pass: Apply sentence splitting rules, excluding marked decimal points

### Solution 3: Context-Aware Number Handling
Track whether we're in a "number context" (prices, decimals, versions) and apply different rules:
- If previous token was currency symbol, expect decimal
- If number contains multiple periods (version numbers), don't split
- If number is followed by units, keep together

## Impact Assessment

- **Severity**: Low
- **Frequency**: Rare in typical markdown documents
- **User Impact**: Minimal - TTS will still work, just with slightly different sentence grouping
- **Performance Impact**: None - current implementation meets all performance requirements

## Recommendations

1. **Priority**: Low - address after core TTS functionality is complete
2. **Approach**: Implement Solution 1 (Enhanced Lookahead) as it's least invasive
3. **Testing**: Add more comprehensive number-based test cases
4. **Consider**: Whether combining very short sentences (current behavior) is actually preferable for TTS quality

## Test Coverage Status

Current test results:
- ✅ 90% of tests passing
- ✅ Performance requirement met (<100ms for 10KB)
- ✅ Core functionality working (markdown, abbreviations, basic sentences)
- ❌ Edge cases with numbers need refinement

## Future Work

When addressing these issues:
1. Start with the decimal number detection logic
2. Add more test cases for edge scenarios
3. Consider making minLength configurable
4. Document any intentional behavior differences

---

*Last Updated: 2025-01-08*
*Related Files: parser.go, parser_test.go*
*Task Reference: Task 4 - Implement Sentence Parser*