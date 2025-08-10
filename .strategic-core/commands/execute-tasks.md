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

Before starting implementation, I'll verify:

1. **Pre-Implementation Checklist** (from task)
   - [ ] Dependencies reviewed and available
   - [ ] Acceptance criteria understood
   - [ ] Technical approach decided
   - [ ] Required files/modules identified
   - [ ] Test approach planned

2. **Review Context**
   - Read specification details
   - Check project standards
   - Review existing code

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
1. **Complete Validation Checklist**
   - [ ] Code compiles/runs without errors
   - [ ] Tests pass
   - [ ] Standards compliance checked
   - [ ] Documentation updated if needed

2. **Update task status in tasks.md**:
   - Mark checkboxes as complete using [x]
   - Update task status
   - Note any issues or blockers

3. Review updated progress
4. Identify next logical task

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

RECOMMENDED NEXT STEPS:

1. Analyze code quality and standards compliance
   Use: /analyze-standards-fit
   - Check how well your code matches project standards
   - Identify areas for improvement
   - Get recommendations for refactoring

2. Run tests and verify functionality
   - Execute your test suite
   - Verify all features work as expected
   - Check edge cases

3. Review and document
   - Update documentation if needed
   - Add code comments where helpful
   - Create user guides if applicable
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
