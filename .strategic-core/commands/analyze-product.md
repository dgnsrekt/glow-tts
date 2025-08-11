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

### Step 0: Check Ideas Folder

First, I'll check `.strategic-core/ideas/` for any new feature plans or requirements:

```
Scanning for:
- Feature specifications
- Enhancement proposals
- Bug fix requirements
- Architecture changes
- Any materials describing desired changes
```

This helps me understand:
- What new features you're planning to add
- How existing code should evolve
- Gaps between current implementation and vision

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

## Ideas Integration

If materials exist in `.strategic-core/ideas/`:
- **Feature Gaps**: I'll identify what needs to be built
- **Enhancement Opportunities**: Where current code can improve
- **Architecture Evolution**: How structure should change
- **Implementation Roadmap**: Path from current to desired state

## Output

After analysis, you'll have:

```
.strategic-core/
└── product/
    ├── mission.md          # Inferred project purpose
    ├── current-state.md    # What exists now (no speculation)
    ├── tech-stack.md       # Detected technologies
    ├── decisions.md        # Observed patterns
    └── gaps-analysis.md    # If ideas folder has content
```

The documentation will reflect:
- Current implementation (from code analysis)
- Future direction (from ideas folder)
- Clear distinction between what exists and what's planned

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

## Workflow Guidance

After analysis is complete, I'll provide this guidance:

```
📊 CODEBASE ANALYSIS COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your existing codebase has been documented!

NEXT COMMAND SUGGESTIONS:

📋 Review generated documentation:
   • Verify current-state.md accuracy
   • Check tech-stack.md completeness
   • Update mission.md if needed

🎯 /refine-standards
   • Align standards with detected patterns
   • Add missing coding guidelines
   • Standardize existing practices

📝 /create-spec
   • Plan new features or improvements
   • Address technical debt
   • Implement missing functionality

🤖 /generate-agents
   • Create agents for your tech stack
   • Get specialized help for refactoring
   • Improve code quality systematically
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Next Steps

With your codebase documented:
1. Review and correct any misinterpretations
2. Identify highest priority improvements
3. Plan systematic enhancements

## Notes

- Analysis is based on code patterns and structure
- Some interpretation may be needed
- Documentation should be refined based on actual intent
- Use this as a starting point, not final documentation
