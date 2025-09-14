# Pipeline Test Documentation

This file is created to test the automated release pipeline system.

## Test Scenarios

1. **Feature Test**: This commit should trigger a minor version bump
2. **Pipeline Verification**: Test the conventional commits detection
3. **Release Automation**: Verify automatic release creation

## Expected Behavior

- Push to develop → Pre-release (v0.1.0-beta.1)
- Merge to main → Stable release (v0.1.0)

