# Agent Instructions

This project uses **GitHub issues** for issue tracking. Use the `gh` CLI.

## Quick Reference

```bash
gh issue list --state open                    # Find available work
gh issue list --label P1 --state open         # Highest-priority work first
gh issue view <number>                        # View issue details
gh issue create --title "..." --body "..."    # File new work
gh issue close <number> --comment "..."       # Complete work
```

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var

## Issue Tracking with GitHub Issues

**IMPORTANT**: This project uses **GitHub issues** for ALL issue tracking. Do NOT use markdown TODOs, task lists in files, or other tracking methods.

### Labels

**Priority** (every issue gets one):
- `P0` - Critical (security, data loss, broken builds)
- `P1` - High (major features, important bugs)
- `P2` - Medium (default, nice-to-have)
- `P3` - Low (polish, optimization)
- `P4` - Backlog (future ideas)

**Type**:
- `bug` - Something broken
- `enhancement` - New functionality
- `task` - Work item (tests, docs, refactoring, chores)

### Dependencies

Issues that cannot start until other work lands say so in the first line of the body: `Blocked by #123`. Before picking up an issue, check its body for blockers and skip it if any referenced issue is still open. When filing follow-up work discovered while implementing an issue, reference the source: `Discovered while working on #123`.

### Workflow for AI Agents

1. **Find work**: `gh issue list --state open`, pick the highest-priority issue that has no open blockers
2. **Work on it**: Implement, test, document — on a branch (`feat/`, `fix/`, or `chore/` prefix)
3. **Discover new work?** File a linked issue with `gh issue create`
4. **Complete**: Open a PR with `Fixes #<number>` in the body so the issue closes automatically on merge

### Important Rules

- ✅ Use GitHub issues for ALL task tracking
- ✅ Add `--json` flags to `gh` commands for programmatic use
- ✅ Link PRs to issues with `Fixes #<number>` / `Closes #<number>`
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT duplicate tracking systems

## Landing the Plane (Session Completion)

**When ending a work session**, complete ALL steps below. Work is NOT complete until it is pushed and a PR exists.

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Push the branch and open a PR** - Reference the issues it fixes:
   ```bash
   git push -u origin <branch>
   gh pr create --fill --body "Fixes #<number>"
   ```
4. **Verify** - All changes committed AND pushed; `git status` clean
5. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until the branch is pushed - never leave work stranded locally
- Never push directly to main - all changes go through PRs
- If push fails, resolve and retry until it succeeds
