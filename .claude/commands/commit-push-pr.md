---
description: "Commit staged files only, generate message/PR, push to new branch"
allowed-tools: ["Bash(git*)"]
---
# /commit-push-pr [optional-feature-name]

1. **Verify staged**: Run `git status --porcelain` and `git diff --cached`. If no staged changes, stop and say "No staged files. Stage with git add first."
2. **Branch**: Use $ARGUMENTS for name (e.g., feat/my-feature) or prompt user. Run `git checkout -b $BRANCH_NAME`.
3. **Commit**: Generate Conventional Commit message from `git diff --cached` only (e.g., "feat(cmd)!: replace diff config with validate command"). Run `git commit -m "$MESSAGE"`.
4. **Push**: `git push -u origin $BRANCH_NAME`.
5. **PR**: Write the PR description following `.github/PULL_REQUEST_TEMPLATE.md`, fill with commit details, run `gh pr create --title "$PR_TITLE" --body "$PR_BODY" --fill` (assume GitHub CLI).
6. Confirm each step.

Prioritize tfskel conventions: modular Go CLI, semantic versioning.
