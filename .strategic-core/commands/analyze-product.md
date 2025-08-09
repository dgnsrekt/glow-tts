---
description: Strategic Core: Analyze existing codebase and generate documentation
---

# /analyze-product

## Purpose

Analyze an existing codebase and create Strategic Core documentation based on **what currently exists**, without inventing or suggesting new features unless explicitly requested.

## Prerequisites

- Strategic Core is installed (`.strategic-core/` directory exists)
- You have an existing codebase to analyze
- The codebase has some structure (not just scripts)

## Process

### Step 1: Codebase Analysis

I'll analyze your project to understand:

1. **Project Structure**
   - Directory organization
   - File naming patterns
   - Module organization

2. **Technology Stack**
   - Programming languages used
   - Frameworks and libraries
   - Build tools and dependencies
   - Database systems

3. **Code Patterns**
   - Architecture style
   - Design patterns
   - Code organization

4. **Existing Documentation**
   - README files
   - API documentation
   - Code comments

### Step 2: Generate Mission Documentation

Based on the analysis, I'll create `.strategic-core/product/mission.md` inferring:

- Apparent purpose from code structure
- Target users from features
- Core functionality
- Project scope

### Step 3: Document Current State

I'll document in `.strategic-core/product/current-state.md`:

- **Existing Features Only** - What's actually implemented
- **Work in Progress** - Based on TODO comments and branches
- **Known Issues** - From existing issue tracking

**Note**: I will NOT suggest new features or improvements unless you explicitly ask.

### Step 4: Document Detected Tech Stack

I'll create `.strategic-core/product/tech-stack.md` listing:

- All detected technologies
- Version information where available
- Dependencies and their purposes
- Development tools in use

### Step 5: Extract Architectural Decisions

I'll create `.strategic-core/product/decisions.md` documenting:

- Inferred architectural patterns
- Technology choices evident in code
- Patterns that should be followed
- Areas needing clarification

## Output

After analysis, you'll have:

```
.strategic-core/
└── product/
    ├── mission.md          # Inferred project purpose
    ├── current-state.md    # What exists now (no speculation)
    ├── tech-stack.md       # Detected technologies
    └── decisions.md        # Observed patterns
```

### Optional: Future Planning

If you want suggestions for improvements, you can ask me to:
- Generate a `roadmap.md` with potential enhancements
- Create a `improvements.md` with technical debt items
- Suggest new features based on patterns

**Important**: These will only be created if you explicitly request them.

## Review Process

After generation:
1. **Review each file** for accuracy
2. **Fill in gaps** where analysis couldn't determine intent
3. **Correct any misinterpretations**
4. **Add missing context**

## Next Steps

Once documentation is reviewed:
1. Use `/create-spec` to plan new features
2. Use `/execute-tasks` to implement changes
3. Keep documentation updated going forward

## Notes

- Analysis is based on code patterns and structure
- Some interpretation may be needed
- Documentation should be refined based on actual intent
- Use this as a starting point, not final documentation
