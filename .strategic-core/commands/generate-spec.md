---
description: Strategic Core: Generate spec from external sources (URLs, repos)
---

# /generate-spec

Generate feature specifications from analyzed projects.

## Usage

```
/generate-spec
```

## Description

This command analyzes a project (local or web-based) and generates:
- Feature specifications with technical requirements
- Project overview specifications
- Task breakdowns based on detected patterns

## Process

1. **Source Selection**: Choose a project to analyze
   - Local directory (default: current project)
   - GitHub repository URL
   - Documentation website URL

2. **Analysis**: The system analyzes:
   - Technology stack (languages, frameworks)
   - Code standards and practices
   - Project structure and patterns
   - Dependencies and tools

3. **Spec Generation**: Choose what to generate:
   - **Feature Spec**: Detailed specification for a new feature
   - **Project Overview**: High-level project analysis and structure
   - **Both**: Generate both types

## Generated Specs Include

### Feature Specifications
- Functional and non-functional requirements
- Technical design and architecture
- Implementation guidelines based on detected standards
- Testing strategy
- Task breakdown
- Acceptance criteria

### Project Overview
- Technology stack breakdown
- Directory structure
- Development practices
- Detected standards and patterns
- Entry points and API patterns

## Examples

```bash
# Generate a feature spec for the current project
/generate-spec

# The command will prompt for:
# - Source to analyze (default: current directory)
# - Type of spec to generate
# - Feature details (if generating feature spec)
```

## Output

Specs are saved to `.strategic-core/specs/` with timestamps:
- `20240115_143022_user_authentication.md` (feature spec)
- `project_overview.md` (project overview)

## Benefits

- Ensures new features follow existing patterns
- Documents technical decisions based on analysis
- Creates consistent specification format
- Saves time on requirements gathering
- Provides clear implementation guidelines
