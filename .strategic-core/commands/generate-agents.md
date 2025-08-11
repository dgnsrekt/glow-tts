---
description: Strategic Core: Generate specialized AI agents for your project
---

# /generate-agents

## Purpose

Analyze your project specifications and generate specialized AI agents tailored to your implementation needs. These agents provide focused expertise for specific development tasks.

## Prerequisites

- Strategic Core is installed and initialized
- Project specifications exist in `.strategic-core/specs/`
- You've completed initial planning and specification phases

## Execution Instructions for Claude

When this command is invoked, follow these steps in order:

1. **Gather documentation** for detected frameworks and tools
2. **Analyze** the project specs and tech stack
3. **Present a summary** of agents to be generated
4. **Ask for confirmation** before proceeding
5. **Generate agents** only after user approval
6. **Create symlink** for Claude integration
7. **Show verification instructions** for the user to test agents

## Workflow Integration

This command works best when run immediately after `/create-spec`. The agents will be tailored to your specification's requirements, ensuring optimal coverage for implementation tasks.

## Process

### Step 0: Check for Recent Specifications

I'll first check if you have recently created specifications:

```
🔍 CHECKING FOR SPECIFICATIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Looking for recent specifications to tailor agent generation...
```

If a spec was created in this session or recently (within last 24 hours), I'll prioritize agents for that specific feature.

### Step 0.5: Documentation Gathering

Before analyzing the project, I'll fetch relevant documentation:

```
📚 GATHERING DOCUMENTATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Fetching documentation for informed agent generation...
```

Documentation to fetch (adapt based on project):

**Always fetch:**
- Claude Code Sub-agents docs: `WebFetch("https://docs.anthropic.com/en/docs/claude-code/sub-agents", "sub-agent format and configuration")`
- Project README: `Read("README.md")` if exists
- Package manifest: `Read("package.json")`, `Read("pyproject.toml")`, `Read("Cargo.toml")`, etc.

**Framework-specific (detect first, then fetch relevant docs):**
- **React/Next.js**: Component patterns, hooks, server components
- **Vue/Nuxt**: Composition API, components, SSR patterns
- **FastAPI/Django**: Route handlers, models, dependency injection
- **Express/NestJS**: Middleware, controllers, decorators
- **Rails/Sinatra**: MVC patterns, Active Record, helpers
- **Go (Gin/Echo/Fiber)**: Handlers, middleware, goroutines
- **Rust (Actix/Rocket)**: Handlers, extractors, async patterns

**Testing frameworks (if detected):**
- Jest/Vitest, Pytest, RSpec, Go testing, Rust tests

**Database/ORM (if detected):**
- Prisma, SQLAlchemy, Django ORM, ActiveRecord, GORM, Diesel

**IMPORTANT**: Wait for all documentation fetches to complete before proceeding to Step 1.

### Step 1: Project Analysis

I'll analyze your project to understand:
- Technology stack and frameworks
- Specification patterns and requirements
- Task complexity and domains
- Existing project structure

### Step 2: Agent Planning & Summary

I'll provide a summary of agents to be generated:

```
📋 AGENT GENERATION SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Based on your project analysis, I'll create:

Recommended Agents (Based on Your Stack):
  [List of 4-6 agents specific to the project]

Total: [X] specialized agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Common agent types include:
- **api-implementer**: For REST/GraphQL endpoint implementation
- **test-specialist**: For comprehensive test coverage
- **database-migrator**: For schema changes and migrations
- **frontend-builder**: For UI component development
- **refactoring-assistant**: For code quality improvements
- **security-auditor**: For security analysis and fixes
- **performance-optimizer**: For optimization tasks

### Step 3: Confirmation

Before proceeding, I'll ask for your confirmation to generate these agents. You can:
- Approve all agents
- Request changes to the agent list
- Cancel the generation

### Step 4: Agent Generation

Once confirmed, for each agent I'll:
1. Create a tailored agent definition with your project context
2. Configure appropriate tools and permissions
3. Define trigger patterns for automatic suggestion
4. Set up capability tracking
5. Generate a registry.md file listing all agents

### Step 5: Integration Setup

The generated agents will be:
- Saved in `.strategic-core/agents/implementations/`
- Made accessible via a directory symlink: `.claude/agents` → `.strategic-core/agents/implementations`
- Registered in the agent registry for discovery
- Configured with your project standards

### Step 6: Path Validation

After generation, I'll validate and fix paths in CLAUDE.md:

```python
# Ensure CLAUDE.md references are accurate
from pathlib import Path

claude_md = Path("CLAUDE.md")
if claude_md.exists():
    content = claude_md.read_text()
    original = content

    # Fix common path errors
    content = content.replace(".strategic-core/standards/", ".strategic-core/specs/")
    content = content.replace("@.strategic-core/standards/", "@.strategic-core/specs/")

    if content != original:
        claude_md.write_text(content)
        print("✓ Fixed CLAUDE.md paths to match actual locations")
```

### Step 7: Verification

After generation and path validation, I'll provide verification instructions:

```
✅ AGENTS GENERATED SUCCESSFULLY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

To verify your agents are properly registered:

1. Close Claude Code (Ctrl+C or Cmd+C)
2. Start Claude Code again
3. Run the /agents command
4. You should see all your new agents listed with their colors

If agents don't appear, check:
- Symlink exists: ls -la .claude/agents
- Agent files exist: ls .strategic-core/agents/implementations/
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

IMPORTANT: Create a single directory symlink, NOT individual file symlinks:
```bash
# Correct approach (what you should do):
ln -s ../.strategic-core/agents/implementations .claude/agents

# Wrong approach (do NOT do this):
ln -s ../../.strategic-core/agents/implementations/agent1.md .claude/agents/agent1.md
```

## Agent Structure

Each agent MUST use Claude's YAML frontmatter format:

```yaml
---
name: agent-name
description: Brief description of the agent's purpose
tools: [Read, Write, Edit, MultiEdit, Grep, Glob, Bash]
model: inherit
color: red  # One of: red, blue, green, yellow, purple, orange, pink, cyan, brown, gray
---

# Agent prompt starts here
You are the agent-name agent...
```

Key requirements:
- **YAML frontmatter** between `---` markers (NOT in a code block)
- **Required fields**: name, description, tools, model (always "inherit"), color
- **NO deprecated fields**: Do not include version, triggers, or capabilities
- **Color assignment**: Assign colors in order from the list above

## Usage After Generation

Once agents are generated, they can be used in two ways:

### 1. Automatic Triggering
Agents will suggest themselves when you use trigger phrases:
- "implement the user endpoint" → api-implementer activates
- "write tests for auth module" → test-specialist activates

### 2. Explicit Invocation
You can explicitly request an agent:
- "Use the api-implementer agent to create the products endpoint"
- "Have the test-specialist write integration tests"

## Communication Protocol

### Context Handoff
When an agent is invoked, I'll prepare a focused context package including:
- Specific task description
- Relevant specification sections
- Current implementation state
- Expected deliverables

### Agent Reports
Agents return structured summaries with:
- Completed work (what was accomplished)
- Key decisions (technical choices and rationale)
- Issues encountered (blockers or concerns)
- Next steps (recommended follow-up)
- Files modified (with descriptions)

## Session Continuity

The agent system maintains session continuity:
- Context is preserved between agent invocations
- Progress is tracked across sessions
- Agent history is maintained for reference
- Performance metrics are collected

## Best Practices

1. **Generate agents after specifications are complete** - Agents work best with clear requirements
2. **Start with high-priority agents** - Focus on agents that match your immediate needs
3. **Customize as needed** - Generated agents can be edited in `.strategic-core/agents/`
4. **Monitor effectiveness** - Check agent metrics in the registry
5. **Update agents as project evolves** - Regenerate when requirements change significantly

## Example Interaction

```
You: /generate-agents

Claude: I'll generate specialized agents for your project. Let me start by gathering relevant documentation...

📚 GATHERING DOCUMENTATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Fetching documentation for informed agent generation...

[Fetches Claude Code docs, README, package.json, framework docs...]

Now analyzing your project based on the documentation...

📋 AGENT GENERATION SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Based on your React/FastAPI project, I'll create:

Recommended Agents (Based on Your Stack):
  ✓ api-implementer - FastAPI endpoints with Pydantic models
  ✓ frontend-builder - React components with TypeScript
  ✓ test-specialist - Pytest and Jest test coverage
  ✓ database-migrator - Alembic migrations with SQLAlchemy
  ✓ refactoring-assistant - Code quality improvements

Total: 5 specialized agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Would you like me to generate these agents? (yes/no)
```

## Implementation Instructions for Claude

### Option 1: Use CLI Tool (Preferred if available)
If the Strategic Core CLI is available, you can use:
```bash
strategic agents generate --auto
```
This will automatically handle all formatting and symlink creation correctly.

### Option 2: Manual Implementation
When implementing this command manually:

1. **Use proper YAML frontmatter format** - Agents won't work without it
2. **Assign colors sequentially** from: red, blue, green, yellow, purple, orange, pink, cyan, brown, gray
3. **Create directory symlink** using: `ln -s ../.strategic-core/agents/implementations .claude/agents`
4. **Include self-identification** in each agent prompt: "You are the {agent-name} agent"
5. **Set model: inherit** for all agents (required field)
6. **Don't include deprecated fields** like version, triggers, or capabilities
7. **Create registry.md file** in `.strategic-core/agents/` listing all generated agents with descriptions

### Registry File Example:
```markdown
# Agent Registry

## Generated Agents

1. **api-implementer** - Implements REST/GraphQL endpoints
2. **test-specialist** - Creates comprehensive test coverage
3. **frontend-builder** - Builds UI components and interfaces
4. **database-migrator** - Handles schema changes and migrations
5. **refactoring-assistant** - Code quality improvements

Generated on: [DATE]
Total agents: 5
```

### Correct Agent Example:
```yaml
---
name: api-implementer
description: Implements REST/GraphQL API endpoints with proper validation
tools: [Read, Write, Edit, MultiEdit, Grep, Glob]
model: inherit
color: red
---

You are the api-implementer agent, specialized in creating robust API endpoints.

## Core Responsibilities
- Implement REST or GraphQL endpoints
- Add proper input validation and error handling
- Create Pydantic models or TypeScript types
- Follow RESTful conventions
- Add appropriate status codes
- Include comprehensive error responses

## Implementation Approach

1. **Read existing code structure** to understand patterns
2. **Follow project conventions** for routing and validation
3. **Implement with security in mind** - validate all inputs
4. **Add proper documentation** - OpenAPI/Swagger annotations
5. **Include error handling** - try/catch blocks, proper status codes

## Best Practices
- Use dependency injection where appropriate
- Implement rate limiting for public endpoints
- Add logging for debugging
- Follow project's authentication patterns
- Write integration tests for endpoints
```

## Workflow Guidance

After generating agents, provide this guidance:

```
🤖 AGENTS GENERATED SUCCESSFULLY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Specialized agents have been created for your project!

⚠️ IMPORTANT: Reload Claude Code to activate agents
After reloading, your agents will be available.

NEXT STEPS (IMPORTANT):

⚠️ Reload Claude Code (required):
   • Close Claude Code completely
   • Restart Claude Code
   • Agents will be automatically loaded

Then continue with:

🔍 Verify agents loaded:
   • Type: /agents
   • Should list all your new agents
   • Each with their specialization

🔨 /execute-tasks
   • Start implementing with agent assistance
   • Agents auto-activate based on context
   • Better code quality guaranteed

🎯 /refine-standards (optional)
   • Customize standards for your tech stack
   • Add project-specific guidelines
   • Ensure consistency across agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
