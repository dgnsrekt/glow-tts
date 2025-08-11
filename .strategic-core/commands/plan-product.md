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

### Step 0: Check Ideas Folder

First, let me check if you have any materials in your ideas folder that can inform the planning:

```
Checking .strategic-core/ideas/ for:
- Requirements documents
- Mockups and wireframes
- Research notes
- Technical specifications
- Any other inspiration materials
```

If I find materials in the ideas folder, I'll analyze them to better understand your vision before asking questions.

### Step 1: Understand the Product Vision

Based on any ideas found (or starting fresh), I need to understand your product vision. Please tell me:

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

## Ideas Integration

If you have materials in `.strategic-core/ideas/`:
- I'll extract key requirements and constraints
- Mockups will inform UI/UX decisions
- Technical specs will guide architecture choices
- All insights will be incorporated into the generated documentation

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

All documentation will be informed by:
- Your interactive responses
- Materials from the ideas folder (if present)
- Best practices for your technology choices

## Workflow Guidance

After planning is complete, I'll provide this guidance:

```
🎯 PRODUCT PLANNING COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your product foundation has been established!

NEXT COMMAND SUGGESTIONS:

📋 Review the documentation:
   • Check mission.md aligns with your vision
   • Verify roadmap.md priorities
   • Confirm tech-stack.md choices

🎯 /refine-standards
   • Choose coding standards for your tech stack
   • Add project-specific guidelines
   • Set quality expectations

📝 /create-spec
   • Plan your first feature from the roadmap
   • Create detailed specifications
   • Define implementation tasks

🤖 /generate-agents (after creating specs)
   • Generate AI agents for your project
   • Get specialized implementation help
   • Improve development efficiency
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Next Steps

The generated documentation serves as your project foundation:
1. Review and refine all documents
2. Start with the highest priority feature
3. Maintain documentation as the project evolves

## Notes

- All generated files can be edited and customized
- These documents guide future development
- Keep them updated as your product evolves
