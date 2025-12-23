# GitHub Free Account Setup Guide

This guide helps you configure GitHub Actions workflows for free GitHub accounts.

## Common Issues with Free GitHub Accounts

### 1. **Access to Private Organization Repositories**

**Problem**: The default `GITHUB_TOKEN` does NOT have access to private organization repositories.

**Solution**: Create a Personal Access Token (PAT) with `repo` scope.

#### Steps:

1. **Create a Personal Access Token**:
   - Go to: https://github.com/settings/tokens
   - Click "Generate new token" → "Generate new token (classic)"
   - Name: `SYNTEGRITY_DAGGER_TOKEN`
   - Select scope: **`repo`** (full control of private repositories)
   - Click "Generate token"
   - **Copy the token immediately** (you won't see it again)

2. **Add Token as Secret**:
   - Go to your repository: `Settings` → `Secrets and variables` → `Actions`
   - Click "New repository secret"
   - Name: `SYNTEGRITY_DAGGER_TOKEN`
   - Value: Paste your PAT
   - Click "Add secret"

3. **Verify Access**:
   - The token must have access to: `github.com/getsyntegrity/go-kit-logger`
   - If the repository is in an organization, ensure your account has access

### 2. **GitHub Actions Minutes Limit**

**Free Account Limits**:
- **Private repositories**: 2,000 minutes/month
- **Public repositories**: Unlimited minutes
- **Concurrent jobs**: Limited (usually 1-2 jobs)

**Tips to Reduce Minutes**:
- Use caching effectively (already configured)
- Skip unnecessary jobs for feature branches
- Combine jobs when possible
- Use `if` conditions to skip jobs when not needed

### 3. **Concurrency Limits**

**Free Account Limits**:
- Limited concurrent jobs (usually 1-2)
- Jobs may queue if limit is reached

**Workflow Optimization**:
- Jobs run in parallel when possible
- Use `needs` to control job dependencies
- Skip jobs that aren't needed for the current pipeline type

### 4. **Sudo Commands**

**Problem**: Some workflows use `sudo` which may not be available or may cause issues.

**Solution**: The workflows have been optimized to avoid `sudo` where possible. If you encounter issues:
- Check if the command really needs `sudo`
- Use alternatives that don't require elevated privileges

## Troubleshooting

### Error: "Failed to download dependencies"

**Cause**: Token doesn't have access to private repository.

**Solution**:
1. Verify `SYNTEGRITY_DAGGER_TOKEN` secret is configured
2. Verify token has `repo` scope
3. Verify token has access to `github.com/getsyntegrity/go-kit-logger`
4. Check if repository is private and your account has access

### Error: "Workflow run limit exceeded"

**Cause**: You've exceeded the monthly minutes limit (2,000 for private repos).

**Solution**:
1. Check your usage: https://github.com/settings/billing
2. Wait for the next billing cycle
3. Consider making the repository public (unlimited minutes)
4. Optimize workflows to use fewer minutes

### Error: "Job queued" or "Waiting for runner"

**Cause**: Concurrency limit reached.

**Solution**:
1. Wait for other jobs to complete
2. Cancel unnecessary workflow runs
3. Consider upgrading to GitHub Pro (more concurrent jobs)

### Error: "Permission denied" or "sudo: command not found"

**Cause**: Workflow uses `sudo` which may not be available.

**Solution**: The workflows have been updated to avoid `sudo`. If you still see this:
1. Check the workflow file for any remaining `sudo` commands
2. Report the issue if it persists

## Best Practices

1. **Always use PAT for private repos**: Don't rely on `GITHUB_TOKEN` for private organization repositories
2. **Monitor minutes usage**: Check your usage regularly to avoid hitting limits
3. **Optimize workflows**: Use caching, skip unnecessary jobs, combine steps when possible
4. **Use conditional jobs**: Skip jobs that aren't needed for the current pipeline type
5. **Test locally first**: Run tests locally before pushing to reduce failed workflow runs

## Checking Your Setup

### Verify Token Access

You can test if your token has access by running:

```bash
# Replace YOUR_TOKEN with your PAT
git ls-remote --heads "https://x-access-token:YOUR_TOKEN@github.com/getsyntegrity/go-kit-logger.git"
```

If this succeeds, your token has access. If it fails, check:
- Token has `repo` scope
- Token hasn't expired
- Your account has access to the repository

### Check Workflow Logs

1. Go to your repository → `Actions` tab
2. Click on a failed workflow run
3. Check the "Validate GitHub account and token access" step
4. Look for error messages or warnings

## Additional Resources

- [GitHub Actions Limits](https://docs.github.com/en/actions/learn-github-actions/usage-limits-billing-and-administration)
- [Creating a Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Managing Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)

## Dependabot Configuration

**IMPORTANT**: Dependabot uses **separate secrets** from GitHub Actions!

Even if you've configured `SYNTEGRITY_DAGGER_TOKEN` for Actions, you **must also** configure it for Dependabot:

1. Go to: `Settings` → `Secrets and variables` → `Dependabot`
2. Add the same token as `SYNTEGRITY_DAGGER_TOKEN`
3. This allows Dependabot to access private repositories

See `docs/DEPENDABOT_SETUP.md` for detailed instructions.

## Support

If you continue to experience issues:
1. Check the workflow logs for specific error messages
2. Verify your token setup following this guide
3. For Dependabot issues, see `docs/DEPENDABOT_SETUP.md`
4. Check GitHub Actions status: https://www.githubstatus.com/
5. Review GitHub Actions documentation for your specific error

