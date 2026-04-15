---
name: commit-push-pr
description: "Commit staged files only, generate message/PR, push to new branch"
disable-model-invocation: true
allowed-tools: ["Bash(git *)", "Bash(gh *)", "Read", "Grep"]
---

1. **Verify staged changes**: Run `git diff --cached`. If empty, stop and tell the user: "No staged files. Stage with `git add` first."

2. **Analyze the diff**: Read the staged diff carefully to understand what changed and why. This informs the commit message and PR body.

3. **Branch**: If `$ARGUMENTS` is provided, use it as the branch name. Otherwise, ask the user. Run `git checkout -b <branch-name>`.

4. **Commit**: Generate a Conventional Commit message from the staged diff following CLAUDE.md conventions (`type(scope): description`). Run `git commit` using a heredoc for the message.

5. **Push**: Run `git push -u origin <branch-name>`.

6. **Create PR**: Read `.github/PULL_REQUEST_TEMPLATE.md`, fill it in based on the commit(s), then run:
   ```
   gh pr create --title "<title>" --body "<filled template>"
   ```

Confirm with the user before each destructive step (commit, push, PR creation).
