# Bug Fix Plan - Glow-TTS (Revised with Documentation Insights)

## 🔴 Priority 1: Critical Fixes (Immediate)

### Fix #1: Piper Command Conflict Resolution ⚡ CRITICAL DISCOVERY
**Bug**: Piper process dies immediately after starting
**Root Cause**: Using both `--output-raw` AND `--output-file -` causes Piper to fail
**Impact**: No real TTS, only mock tone works

**The Problem**:
- Documentation shows: When using `--output-raw`, Piper outputs to stdout automatically
- Our code adds BOTH `--output-raw` and `--output-file -` which creates a conflict
- This is why the process dies immediately\!

**Solution**: Remove the conflicting `--output-file -` argument

**Implementation**:
```go
// In both piper.go:233 and piper_v2.go:230, REMOVE this line:
// args = append(args, "--output-file", "-")  // DELETE THIS LINE\!

// The startProcess function should only have:
if e.config.OutputRaw {
    args = append(args, "--output-raw")
    // NO --output-file argument here\!
}
```

**Quick Test**:
```bash
# This works:
echo "Test" | piper --model ~/piper-voices/en_US-amy-medium.onnx --output-raw

# This fails (our current bug):
echo "Test" | piper --model ~/piper-voices/en_US-amy-medium.onnx --output-raw --output-file -
```

### Fix #2: Enable Fresh Process Mode (Already Works\!)
**Discovery**: PIPER_FRESH_MODE is already implemented and should work once Fix #1 is applied

```bash
# Just enable it - code already exists:
export PIPER_FRESH_MODE=true
./glow-tts -t test.md
```

### Fix #3: TUI Auto-Detection
**Bug**: Requires -t flag to keep TUI open
**Solution**: Auto-enable pager when terminal detected

**Implementation** in main.go around line 316:
```go
// Auto-enable pager for terminal + file
case pager || cmd.Flags().Changed("pager") || 
     (term.IsTerminal(int(os.Stdout.Fd())) && len(args) > 0):
    // Run TUI pager
```

## 🟡 Priority 2: UI Fixes

### Fix #4: Sentence Highlighting Visibility
**Bug**: Indicator not showing in viewport
**Solution**: Add above viewport, not in content

**Implementation** in ui/pager.go:356:
```go
func (m pagerModel) View() string {
    var b strings.Builder
    
    // Add TTS indicator FIRST (above viewport)
    if m.tts \!= nil && m.tts.IsEnabled() {
        sentence := m.tts.GetCurrentSentence()
        if sentence >= 0 {
            indicator := fmt.Sprintf("🔊 Playing: Sentence %d of %d", 
                sentence+1, m.tts.GetTotalSentences())
            b.WriteString(lipgloss.NewStyle().
                Background(lipgloss.Color("226")).
                Foreground(lipgloss.Color("0")).
                Bold(true).
                Padding(0, 1).
                Render(indicator))
            b.WriteString("\n")
        }
    }
    
    // Then viewport
    fmt.Fprint(&b, m.viewport.View())
    fmt.Fprint(&b, "\n")
    
    // Then status bar
    m.statusBarView(&b)
    // ... rest
}
```

## 🟢 Priority 3: Verification & Polish

### Step 1: Test Piper Directly
```bash
#\!/bin/bash
# test_piper_fix.sh

echo "Testing Piper with correct arguments..."
echo "Hello world" | piper \
    --model ~/piper-voices/en_US-amy-medium.onnx \
    --output-raw > /tmp/test.pcm

if [ $? -eq 0 ]; then
    echo "✅ Piper works\! Generated $(wc -c < /tmp/test.pcm) bytes"
    # Play it back
    aplay -r 22050 -f S16_LE -t raw /tmp/test.pcm
else
    echo "❌ Piper failed"
fi
```

### Step 2: Implementation Order (15 minutes total)

1. **[2 min]** Remove `--output-file -` from piper.go and piper_v2.go
2. **[1 min]** Test with PIPER_FRESH_MODE=true
3. **[5 min]** Add TUI auto-detection to main.go
4. **[5 min]** Fix highlighting indicator in pager.go
5. **[2 min]** Test full workflow

## 📊 Success Metrics

- ✅ Real speech from Piper (not mock tones)
- ✅ Sentence indicator visible during playback
- ✅ TUI opens without -t flag
- ✅ Play/pause/stop work reliably
- ✅ No process crashes

## 🔍 Root Cause Analysis

The main issue was a **command-line argument conflict** in Piper:
- `--output-raw` tells Piper to output raw PCM to stdout
- `--output-file -` tells Piper to output to stdout (redundant and conflicting)
- Using both causes Piper to fail immediately

This explains why:
- Process dies after starting (conflicting arguments)
- Fresh mode didn't help (same argument issue)
- Manual testing works (we don't add the extra argument)

## 💡 Lessons Learned

1. Always test external commands with exact arguments
2. Read documentation carefully for command-line tools
3. Simple exec per request is often more stable than process pools
4. Mock engines are valuable for isolating issues

## 🚀 Next Steps After Fixes

1. Remove process pooling complexity (use fresh mode)
2. Add streaming support for large documents
3. Implement word-level highlighting (v2.0)
4. Add voice selection UI
5. Cache generated audio for replay
EOF < /dev/null