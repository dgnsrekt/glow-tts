---
description: Strategic Core: Initialize product with mission, roadmap, and architecture
---

# /plan-product

## Purpose

Initialize Strategic Core for a new product by creating foundational documentation that defines your product's mission, roadmap, technology choices, and architectural decisions.

## Prerequisites

- Strategic Core is installed (`.strategic-core/` directory exists)
- You are starting a new project (not an existing codebase)
- You have a clear idea of what you want to build

## Interactive Process

This command will guide you through an interactive process to gather information about your product.

### Step 1: Understand the Product Vision

First, I need to understand your product vision. Please tell me:

1. **What problem does your product solve?**
2. **Who are your target users?**
3. **What makes your solution unique?**
4. **What are your main goals?**

### Step 2: Create Mission Documentation

Based on your responses, I'll create `.strategic-core/product/mission.md` that includes:

- Product vision statement
- Target audience definition
- Core values and principles
- Success metrics
- Key differentiators

### Step 3: Define the Roadmap

Next, I'll create `.strategic-core/product/roadmap.md` with:

- Development phases
- Feature priorities
- Milestone definitions
- Timeline estimates
- MVP scope

### Step 4: Document Technology Decisions

I'll create `.strategic-core/product/tech-stack.md` documenting:

- Programming languages
- Frameworks and libraries
- Database choices
- Infrastructure decisions
- Third-party services

### Step 5: Initialize Decision Log

Finally, I'll create `.strategic-core/product/decisions.md` to track:

- Major architectural decisions
- Technology choices rationale
- Trade-offs considered
- Future considerations

## Output

After running this command, you'll have:

```
.strategic-core/
└── product/
    ├── mission.md      # Product vision and goals
    ├── roadmap.md      # Development priorities
    ├── tech-stack.md   # Technology choices
    └── decisions.md    # Architectural decisions
```

## Next Steps

Once complete, you can:
1. Review and refine the generated documentation
2. Use `/create-spec` to plan your first feature
3. Use `/execute-tasks` to start building

## Notes

- All generated files can be edited and customized
- These documents guide future development
- Keep them updated as your product evolves
