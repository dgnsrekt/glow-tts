# Strategic Core Documentation Index

> Complete reference for all Strategic Core documentation

## 📚 Documentation Structure

```
.strategic-core/
├── IMPLEMENTATION_GUIDE.md          # Quick start guide for TTS implementation
├── DOCUMENTATION_INDEX.md           # This file - complete doc reference
├── commands/                        # Strategic Core commands
│   ├── analyze-product.md          # Analyze existing codebase
│   ├── analyze-standards-fit.md    # Check standards compliance
│   ├── create-spec.md              # Create feature specifications
│   ├── execute-tasks.md            # Execute implementation tasks
│   ├── generate-agents.md          # Generate AI agents
│   ├── generate-spec.md            # Generate specs from sources
│   ├── lint-spec.md                # Validate specifications
│   ├── plan-product.md             # Initialize new product
│   ├── refactor-to-pure.md         # Refactor to functional style
│   └── refine-standards.md         # Customize coding standards
├── ideas/                           # Ideas and lessons
│   ├── idea.md                     # Original TTS PRD
│   └── tts-lessons-learned.md      # Critical stdin race discovery ⚠️
├── product/                         # Product documentation
│   ├── mission.md                  # Project vision and goals
│   ├── current-state.md            # What exists now
│   ├── tech-stack.md               # Technology choices
│   └── decisions.md                # Architectural decisions
├── specs/                           # Feature specifications
│   └── 2025-01-10-tts-core-infrastructure/
│       ├── spec.md                 # Main specification
│       ├── tasks.md                # Implementation tasks (20 tasks)
│       └── sub-specs/
│           ├── technical-spec.md   # Technical design
│           └── tests.md            # Test requirements
└── standards/                       # Coding standards
    └── active/
        ├── 2025-01-10-0001-git-simple.md           # Git workflow
        ├── 2025-01-10-0002-general-best-practices.md # General practices
        ├── 2025-01-10-0003-go-code-style.md        # Go conventions
        ├── 2025-01-10-0004-bubble-tea-tui.md       # TUI patterns
        ├── 2025-01-10-0005-go-testing.md           # Testing strategy
        ├── 2025-01-10-0006-api-interface-design.md # Interface design
        ├── 2025-01-10-0007-go-concurrency.md       # Concurrency patterns
        ├── 2025-01-10-0008-documentation.md        # Documentation standards
        ├── 2025-01-10-0009-performance.md          # Performance optimization
        └── 2025-01-10-0010-subprocess-handling.md  # Subprocess patterns ⚠️
```

## 🚀 Quick Access Links

### Must Read First
1. **Lessons Learned**: `ideas/tts-lessons-learned.md` - Critical stdin race condition
2. **Implementation Guide**: `IMPLEMENTATION_GUIDE.md` - Quick start for developers
3. **Subprocess Handling**: `standards/active/2025-01-10-0010-subprocess-handling.md` - Prevent race conditions

### For Implementation
1. **Tasks**: `specs/2025-01-10-tts-core-infrastructure/tasks.md` - 20 implementation tasks
2. **Technical Spec**: `specs/2025-01-10-tts-core-infrastructure/sub-specs/technical-spec.md`
3. **Test Requirements**: `specs/2025-01-10-tts-core-infrastructure/sub-specs/tests.md`

### For Understanding
1. **Mission**: `product/mission.md` - Why we're building this
2. **Current State**: `product/current-state.md` - What exists now
3. **Decisions**: `product/decisions.md` - Why we chose this approach

## ⚠️ Critical Warnings

### The Stdin Race Condition

The experimental branch discovered a critical race condition when using `StdinPipe()` with programs that read stdin immediately (like Piper). This caused weeks of debugging.

**Never do this:**
```go
cmd.Start()
stdin := cmd.StdinPipe()  // RACE CONDITION!
stdin.Write(text)
```

**Always do this:**
```go
cmd.Stdin = strings.NewReader(text)
cmd.Run()
```

See `standards/active/2025-01-10-0010-subprocess-handling.md` for complete details.

## 📊 Documentation Statistics

- **10** Strategic Core commands
- **10** Active coding standards
- **20** Implementation tasks
- **4** Product documents
- **2** Ideas/lessons documents
- **1** Active specification
- **1** Implementation guide

## 🔄 Documentation Status

### Complete ✅
- All Strategic Core commands
- Product documentation (except roadmap)
- TTS specification with tasks
- All coding standards including subprocess handling
- Implementation guide
- Lessons learned from experimental branch

### To Be Created 📝
- Product roadmap (`product/roadmap.md`)
- Additional feature specifications as needed

## 💡 Key Takeaways

1. **The stdin race condition is real** - Always review subprocess handling standard
2. **Simple solutions work best** - Pre-configured stdin avoids complexity
3. **Caching is critical** - 80% hit rate mitigates process spawn overhead
4. **Documentation prevents mistakes** - Learn from experimental branch failures

## 🔗 External References

- **Strategic Core**: https://github.com/dgnsrekt/strategic-core
- **Glow**: https://github.com/charmbracelet/glow
- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **Piper TTS**: https://github.com/rhasspy/piper

---

*This index provides complete navigation of Strategic Core documentation for the Glow-TTS project.*