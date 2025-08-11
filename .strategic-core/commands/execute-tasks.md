---
description: Strategic Core: Implement features following specifications
---

# /execute-tasks

## Purpose

Execute implementation tasks from a specification, focusing on one task at a time while following project standards and maintaining code quality. Automatically uses specialized agents when available for better results.

## Prerequisites

- Strategic Core is installed and initialized
- At least one specification exists in `.strategic-core/specs/`
- You're ready to implement code
- (Optional) Specialized agents generated via `/generate-agents`

## Execution Flow

1. Select specification to work on
2. Review task progress
3. Choose specific task
4. **Automatically use best agent for the task**
5. Implement task with specialized agent
6. Update task status

## Important Implementation Guidelines

⚠️ **NEVER SKIP PRE-IMPLEMENTATION CHECKS**
- Step 4 is mandatory for every task
- Review all checklists before coding
- Confirm understanding of requirements
- Identify dependencies and approach

📋 **USE TodoWrite TOOL FOR TASK TRACKING**
- Add ALL pre-implementation checklist items to TodoWrite
- Add ALL acceptance criteria as individual todos
- Add ALL validation steps to track completion
- Mark items as in_progress/completed as you work
- This ensures nothing is forgotten or skipped

## Process

### Step 1: Select Specification

I'll list available specifications and ask which one to work on:

```
Available specifications:
1. 2024-01-15-user-authentication
2. 2024-01-14-dashboard-redesign
3. 2024-01-10-api-endpoints

Which specification would you like to work on?
```

### Step 2: Review Current Progress

I'll check the tasks.md file in the selected specification folder:
- Completed tasks ✓
- In-progress tasks 🔄
- Pending tasks ⏳
- Blocked tasks 🚫

### Step 3: Task Selection

I'll ask which specific task to work on, showing:
- Task description
- Acceptance criteria
- Dependencies
- Technical notes

### Step 3.5: Agent Selection (Automatic)

I'll automatically check for and use specialized agents if available:

1. **Check for Specialized Agents**
   - Read `.strategic-core/agents/registry.md` if it exists
   - Read `.claude/agents/` directory for available agents
   - If agents exist, identify ones that match the task domain
   - Automatically select the most suitable agent for the task

2. **Seamless Agent Integration**
   - If a suitable agent is found, I'll mention it briefly:
     ```
     Using specialized agent: api-implementer
     ```
   - The agent will handle the implementation with domain expertise
   - **Always provide full file paths** to agents (e.g., `.strategic-core/specs/2025-01-04-project-name/tasks.md`)
   - Continue with task implementation using the agent's capabilities

## Agent Communication Protocol

**CRITICAL**: When calling sub-agents, use structured, specific prompts:

### For specialized agents:
- Be specific about what needs to be done
- Include file paths and concrete requirements
- Don't use vague requests like "please handle this"
- Provide clear context and expected outcomes

**Note**: If no agents are available, I'll implement the task using standard practices. Agents enhance the process but are not required.

### Step 4: Pre-Implementation Checks

**IMPORTANT: This step is MANDATORY and must not be skipped.**

Before starting ANY implementation, I MUST:

1. **Load Task Requirements into TodoWrite Tool**
   ```python
   # Use TodoWrite to track EVERYTHING:
   - Pre-implementation checklist items
   - Each acceptance criterion
   - Each validation step
   - Technical subtasks identified
   ```

2. **Pre-Implementation Checklist** (from task)
   ```
   ✓ Dependencies reviewed and available
   ✓ Acceptance criteria understood
   ✓ Technical approach decided
   ✓ Required files/modules identified
   ✓ Test approach planned
   ```

   I will explicitly go through each item, add to TodoWrite, and confirm readiness.

3. **Review Context**
   - Read specification details thoroughly
   - Check project standards and patterns
   - Review existing code for consistency
   - Identify potential integration points
   - Add any additional discovered tasks to TodoWrite

**Note**: If any checklist item cannot be confirmed, I must address it before proceeding with implementation. The TodoWrite tool ensures complete tracking and accountability.

### Step 5: Implementation

For the selected task, I will:

1. **Implement Solution**
   - Write code following standards
   - Add necessary tests
   - Update documentation

2. **Validate Implementation**
   - Ensure acceptance criteria are met
   - Run relevant tests
   - Check code quality

### Step 6: Task Completion

After implementation:
1. **Update TodoWrite Tool**
   - Mark all completed items as completed
   - Verify all acceptance criteria are met
   - Confirm all validation steps passed
   - Add any follow-up tasks discovered

2. **Complete Validation Checklist**
   - [ ] Code compiles/runs without errors
   - [ ] Tests pass
   - [ ] Standards compliance checked
   - [ ] Documentation updated if needed

3. **Update task status in tasks.md**:
   - Mark checkboxes as complete using [x]
   - Update task status
   - Note any issues or blockers

4. Review updated progress in both TodoWrite and tasks.md
5. Identify next logical task

## Example TodoWrite Usage

When starting a task like "Create HTML Structure", I would use TodoWrite like this:

```python
# Pre-Implementation Checks
- Review HTML5 semantic structure requirements
- Check existing UI patterns in codebase
- Identify required meta tags
- Plan file structure approach
- Review accessibility requirements

# Acceptance Criteria
- Valid HTML5 document structure created
- Semantic elements used appropriately
- Meta tags for viewport and charset included
- Accessibility attributes added
- Responsive viewport meta tag works

# Validation Steps
- HTML validates with W3C validator
- Page loads without errors
- Canvas element renders correctly
- Responsive viewport works on mobile
```

Each item gets tracked individually, marked as in_progress when working on it, and completed when done.

## Working Principles

### Code Standards
I'll follow your project's standards from:
- `.strategic-core/standards/tech-stack.md`
- `.strategic-core/standards/code-style.md`
- `.strategic-core/standards/best-practices.md`

### Focus Approach
- Work on ONE task at a time
- Complete it fully before moving on
- Don't skip ahead to other tasks
- Maintain context within the task

### Quality Checks
For each implementation:
- ✓ Meets acceptance criteria
- ✓ Follows code standards
- ✓ Includes appropriate tests
- ✓ Has necessary documentation
- ✓ Handles edge cases

## Output

For each task, I'll provide:
1. Implementation code
2. Test code (if applicable)
3. Documentation updates
4. Summary of changes
5. Next task recommendation

## Task Types

### Setup Tasks
- Environment configuration
- Dependency installation
- Initial file structure

### Implementation Tasks
- Core feature code
- Business logic
- UI components

### Testing Tasks
- Unit tests
- Integration tests
- E2E tests

### Documentation Tasks
- Code documentation
- API documentation
- User guides

## Next Steps

After completing a task:
1. Review the implementation
2. Run tests to verify
3. Choose next task or take a break
4. Update roadmap if needed

## Agent Matching Guide

When checking the agent registry, match tasks to agents based on:

### Task Keywords → Agent Mapping
- **API/endpoint/route** → api-implementer
- **test/spec/coverage** → test-specialist
- **UI/component/frontend** → frontend-builder
- **database/migration/schema** → database-migrator
- **refactor/cleanup/optimize** → refactoring-assistant
- **security/auth/permissions** → security-auditor
- **performance/speed/optimization** → performance-optimizer

### Example Matches
- "Create REST endpoint for user login" → api-implementer
- "Add unit tests for auth module" → test-specialist
- "Build dashboard component" → frontend-builder
- "Update database schema for new fields" → database-migrator

## Notes

- Stay focused on one task
- Ask for clarification if needed
- Maintain code quality throughout
- Celebrate small victories!
- Use specialized agents when available for better results
- **Always update task status after completing work**

## Workflow Guidance

After completing all tasks, provide this guidance:

```
✅ ALL TASKS COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Congratulations! All implementation tasks are finished.

NEXT COMMAND SUGGESTIONS:

✅ Verify your implementation:
   • Run your test suite
   • Check all acceptance criteria
   • Test edge cases

📦 /commit-work (recommended)
   • Organize changes into logical commits
   • Generate professional commit messages
   • Maintain clean git history
   • Link commits to tasks

📊 /analyze-standards-fit
   • Check code compliance with standards
   • Identify areas for improvement
   • Get specific refactoring suggestions

📝 /create-spec (if more features needed)
   • Plan the next feature
   • Create new specifications
   • Continue development momentum

🔧 /refactor-to-pure (optional)
   • Transform code to functional style
   • Improve testability
   • Reduce side effects
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
