---
description: Strategic Core: Create a detailed specification for a new feature
---

# /create-spec

## Purpose

Create a detailed specification for a new feature, including user stories, technical design, implementation tasks, and test requirements.

## Prerequisites

- Strategic Core is installed and initialized
- Product documentation exists in `.strategic-core/product/`
- You have a feature idea to implement

## Process

### Step 1: Feature Selection

First, I'll check your roadmap and ask:

1. **What feature do you want to implement?**
2. **Why is this feature important?**
3. **Who will use this feature?**
4. **What problem does it solve?**

### Step 2: Create Feature Specification

I'll create `.strategic-core/specs/YYYY-MM-DD-feature-name/spec.md` with:

- Feature overview
- User stories
- Acceptance criteria
- Success metrics
- Dependencies
- Constraints

### Step 3: Technical Specification

I'll create `sub-specs/technical-spec.md` detailing:

- Architecture overview
- Component design
- Data flow
- Integration points
- Performance considerations
- Security considerations

### Step 4: API Specification (if applicable)

For features with APIs, I'll create `sub-specs/api-spec.md`:

- Endpoint definitions
- Request/response formats
- Authentication requirements
- Error handling
- Rate limiting

### Step 5: Database Schema (if applicable)

For features requiring data changes, I'll create `sub-specs/database-schema.md`:

- New tables/collections
- Schema modifications
- Relationships
- Indexes
- Migration strategy

### Step 6: Test Specification

I'll create `sub-specs/tests.md` outlining:

- Unit test scenarios
- Integration test cases
- End-to-end test flows
- Performance test criteria
- Test data requirements

### Step 7: Implementation Tasks

Finally, I'll create `tasks.md` with numbered tasks:

1. Setup tasks
2. Implementation tasks (ordered)
3. Testing tasks
4. Documentation tasks
5. Deployment tasks

## Output

Complete specification structure:

```
.strategic-core/specs/YYYY-MM-DD-feature-name/
├── spec.md                 # Main specification
├── tasks.md               # Implementation tasks
└── sub-specs/
    ├── technical-spec.md  # Technical design
    ├── api-spec.md        # API details (if needed)
    ├── database-schema.md # Data changes (if needed)
    └── tests.md           # Test requirements
```

## Task Structure

Each task in `tasks.md` follows this format:

```markdown
## Task 1: [Task Title]

**Type**: setup|implementation|testing|documentation
**Priority**: high|medium|low
**Estimated Hours**: X

### Pre-Implementation Checklist
- [ ] Dependencies reviewed and available
- [ ] Acceptance criteria understood
- [ ] Technical approach decided
- [ ] Required files/modules identified
- [ ] Test approach planned

### Description
[What needs to be done]

### Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

### Validation Steps
- [ ] Code compiles/runs without errors
- [ ] Tests pass
- [ ] Standards compliance checked
- [ ] Documentation updated if needed

### Technical Notes
[Any helpful details]
```

## Next Steps

After specification is complete:
1. Review all generated documents
2. Refine requirements as needed
3. **Generate specialized agents** for implementation (recommended)
4. Use `/execute-tasks` to start implementation
5. Track progress against tasks

### Agent Generation Prompt

After creating the specification, I'll provide workflow guidance:

```
✨ SPECIFICATION COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your specification has been created successfully!

RECOMMENDED NEXT STEPS:

1. Review and refine your project standards (optional)
   Use: /refine-standards
   - Customize coding standards for this feature
   - Add feature-specific guidelines
   - Update best practices

2. Generate specialized AI agents for this spec (recommended)
   Use: /generate-agents
   - Task tracking and progress management
   - Implementation of specific components
   - Test creation and validation
   - Code review and optimization

3. Start implementation
   Use: /execute-tasks
   - Implement features following the spec
   - Track progress automatically
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Notes

- Specifications are living documents
- Update them as requirements change
- Use them to guide implementation
- Reference them in code comments
