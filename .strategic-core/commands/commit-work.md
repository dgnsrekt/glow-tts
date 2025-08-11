---
description: Strategic Core: Organize and commit changes in logical chunks
---

# /commit-work

## Purpose

Intelligently organize and commit your changes in logical, atomic chunks after completing implementation tasks. This command ensures clean git history with meaningful commits that follow best practices.

## Prerequisites

- Git repository initialized
- Changes made during task implementation
- Task completion verified (tests passing, acceptance criteria met)

## When to Use

Run this command immediately after `/execute-tasks` when:
- You've completed implementing a task
- Multiple files have been changed
- You want clean, organized commit history
- You need to separate concerns (feature, tests, docs)

## Process

### Step 1: Analyze Changes

I'll examine all modified files to understand:
- What was implemented (features, fixes, tests, docs)
- File relationships and dependencies
- Logical groupings for commits
- Impact and scope of changes

### Step 2: Categorize Changes

Changes are organized into logical groups:

1. **Core Implementation**
   - Main feature/fix code
   - Related utility functions
   - Core business logic

2. **Tests**
   - Unit tests
   - Integration tests
   - Test fixtures and helpers

3. **Documentation**
   - Code comments and docstrings
   - README updates
   - API documentation

4. **Configuration**
   - Config file changes
   - Dependencies updates
   - Build/tooling changes

5. **Refactoring**
   - Code cleanup
   - Performance improvements
   - Style/formatting fixes

### Step 3: Create Commit Plan

I'll present a commit plan for your review:

```
📋 COMMIT PLAN
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Commit 1: feat: implement user authentication
  Files:
    • src/auth/login.py (new)
    • src/auth/validate.py (new)
    • src/models/user.py (modified)

Commit 2: test: add authentication unit tests
  Files:
    • tests/test_auth.py (new)
    • tests/fixtures/users.json (new)

Commit 3: docs: update README with auth endpoints
  Files:
    • README.md (modified)
    • docs/api.md (modified)

Review and confirm? [Y/n/edit]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Step 4: Generate Commit Messages

Each commit gets a properly formatted message following Conventional Commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests
- `chore`: Changes to build process or auxiliary tools

**Message includes:**
- Clear, concise subject line (50 chars max)
- Detailed body explaining what and why
- References to related issues/tasks
- Co-authorship attribution

### Step 5: Execute Commits

For each logical group:
1. Stage relevant files
2. Create commit with generated message
3. Run pre-commit hooks
4. Verify commit created successfully

### Step 6: Post-Commit Summary

After all commits:
```
✅ COMMITS CREATED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

3 commits created successfully:
• abc1234 feat: implement user authentication
• def5678 test: add authentication unit tests
• ghi9012 docs: update README with auth endpoints

Task Reference: #42 - User Authentication
Spec: .strategic-core/specs/2024-01-15-auth/

Ready to push? Use: git push origin <branch>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Smart Features

### 1. Task Context Integration
- Reads task description from last `/execute-tasks`
- Links commits to spec and task files
- Includes acceptance criteria references

### 2. Dependency Detection
- Identifies related files that should be committed together
- Prevents breaking commits by keeping dependencies atomic
- Warns about uncommitted dependencies

### 3. Test-Code Pairing
- Suggests committing tests with their implementation
- Or separates them based on project conventions
- Detects test coverage for new code

### 4. Automatic Formatting
- Ensures consistent commit message format
- Adds conventional commit prefixes
- Includes co-author attribution for AI assistance

### 5. Safety Checks
- Warns about large files
- Detects potential secrets/credentials
- Checks for debugging code (console.log, print statements)
- Validates all tests pass before committing

## Configuration Options

You can customize behavior by answering prompts:

1. **Commit Granularity**
   - Fine: Many small, atomic commits
   - Balanced: Logical groupings (default)
   - Coarse: Fewer, larger commits

2. **Message Style**
   - Conventional Commits (default)
   - GitHub style
   - Custom format

3. **Test Handling**
   - Separate test commits
   - Include with implementation
   - Ask each time

## Example Workflow

After completing a task with `/execute-tasks`:

```bash
# 1. Run commit-work command
/commit-work

# 2. Review proposed commit plan
# 3. Confirm or edit as needed
# 4. Commits are created automatically
# 5. Review git log to verify
git log --oneline -5

# 6. Push when ready
git push origin feature-branch
```

## Integration with TodoWrite

The command checks your TodoWrite list to:
- Verify all acceptance criteria were met
- Include task references in commit messages
- Mark implementation as fully committed
- Update task status to "committed"

## Best Practices

1. **Run after each task completion** - Don't let changes accumulate
2. **Review the plan** - Ensure logical groupings make sense
3. **Keep commits focused** - Each commit should have a single purpose
4. **Write for the future** - Messages should explain why, not just what
5. **Test before committing** - Ensure each commit leaves code in working state

## Advanced Features

### Fixup Support
If you need to fix a previous commit:
```
/commit-work --fixup abc1234
```

### Interactive Mode
For more control:
```
/commit-work --interactive
```
This lets you:
- Manually select files for each commit
- Edit messages before committing
- Reorder commits
- Squash related changes

### Stash Integration
Automatically stashes unrelated work:
- Identifies work not related to current task
- Stashes it before committing
- Reminds you to pop stash after

## Notes

- Works with any version control system (git)
- Respects .gitignore patterns
- Integrates with pre-commit hooks
- Preserves commit authorship
- Can be undone with standard git commands
