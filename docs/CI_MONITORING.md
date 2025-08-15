# CI Monitoring Dashboard

Quick reference for monitoring CI/CD health and performance.

## Workflow Status

| Workflow | Badge | Purpose | Schedule |
|----------|-------|---------|----------|
| Build & Test | [![Build & Test](https://github.com/dgnsrekt/glow-tts/actions/workflows/build.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/build.yml) | Quick build verification | On push/PR |
| Test Suite | [![Test Suite](https://github.com/dgnsrekt/glow-tts/actions/workflows/test.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/test.yml) | Comprehensive testing | On push/PR |
| Static Analysis | [![Static Analysis](https://github.com/dgnsrekt/glow-tts/actions/workflows/static-analysis.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/static-analysis.yml) | Code quality checks | On push/PR |
| PR Checks | [![PR Checks](https://github.com/dgnsrekt/glow-tts/actions/workflows/pr.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/pr.yml) | PR validation | On PR events |
| Nightly | [![Nightly](https://github.com/dgnsrekt/glow-tts/actions/workflows/nightly.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/nightly.yml) | Extended testing | Daily 2 AM UTC |

## Performance Metrics

### Target Execution Times

| Workflow | Target | Acceptable | Critical |
|----------|--------|------------|----------|
| PR Checks | < 3 min | < 5 min | > 10 min |
| Unit Tests | < 2 min | < 3 min | > 5 min |
| Build (per platform) | < 1 min | < 2 min | > 3 min |
| Static Analysis | < 5 min | < 7 min | > 10 min |
| Full Test Suite | < 10 min | < 15 min | > 20 min |

## Health Indicators

### ✅ Healthy CI System
- All badges showing "passing"
- Average workflow time < 10 minutes
- No repeated failures on main branch
- Test coverage > 70%

### ⚠️ Warning Signs
- Flaky tests (intermittent failures)
- Workflow times increasing trend
- Multiple retries needed
- Deprecation warnings in logs

### ❌ Critical Issues
- Main branch failing for > 24 hours
- Security vulnerabilities unpatched
- CI completely blocked
- Test coverage dropping below 60%

## Quick Actions

### Check Recent Failures
```bash
# View recent workflow runs
gh run list --limit 10

# View specific workflow details
gh run view <run-id>

# Download logs
gh run download <run-id>
```

### Rerun Failed Workflows
```bash
# Rerun failed jobs
gh run rerun <run-id> --failed

# Rerun all jobs
gh run rerun <run-id>
```

### Monitor Performance Trends
1. Go to [Actions tab](https://github.com/dgnsrekt/glow-tts/actions)
2. Click on a workflow name
3. View "Workflow runs" timing chart
4. Check for performance degradation

## Notification Channels

| Channel | Status | Configuration |
|---------|--------|---------------|
| GitHub Issues | ✅ Enabled | Auto-created for critical failures |
| Discord | ⚪ Optional | Set `DISCORD_WEBHOOK_URL` variable |
| Slack | ⚪ Optional | Set `SLACK_WEBHOOK_URL` variable |
| Email | ✅ Default | GitHub user notification settings |

## Common Issues & Solutions

### Issue: Workflows Timing Out
**Solution:** Check for hanging tests, reduce parallelism, or increase timeout

### Issue: OOM (Out of Memory) Errors
**Solution:** Reduce test data size, fix memory leaks, use `--parallel=1`

### Issue: Flaky Platform-Specific Tests
**Solution:** Use mock audio, add retries, improve test isolation

### Issue: Cache Corruption
**Solution:** Clear GitHub Actions cache via UI or API

## Maintenance Checklist

### Daily
- [ ] Check main branch status
- [ ] Review any CI failure issues
- [ ] Monitor workflow execution times

### Weekly
- [ ] Review and close resolved CI issues
- [ ] Check for workflow deprecation warnings
- [ ] Update dependencies if needed

### Monthly
- [ ] Analyze performance trends
- [ ] Review and optimize slow tests
- [ ] Update CI documentation
- [ ] Clean up old workflow runs

## Useful Links

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Workflow Syntax](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [Actions Status Page](https://www.githubstatus.com/)
- [CI Setup Guide](CI_SETUP.md)
- [Workflow README](.github/workflows/README.md)