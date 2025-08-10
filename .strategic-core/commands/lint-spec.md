---
description: Strategic Core: Validate specification format and completeness
---

# Command: /lint-spec

## Description
Lint and validate feature specifications to ensure they follow the correct format and contain all required information.

## Usage
```
/lint-spec [spec-name]
```

## Arguments
- `spec-name` (optional): Name of a specific spec to lint. If not provided, all specs will be linted.

## What It Does
1. **Format Checking**:
   - Verifies all required sections are present
   - Checks for recommended sections
   - Identifies empty sections
   - Finds TODO placeholders

2. **Content Validation**:
   - Ensures acceptance criteria use checklist format
   - Verifies implementation tasks are properly formatted
   - Checks markdown formatting consistency

3. **Project Validation**:
   - Validates technical stack matches project
   - Checks dependencies exist in project

## Lint Rules

### Error Level
- **missing-required-section**: A required section is missing
- **file-not-found**: Specified spec file doesn't exist

### Warning Level
- **empty-section**: A section has no content
- **todo-placeholder**: Found TODO/FIXME placeholder text
- **no-acceptance-criteria**: Acceptance criteria has no checklist items
- **no-implementation-tasks**: Implementation tasks section has no tasks

### Info Level
- **missing-recommended-section**: A recommended section is missing
- **multiple-blank-lines**: Too many consecutive blank lines
- **heading-spacing**: No blank line before heading

## Examples

### Lint all specs
```
/lint-spec
```

### Lint specific spec
```
/lint-spec user-authentication
```

## Output Example
```
Linting user-authentication.md...
╭─ Spec Lint Results - user-authentication.md ─╮
│ ✅ Found 3 issues: 0 errors, 2 warnings       │
╰───────────────────────────────────────────────╯
┏━━━━━━┳━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ Line ┃ Severity ┃ Rule             ┃ Message                                ┃
┡━━━━━━╇━━━━━━━━━━╇━━━━━━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┩
│ 10   │ warning  │ todo-placeholder │ TODO placeholder found: [TODO: Define] │
│      │          │                  │ 💡 Replace placeholder with content    │
│ 45   │ warning  │ empty-section    │ Empty section: ## Dependencies         │
│      │          │                  │ 💡 Add content or remove section       │
└──────┴──────────┴──────────────────┴────────────────────────────────────────┘

==================================================
✅ All specs passed! (1 files, 2 warnings)
```

## Tips
- Run regularly during spec development to catch issues early
- Address all errors before implementation begins
- Warnings can be addressed based on context
- Use info-level issues as suggestions for improvement

## Next Steps
After linting passes:
1. Use `/execute-tasks` to implement the spec
2. Keep spec updated as implementation progresses
3. Re-lint after making spec changes
