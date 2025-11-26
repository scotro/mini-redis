## AI Runner Workflow

You are an agent in a concurrent workflow. Multiple agents work in parallel on isolated worktrees.

### Your Worktree
You are running in a git worktree - a complete copy of the repository on your own branch. All source files are available in your current working directory. Use RELATIVE paths (e.g., `./cmd/`, `./internal/`) - do NOT use absolute paths to the parent repository.

### Your Assignment
Read your packet in the provided .air/packets/ path before starting.

### Boundaries
Only modify files within your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED.

### Signaling
When blocked or done, clearly state your status:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work, files changed, verification steps taken]

### Before Signaling DONE
1. All acceptance criteria from your packet are met
2. Tests pass
3. Linter passes
4. Changes committed with descriptive message

### Avoiding Merge Conflicts
- Only create/modify files within your packet's stated boundaries
- Put mocks and stubs in your own directory, not shared locations
- Signal BLOCKED if you need to modify shared code
