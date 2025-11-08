---
description: Create and push a new release tag with auto-incremented version
allowed-tools: Bash(git checkout:*), Bash(git pull:*), Bash(git tag:*), Bash(git push:*), Bash(git log:*)
---

## Instructions

1. Check current branch and ensure we're on main:
   - Run `git branch --show-current`
   - If not on main, run `git checkout main`
   - If there are uncommited changes, stop and inform the user

2. Pull latest changes:
   - Run `git pull`
   - If there are conflicts or errors, stop and inform the user

3. Get the latest tag and increment version:
   - Run `git tag --sort=-version:refname | head -1` to get the latest tag
   - Parse the version (format: v0.X.0)
   - Increment the minor version (middle number) by 1
   - New version format: v0.(X+1).0

4. Get changes since last tag:
   - Run `git log <latest-tag>..HEAD --oneline` to get commit messages
   - Create a brief single line summary of changes (2-3 main points)
   - Prefix the message with the version, example: `v0.2.0 - Add authentication and fix memory leak`

5. Create and push the tag:
   - Run `git tag -a <new-version> -m "<summary of changes>"`
   - Run `git push origin <new-version>`

6. Report success to user with:
   - The new version number
   - Confirmation that the tag was pushed

## Notes

- Version format follows semantic versioning: vMAJOR.MINOR.PATCH
- This command increments the MINOR version (middle number)
- The tag message should be a brief summary of changes, not a full changelog
- After pushing the tag, GitHub Actions or other CI/CD may trigger automatic release builds
