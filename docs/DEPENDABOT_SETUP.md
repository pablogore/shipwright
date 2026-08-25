# Dependabot Setup Guide for Free GitHub Accounts

This guide helps you configure Dependabot to work with free GitHub accounts.

## Common Dependabot Errors

### Error: `git_dependencies_not_reachable`

**Problem**: Dependabot cannot access Git repositories (both public and private).

**Cause**: The authentication token is not configured or doesn't have the correct permissions.

## Solution: Configure Dependabot Secrets

Dependabot uses **separate secrets** from GitHub Actions. You need to configure the token in Dependabot secrets specifically.

### Step 1: Create a Personal Access Token (PAT)

1. Go to: https://github.com/settings/tokens
2. Click "Generate new token" → "Generate new token (classic)"
3. Name: `SHIPWRIGHT_TOKEN` (or any name you prefer)
4. Select scopes:
   - **`repo`** (full control of private repositories) - Required for private repos
   - **`read:packages`** (optional, for package registries)
5. Click "Generate token"
6. **Copy the token immediately** (you won't see it again)

### Step 2: Add Token to Dependabot Secrets

**IMPORTANT**: Dependabot secrets are **different** from Actions secrets!

1. Go to your repository
2. Navigate to: `Settings` → `Secrets and variables` → `Dependabot`
3. Click "New Dependabot secret"
4. Name: `SHIPWRIGHT_TOKEN`
5. Value: Paste your PAT
6. Click "Add secret"

### Step 3: Verify Configuration

After adding the secret, Dependabot should be able to:
- Access private repositories (e.g., `github.com/getsyntegrity/go-kit-logger`)
- Access public repositories (even though they're public, Dependabot may need authentication)
- Update dependencies successfully

## Differences: Dependabot Secrets vs Actions Secrets

| Feature | Actions Secrets | Dependabot Secrets |
|---------|----------------|-------------------|
| Location | `Settings` → `Secrets and variables` → `Actions` | `Settings` → `Secrets and variables` → `Dependabot` |
| Used by | GitHub Actions workflows | Dependabot only |
| Access | Can be used in workflows | Only accessible to Dependabot |
| Configuration | Same token can be used | Must be configured separately |

**Important**: Even if you have `SHIPWRIGHT_TOKEN` configured in Actions secrets, you **must also** add it to Dependabot secrets for Dependabot to work.

## Troubleshooting

### Error: "git_dependencies_not_reachable" for all dependencies

**Cause**: Token not configured in Dependabot secrets or token doesn't have correct permissions.

**Solution**:
1. Verify token is in Dependabot secrets (not just Actions secrets)
2. Verify token has `repo` scope
3. Verify token hasn't expired
4. Check if your account has access to the repositories

### Error: "git_dependencies_not_reachable" for private repos only

**Cause**: Token doesn't have access to private repositories.

**Solution**:
1. Ensure token has `repo` scope
2. Verify your account has access to the private repository
3. If repository is in an organization, ensure your account is a member

### Error: "git_dependencies_not_reachable" for public repos

**Cause**: Even public repos may require authentication in some cases, or there's a network issue.

**Solution**:
1. Ensure token is configured in Dependabot secrets
2. Check GitHub status: https://www.githubstatus.com/
3. Try again later (may be a temporary network issue)

### Dependabot not running at all

**Cause**: Dependabot may be disabled or not configured.

**Solution**:
1. Check if Dependabot is enabled: `Settings` → `Code security and analysis`
2. Verify `.github/dependabot.yml` exists and is valid
3. Check Dependabot logs: `Insights` → `Dependency graph` → `Dependabot`

## Configuration File

The Dependabot configuration is in `.github/dependabot.yml`:

```yaml
version: 2
registries:
  github-ep:
    type: git
    url: https://github.com
    username: x-access-token
    password: ${{ secrets.SHIPWRIGHT_TOKEN }}

updates:
  - package-ecosystem: "gomod"
    directory: "/"
    target-branch: "main"
    schedule:
      interval: "daily"
    registries:
      - github-ep
```

## Free Account Limitations

Free GitHub accounts have some limitations with Dependabot:

1. **Dependabot is available**: Free accounts can use Dependabot
2. **Private repos**: Require a PAT with `repo` scope
3. **Rate limits**: May apply if you have many dependencies
4. **Update frequency**: Daily updates are available

## Best Practices

1. **Use the same token for both Actions and Dependabot**: Configure `SHIPWRIGHT_TOKEN` in both:
   - Actions secrets (for GitHub Actions workflows)
   - Dependabot secrets (for Dependabot updates)

2. **Token security**:
   - Use a PAT with minimal required scopes (`repo` for private repos)
   - Rotate tokens regularly
   - Don't commit tokens to the repository

3. **Monitor Dependabot**:
   - Check Dependabot logs regularly
   - Review PRs created by Dependabot
   - Update dependencies manually if Dependabot fails

4. **Group updates**: The configuration groups updates to reduce the number of PRs:
   - Security updates: Daily
   - Minor/patch updates: Weekly
   - Major updates: Weekly (with manual review)

## Verification Checklist

- [ ] PAT created with `repo` scope
- [ ] Token added to Dependabot secrets (not just Actions secrets)
- [ ] `.github/dependabot.yml` configured correctly
- [ ] Dependabot enabled in repository settings
- [ ] Token has access to required repositories
- [ ] Dependabot can access both public and private repos

## Additional Resources

- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [Configuring Dependabot Secrets](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot#managing-encrypted-secrets-for-dependabot)
- [Dependabot Configuration Options](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/configuration-options-for-the-dependabot.yml-file)

## Support

If you continue to experience issues:

1. Check Dependabot logs: `Insights` → `Dependency graph` → `Dependabot`
2. Verify token configuration following this guide
3. Check GitHub status: https://www.githubstatus.com/
4. Review Dependabot documentation for your specific error

