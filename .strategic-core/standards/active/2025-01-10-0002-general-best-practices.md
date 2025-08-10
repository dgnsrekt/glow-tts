# Development Best Practices

> Default development guidelines and principles
> Customize this file to match your team's philosophy

## Core Principles

### 1. Keep It Simple (KISS)
- Write code that's easy to understand
- Avoid clever solutions when simple ones work
- Optimize for readability over performance (unless critical)

### 2. Don't Repeat Yourself (DRY)
- Extract common functionality
- Use configuration over duplication
- But don't over-abstract too early

### 3. You Aren't Gonna Need It (YAGNI)
- Build only what's needed now
- Avoid speculative features
- Refactor when requirements emerge

### 4. Single Responsibility
- Each module/class/function does one thing
- Easy to test and understand
- Clear separation of concerns

## Pre-Commit Hooks (MANDATORY)

### Setup Requirements
**CRITICAL**: No features should be built until pre-commit hooks are configured. This is non-negotiable.

1. **Install pre-commit** as the first step in any project:
```bash
pip install pre-commit
# or with other package managers
brew install pre-commit
```

2. **Create `.pre-commit-config.yaml`** with minimum requirements:
```yaml
repos:
  - repo: local
    hooks:
      - id: lint
        name: Lint code
        entry: <your-linter-command>
        language: system
        pass_filenames: false
      - id: typecheck
        name: Type check
        entry: <your-typecheck-command>
        language: system
        pass_filenames: false
      - id: tests
        name: Run tests
        entry: <your-test-command>
        language: system
        pass_filenames: false
```

3. **Install the hooks**:
```bash
pre-commit install
```

### Usage Rules
- **NEVER use `--no-verify`** without explicit authorization from team lead
- **Fix issues immediately** when pre-commit blocks your commit
- **Update hooks regularly** with `pre-commit autoupdate`
- **Run on all files** periodically: `pre-commit run --all-files`

### Minimum Hook Requirements
Every project MUST have:
1. **Linting/Formatting** - Code style consistency
2. **Type Checking** - Catch type errors early (for typed languages)
3. **Test Suite** - Ensure tests pass before commit
4. **Security Scanning** - Basic vulnerability checks

## Development Workflow

### Before Writing Code
1. **Verify pre-commit hooks** are installed and working
2. **Understand the requirement** fully
3. **Check existing code** for similar patterns
4. **Plan the approach** (design before coding)
5. **Consider edge cases** upfront

### While Writing Code
1. **Write tests first** (TDD) when appropriate
2. **Commit frequently** with clear messages
3. **Refactor continuously** as you go
4. **Document decisions** in code

### After Writing Code
1. **Self-review** before requesting review
2. **Run all tests** locally
3. **Update documentation** if needed
4. **Clean up** debug code and comments

## Code Quality

### Functions
- Do one thing well
- Have descriptive names
- Limit parameters (ideally < 4)
- Return early for error cases
- Avoid side effects when possible

### Error Handling
- Fail fast and explicitly
- Provide helpful error messages
- Log errors with context
- Handle errors at the right level
- Never silence errors without reason

### Testing Strategy
- **Unit Tests**: Test individual functions
- **Integration Tests**: Test component interaction
- **E2E Tests**: Test critical user paths
- **Performance Tests**: Test under load
- **Security Tests**: Test for vulnerabilities

### Code Comments
- Explain WHY, not WHAT
- Document complex algorithms
- Note assumptions and limitations
- Link to relevant documentation
- Keep them up to date

## Security Practices

### Input Validation
- Never trust user input
- Validate on the backend
- Sanitize for the context (SQL, HTML, etc.)
- Use allowlists over denylists

### Authentication & Authorization
- Use proven libraries
- Hash passwords properly (bcrypt, argon2)
- Implement proper session management
- Check permissions on every request

### Data Protection
- Encrypt sensitive data
- Use HTTPS everywhere
- Implement rate limiting
- Log security events
- Regular security audits

## Performance Guidelines

### Database
- Add indexes for frequent queries
- Avoid N+1 query problems
- Use pagination for large datasets
- Optimize slow queries
- Monitor query performance

### Caching
- Cache expensive computations
- Use appropriate TTLs
- Invalidate cache properly
- Monitor cache hit rates
- Don't cache personal data

### Frontend
- Lazy load when appropriate
- Optimize images and assets
- Minimize bundle sizes
- Use CDN for static assets
- Monitor Core Web Vitals

## API Design

### RESTful Principles
- Use proper HTTP methods
- Return appropriate status codes
- Version your APIs
- Provide clear error messages
- Document all endpoints

### Response Format
```json
{
  "data": { /* actual response */ },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0"
  },
  "errors": []
}
```

### Pagination
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

## Deployment & DevOps

### Version Control
- **Setup pre-commit hooks FIRST** (see Pre-Commit Hooks section above)
- Branch from main/master
- Use feature branches
- Commit frequently with descriptive messages
- **NEVER use `git commit --no-verify`** without explicit team lead approval
- Squash commits when merging (if using squash merge strategy)
- Delete branches after merge
- Tag releases with semantic versioning

### CI/CD Pipeline
- Automated testing on every push
- Linting and formatting checks
- Security vulnerability scanning
- Automated deployments
- Rollback capability

### Monitoring
- Application performance monitoring
- Error tracking and alerting
- Log aggregation
- Uptime monitoring
- User analytics

## Team Collaboration

### Code Reviews
- Review promptly
- Be constructive
- Focus on the code, not the person
- Suggest improvements
- Approve when good enough

### Documentation
- Keep README up to date
- Document setup procedures
- Maintain architecture diagrams
- Write runbooks for operations
- Create onboarding guides

### Communication
- Over-communicate in remote settings
- Document decisions
- Share knowledge freely
- Ask questions early
- Celebrate successes

## Technical Debt

### Managing Debt
- Track it explicitly
- Allocate time to pay it down
- Refactor opportunistically
- Don't let it accumulate
- Communicate impact

### When to Refactor
- Before adding new features
- When fixing bugs in the area
- When performance degrades
- When patterns emerge
- During planned maintenance

## Learning & Growth

### Stay Updated
- Follow industry trends
- Read documentation
- Attend conferences/meetups
- Contribute to open source
- Share knowledge with team

### Experiment Safely
- Use feature flags
- A/B test changes
- Have rollback plans
- Monitor experiments
- Learn from failures

---

*This is a template. Evolve it based on your team's experiences and learnings.*
