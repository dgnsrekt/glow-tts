---
description: Strategic Core: Check how well code matches project standards
---

# /analyze-standards-fit

## Description

Analyze your project and get recommendations for which standards from the Strategic Core library would be a good fit based on your technology stack and development patterns.

## Usage

```
/analyze-standards-fit
```

## What it does

1. **Analyzes your project** - Scans your codebase to detect:
   - Programming languages used
   - Frameworks and libraries
   - Development patterns (Docker, CI/CD, etc.)
   - Project structure and type

2. **Matches against standards library** - Compares your project with 40+ available standards:
   - Language-specific standards (Python, JavaScript, Go, Rust)
   - Framework standards (FastAPI, React, Next.js)
   - Practice standards (API design, security, databases)
   - Architecture patterns (Clean, Microservices)

3. **Calculates fit scores** - Each standard gets a score based on:
   - Language match (40% weight)
   - Framework match (30% weight)
   - Pattern match (20% weight)
   - Category relevance (10% weight)

4. **Shows recommendations** - Displays top 15 standards with:
   - Fit score percentage
   - Reason for recommendation
   - Current installation status

## Example Output

```
🔍 Standards Analysis

Analyzing project structure...

💡 Recommended Standards
┌──────────────────────────────┬───────────────────┬──────────┬────────────────────┬─────────────┐
│ Standard                     │ Category          │ Fit Score │ Reason             │ Status      │
├──────────────────────────────┼───────────────────┼──────────┼────────────────────┼─────────────┤
│ Python Code Style (Black)    │ languages/python  │    90%    │ Uses python        │ ✓ Installed │
│ FastAPI Best Practices       │ frameworks/fastapi│    70%    │ Uses fastapi       │ Not installed│
│ REST API Design              │ practices/api     │    60%    │ API project        │ ✓ Installed │
│ PostgreSQL Best Practices    │ practices/database│    50%    │ Uses postgres      │ Not installed│
└──────────────────────────────┴───────────────────┴──────────┴────────────────────┴─────────────┘

📊 Summary: 2 recommended standards already installed, 2 could be added

To add missing standards, use:
  /refine-standards
```

## Next Steps

After running this command:

1. Review the recommendations
2. Use `/refine-standards` to add any missing standards you want
3. Re-run periodically as your project evolves

## Notes

- The command analyzes your actual code, not just configuration files
- Scores are based on common patterns and best practices
- Not all recommendations may be relevant - use your judgment
- Standards marked as "Installed" are already in your project
