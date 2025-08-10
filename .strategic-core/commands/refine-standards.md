---
description: Strategic Core: Customize project coding standards and best practices
---

# /refine-standards

## Purpose

Refine and customize the coding standards and best practices for your project after initial setup. This command helps you add, remove, or update standards based on your evolving needs.

## Prerequisites

- Strategic Core is initialized (`.strategic-core/` directory exists)
- You have already run `strategic init`
- Standards directory exists at `.strategic-core/standards/`

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

1. I'll analyze your current standards in `.strategic-core/standards/active/`
2. Show you available options based on what's in the standards library
3. Help you make informed decisions about what to add or remove
4. Maintain version history in `.strategic-core/standards/archive/`

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

## Alternative Method

You can also use the Strategic Core CLI directly:
```bash
strategic slash refine-standards
```

This provides the same functionality through the command line.
