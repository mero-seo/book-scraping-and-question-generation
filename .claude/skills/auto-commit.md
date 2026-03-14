---
name: auto-commit
description: Analyze uncommitted changes and create well-structured conventional commits. Use this skill when the user asks to commit, create commit messages, or organize changes into commits. Automatically determines whether to create a single commit or multiple cherry-pickable commits based on the scope of changes.
---

This skill creates conventional commits that pass commitlint validation, are cherry-pickable, and follow the project's git standards. It analyzes the changeset to determine whether a single commit or multiple ordered commits are appropriate.

The user asks to commit their changes. They may have staged files, unstaged changes, or both.

## Step 1 — Gather Context

Run these commands to understand the current state:

```
git status
git diff --stat
git diff
git diff --cached --stat
git diff --cached
git log --oneline -5
```

Read the project's commit configuration:

- `.commitlintrc.js` for allowed types and length limits
- `.prettierrc` for formatting rules
- `.husky/pre-commit` for pre-commit hooks

## Step 2 — Classify the Changeset

Analyze the diffs and determine the nature of changes:

**Single commit** — when ALL of these are true:

- Changes serve one purpose (one bug fix, one feature, one refactor)
- No more than ~5 files changed
- All changes are tightly related

**Multiple commits** — when ANY of these are true:

- Changes span multiple unrelated features or concerns
- Type definitions AND their consumers both changed
- Mix of refactoring and new features
- Files changed across different domains (types + hooks + components + pages)

## Step 3 — Plan Commits

### Single Commit Path

Draft one commit message following the rules in Step 4.

### Multiple Commits Path

Group changes into logical units. For each group:

1. List the files that belong to it
2. Assign a commit type and scope
3. Write the commit message

**Ordering rules for cherry-pickability:**

- Types, enums, and interfaces FIRST (they are dependencies)
- Hooks and utilities SECOND (they depend on types)
- Components THIRD (they depend on hooks/types)
- Page-level wiring LAST (depends on everything)
- Each commit MUST compile independently — never reference something introduced in a later commit

## Step 4 — Commit Message Rules

Format: `type(scope): subject`

### Subject Line

- Maximum **72 characters** (enforced by commitlint)
- Lowercase first letter
- No period at the end
- Imperative mood ("add feature" not "added feature")

### Body (optional, for complex changes)

- Separate from subject with a blank line
- Wrap lines at **100 characters**
- Explain WHY, not WHAT (the diff shows the what)

### Allowed Types

| Type       | Use                                     |
| ---------- | --------------------------------------- |
| `feat`     | New features                            |
| `fix`      | Bug fixes                               |
| `docs`     | Documentation changes                   |
| `style`    | Code style changes (formatting)         |
| `refactor` | Code refactoring                        |
| `perf`     | Performance improvements                |
| `chore`    | Build process or auxiliary tool changes |
| `test`     | Adding or updating tests                |
| `ci`       | CI configuration changes                |
| `build`    | Build system or dependencies            |
| `revert`   | Revert previous commits                 |
| `wip`      | Work in progress                        |
| `release`  | Release preparation                     |

### Scope

Always provide a scope. Use lowercase. Pick the most relevant domain (e.g., `evaluation`, `types`, `interview`, `ui`).

### Forbidden

- NEVER add `Co-Authored-By` lines
- NEVER use `--no-verify` to bypass hooks

## Step 5 — Present the Plan and Get Approval

Show the user a clear summary before committing:

```
Commit Plan:
━━━━━━━━━━━━
[1/N] type(scope): subject
      Files: file1.tsx, file2.ts

[2/N] type(scope): subject
      Files: file3.ts, file4.tsx
```

If any file has changes belonging to multiple commits, flag it:

```
⚠ Partial staging needed:
   src/pages/create.tsx → commits #2 and #3
```

**Wait for explicit user approval before proceeding.** Do not commit without confirmation.

## Step 6 — Execute Commits

For each commit in order:

1. **Stage files** using `git add <file>` for whole files. For partial staging, extract specific hunks into a patch file and apply with `git apply --cached`.

2. **Commit** using HEREDOC format to preserve formatting:

```bash
git commit -m "$(cat <<'EOF'
type(scope): subject line here

Optional body explaining why this change was made,
wrapping at 100 characters per line.
EOF
)"
```

3. **Verify** with `git log --oneline -1`.

## Step 7 — Pre-Commit Hook Awareness

The project uses Husky pre-commit hooks:

1. `yarn lint-staged` — runs Prettier on staged files
2. `yarn lint` — runs ESLint on the entire project

### Known Pitfalls

**Mixed tabs and spaces:** Prettier (`useTabs: true`) can add alignment spaces in multi-line ternary expressions. ESLint's `no-mixed-spaces-and-tabs` rejects this. Fix by extracting long ternary conditions into variables.

**Partial staging conflicts:** `lint-staged` stashes unstaged changes, runs Prettier on staged portion, then restores. This can cause formatting mismatches. Prefer staging entire files or commit unrelated changes first to avoid partial staging.

If a pre-commit hook fails, fix the issue and create a NEW commit — never amend blindly.

## Step 8 — Final Verification

After all commits:

```
git log --oneline -N
```

Confirm the commit history reads as a clean, logical progression where each commit is independently valid.
