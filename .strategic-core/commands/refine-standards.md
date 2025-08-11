---
description: Strategic Core: Customize project coding standards and best practices
---

# /refine-standards

## Purpose

Refine and customize the coding standards and best practices for your project after initial setup. This command helps you add, remove, or update standards based on your evolving needs.

**This command will also validate and fix all file paths in CLAUDE.md to ensure they point to actual files.**

## Prerequisites

- Strategic Core is initialized (`.strategic-core/` directory exists)
- You have already run `strategic init`
- Specs directory exists at `.strategic-core/specs/`

## Interactive Process

I'll help you refine your project standards through an interactive menu where you can:

### Available Actions

1. **View Current Standards** - See all active standards with version information
2. **Add New Standards** - Browse and add standards from the library
3. **Remove Standards** - Archive standards you no longer need
4. **Browse Available Standards** - Explore standards by category
5. **Search Standards** - Find standards by keyword
6. **Apply Template Sets** - Replace current standards with a pre-configured set
7. **View Version History** - See the history of your standards changes

### Standards Categories

- **Languages**: Python, TypeScript, Go, Rust, Java, etc.
- **Frameworks**: FastAPI, Django, React, Next.js, etc.
- **Practices**: Testing, Security, Git, API design, etc.
- **General**: Best practices that apply to any project

### How It Works

1. I'll analyze your current specs in `.strategic-core/specs/`
2. Show you available options based on what's in the standards library
3. Help you make informed decisions about what to add or remove
4. Update CLAUDE.md with correct file paths
5. Validate all paths are accurate

## Standards Versioning

All standards use versioned filenames: `YYYY-MM-DD-NNNN-standard-name.md`

This allows you to:
- Track when standards were added/modified
- Maintain history of changes
- Rollback if needed

## Example Workflow

```
Current Standards:
1. 2024-01-15-0001-python-style-black.md
2. 2024-01-15-0002-git-github-flow.md

What would you like to do?
> Add standards for API development

Available API Standards:
- api-rest: RESTful API design principles
- api-graphql: GraphQL best practices
- api-versioning: API versioning strategies

Which standards would you like to add?
> api-rest

✓ Added: RESTful API design principles
```

## Output

After refinement, your standards directory will be updated:

```
.strategic-core/standards/
├── active/          # Currently active standards
│   ├── 2024-01-15-0001-python-style-black.md
│   ├── 2024-01-15-0002-git-github-flow.md
│   └── 2024-01-16-0001-api-rest.md         # Newly added
└── archive/         # Previously used standards
    └── 2024-01-14-0001-python-style-pep8.md  # Archived
```

## Next Steps

After refining standards:
1. Review the newly added standards
2. Update your code to follow the new guidelines
3. Configure your tools (linters, formatters) accordingly

## Notes

- Standards are guidelines, not strict rules
- Customize them to fit your team's needs
- Archive (don't delete) standards for history
- You can always rollback to previous versions

## Implementation Instructions for Claude

When this command is invoked, follow these steps:

### Step 1: Analyze Current Standards

1. **Read current standards** from `.strategic-core/standards/`
2. **Identify tech stack** from existing standards
3. **Note versions** and last update dates
4. **Create snapshot** of current standards for comparison

### Step 2: Gather Project Information

1. **Scan project files** to detect actual technologies in use
2. **Check package files** (package.json, pyproject.toml, etc.)
3. **Analyze recent code** for patterns and practices
4. **Compare** detected tech with documented standards

### Step 3: Interactive Refinement

Present options based on findings:

```
📋 STANDARDS REFINEMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Current Standards Analysis:
✓ Python (Black formatter) - Up to date
⚠ Testing (Jest) - Project uses Vitest
✗ API Style (REST) - Detected GraphQL usage

Recommended Updates:
1. Replace Jest standards with Vitest
2. Add GraphQL API standards
3. Update state management (Redux → Zustand)

What would you like to do?
[A]pply all recommendations
[S]elect specific updates
[V]iew current standards
[B]rowse available standards
[Q]uit without changes
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Step 4: Apply Standards Updates

Based on user selection:

1. **Add new standards** to `.strategic-core/standards/`
2. **Archive replaced standards** (don't delete)
3. **Update tech-stack.md** with new components
4. **Track changes** for later comparison

### Step 5: Detect Standards Changes

After applying updates:

```python
# Compare before and after
changes_detected = compare_standards(before_snapshot, current_standards)

if changes_detected:
    # Extract what changed
    framework_changes = extract_framework_changes()
    library_changes = extract_library_changes()
    pattern_changes = extract_pattern_changes()
```

### Step 6: Product Documentation Alignment (If Standards Changed)

If significant changes were detected, offer to update product documentation:

```
📝 STANDARDS UPDATED - DOCUMENTATION SYNC AVAILABLE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Detected significant changes to project standards:
• Testing framework: Jest → Vitest
• API style: REST → GraphQL
• State management: Redux → Zustand

Would you like to update product documentation to
align with these new standards? (recommended)

This will update:
• 3 files in .strategic-core/product/
• Technical references and code examples
• Framework and library mentions

[Y]es, update docs / [N]o, skip / [P]review changes
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Step 7: Update Product Documentation (If User Confirms)

If user chooses to update:

#### 7.1 Create Backup
```bash
# Create timestamped backup
cp -r .strategic-core/product .strategic-core/product.backup.$(date +%Y%m%d_%H%M%S)
```

#### 7.2 Build Replacement Mappings
```python
replacements = {
    # Frameworks
    "Express.js": detected_framework,
    "Fastify": detected_framework,

    # Testing
    "Jest": detected_test_framework,
    "Vitest": detected_test_framework,

    # State Management
    "Redux": detected_state_mgmt,
    "Zustand": detected_state_mgmt,

    # API Style
    "REST API": detected_api_style,
    "GraphQL API": detected_api_style,

    # File paths
    "controllers/": detected_pattern,
    "resolvers/": detected_pattern
}
```

#### 7.3 Smart Update Process

For each file in `.strategic-core/product/`:

1. **Read entire file** for context
2. **Identify update targets**:
   - Technical framework names
   - Library references
   - Code examples
   - Configuration snippets
   - File path references
3. **Preserve unchanged content**:
   - Business requirements
   - User stories
   - Feature descriptions
   - Non-technical prose
4. **Apply updates** with context awareness
5. **Validate** markdown remains valid

#### 7.4 Show Progress
```
🔄 Updating Product Documentation...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Created backup: .strategic-core/product.backup.20240120_143022/
✓ Analyzing standards changes... 12 updates identified
✓ Scanning product docs... 5 files need updates

Updating files:
  → prd.md................... ✓ (3 sections updated)
  → architecture.md.......... ✓ (5 sections updated)
  → technical-spec.md........ ✓ (8 sections updated)
  → api-design.md............ ✓ (rewritten for GraphQL)
  → deployment-guide.md...... ✓ (2 sections updated)

✅ Documentation successfully aligned with new standards!

Summary:
• 5 files updated
• 21 technical references modernized
• 0 errors encountered
• Backup saved for rollback if needed
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Step 8: Validate and Fix CLAUDE.md Paths

**IMPORTANT**: Now I'll check and fix all file paths in CLAUDE.md to ensure they point to actual files:

```python
# Fix common path errors in CLAUDE.md
corrections_made = []

# Read CLAUDE.md
with open("CLAUDE.md", "r") as f:
    content = f.read()

# Apply corrections
original = content
content = content.replace(".strategic-core/standards/active/", ".strategic-core/specs/")
content = content.replace(".strategic-core/standards/", ".strategic-core/specs/")
content = content.replace("@.strategic-core/standards/", "@.strategic-core/specs/")

if content != original:
    with open("CLAUDE.md", "w") as f:
        f.write(content)
    corrections_made.append("Fixed standards → specs paths")

# Validate remaining paths
from pathlib import Path
import re

path_pattern = r'[@]?\.strategic-core/[^\s\)"\'\]]+'
for match in re.finditer(path_pattern, content):
    path_str = match.group().lstrip('@')
    if not Path(path_str).exists():
        print(f"⚠️ Invalid path found: {path_str}")

print(f"✅ CLAUDE.md updated with {len(corrections_made)} corrections")
```

Common fixes applied:
- `.strategic-core/standards/active/*` → `.strategic-core/specs/*`
- `.strategic-core/standards/*` → `.strategic-core/specs/*`
- Validates all remaining paths exist

### Step 9: Final Summary

Show what was accomplished:

```
✅ STANDARDS REFINEMENT COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Standards Updates:
✓ Added 3 new standards
✓ Updated 2 existing standards
✓ Archived 1 outdated standard

Documentation Sync:
✓ Updated 5 product documents
✓ Aligned all technical references
✓ Backup created for safety

Path Validation:
✓ CLAUDE.md paths verified
✓ Invalid paths corrected
✓ All references now valid

Next Steps:
1. Review the updated standards
2. Check product documentation changes
3. Update your code to follow new guidelines
4. Configure tools (linters, formatters) if needed

Use '/analyze-standards-fit' to check code compliance
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Update Examples

### Framework Updates
```markdown
OLD: The backend uses Express.js with middleware for routing
NEW: The backend uses Fastify with plugins for routing
```

### Testing Updates
```markdown
OLD: Tests are written using Jest with the following pattern
NEW: Tests are written using Vitest with the following pattern
```

### API Style Updates
```markdown
OLD: POST /api/users - Create a new user
NEW: mutation createUser - Create a new user
```

### Code Block Updates
```javascript
// OLD
describe('User API', () => {
  test('should create user', () => {
    // Jest test
  });
});

// NEW
describe('User API', () => {
  it('should create user', () => {
    // Vitest test
  });
});
```

## Important Notes

### What Gets Updated
- Technical framework and library names
- API patterns and conventions
- File structure references
- Configuration examples
- Code snippets in documentation
- Architecture pattern descriptions

### What Stays Preserved
- Business logic and requirements
- User stories and acceptance criteria
- Feature descriptions
- Timeline and milestone information
- Budget and resource mentions
- Non-technical content

### Safety Features
- Always create backup before updates
- Preview mode available (P option)
- Validation after updates
- Rollback capability from backup
- Skip ambiguous updates

### Edge Case Handling

#### Ambiguous References
When encountering ambiguous text like "state management", ask for clarification:
```
Found ambiguous reference: "React state management"
Could refer to:
1. Built-in React state (useState/useReducer)
2. Redux (old standard)
3. Zustand (new standard)

How should this be updated? [1/2/3/S]kip
```

#### Partial Matches
For partial matches, use word boundaries:
- Match: "Express.js server" → "Fastify server"
- Don't match: "expression" → "fastifyion"

#### Code Comments
Update comments in code blocks:
```javascript
// OLD: Setup Express middleware
// NEW: Setup Fastify plugins
```

#### No Product Docs
If `.strategic-core/product/` is empty or doesn't exist:
```
ℹ️ No product documentation found to update.
Create product docs with '/create-spec' or '/generate-spec'.
```

#### No Changes Detected
If standards didn't change meaningfully:
```
✓ Standards refined successfully.
No significant changes detected - documentation sync not needed.
```

## Workflow Guidance

After refining standards and syncing documentation:

```
📚 STANDARDS UPDATED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your project standards have been refined!

NEXT COMMAND SUGGESTIONS:

📊 /analyze-standards-fit
   • Check existing code compliance
   • Identify refactoring opportunities
   • Get specific improvement suggestions

🤖 /generate-agents (if major changes)
   • Regenerate agents with new standards
   • Ensure consistent code generation
   • Improve implementation quality

🔨 /execute-tasks
   • Continue implementation with new standards
   • Apply updated patterns consistently
   • Maintain code quality

📝 /create-spec (for new features)
   • Plan features using updated standards
   • Ensure consistency across the project
   • Document new patterns

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Alternative Method

You can also use the Strategic Core CLI directly:
```bash
strategic slash refine-standards
```

This provides the same functionality through the command line.
