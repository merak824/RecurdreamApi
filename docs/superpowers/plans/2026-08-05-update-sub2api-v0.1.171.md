# Sub2API v0.1.171 Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the local Recurdream API fork to Sub2API v0.1.171 without losing current product work or pushing to GitHub.

**Architecture:** Commit the current product work as a recoverable checkpoint, import the official upstream v0.1.170-to-v0.1.171 delta in an isolated worktree, preserve fork-specific behavior while resolving overlaps, and fast-forward the local main branch only after verification.

**Tech Stack:** Git worktrees, Go, PostgreSQL migrations, Vue 3, TypeScript, Vitest, Vite, pnpm, Markdown.

---

### Task 1: Repair the existing HomeView test lifecycle

**Files:**
- Modify: `frontend/src/views/__tests__/HomeView.compact.spec.ts`
- Test: `frontend/src/views/__tests__/HomeView.compact.spec.ts`

- [ ] Reproduce the unhandled typewriter-timer error with the focused Vitest file.
- [ ] Track every wrapper returned by `mountHome` and unmount wrappers in `afterEach`.
- [ ] Run the focused test and confirm a zero exit code with no post-environment timer error.
- [ ] Commit only the test cleanup as `test: clean up HomeView timers`.

### Task 2: Checkpoint the balance-history feature

**Files:**
- Modify: the existing balance-history backend and frontend files reported by `git status`
- Exclude: `.superpowers/`

- [ ] Run focused backend tests for `internal/service`, `internal/handler`, and `internal/server/routes`.
- [ ] Run the two focused balance-history frontend test files.
- [ ] Stage only product source and tests; verify `.superpowers/` is not staged.
- [ ] Commit as `feat: add unified balance history`.

### Task 3: Create an isolated update worktree

**Files:**
- Modify if required: `.gitignore`

- [ ] Detect whether the checkout is already a linked worktree and whether a configured worktree directory exists.
- [ ] Ensure the selected project-local worktree directory is ignored, committing the ignore rule separately if required.
- [ ] Create branch `upgrade/sub2api-v0.1.171` from the checkpointed local `main`.
- [ ] Verify the isolated worktree starts clean and uses the existing dependency lockfiles.

### Task 4: Import the official upstream delta

**Files:**
- Modify: all files changed between official upstream tags `v0.1.170` and `v0.1.171`

- [ ] Fetch the two official tags into non-conflicting local refs; use official GitHub source archives if Git transport remains unavailable.
- [ ] Inspect the upstream diff, migration list, deletions, generated files, and release metadata.
- [ ] Apply the complete upstream delta with three-way context.
- [ ] Resolve conflicts by retaining Recurdream custom behavior around red packets, affiliate withdrawal, TTFT, profit control, homepage, deployment, and balance history while accepting v0.1.171 contract changes.
- [ ] Verify generated Ent/Wire output and migration filenames are internally consistent.

### Task 5: Update local release metadata

**Files:**
- Modify: `backend/cmd/server/VERSION`
- Modify: `README.md`
- Inspect: `README_CN.md`
- Inspect: `README_JA.md`

- [ ] Set the server version to `0.1.171`.
- [ ] Keep the main README Chinese-first and add detailed v0.1.171 features, fixes, breaking refund behavior, and upgrade notes.
- [ ] Preserve the existing v0.1.170 and Recurdream customization records.
- [ ] Verify no conflict markers or accidental English README replacement remain.

### Task 6: Verify and commit the update

**Files:**
- Inspect: the complete upstream update diff

- [ ] Run `git diff --check` and scan for conflict markers.
- [ ] Run focused Go tests for changed backend packages, then `go test ./...` where the environment permits.
- [ ] Run the full frontend Vitest suite and require a zero exit code.
- [ ] Run the frontend production build.
- [ ] Commit the update as `chore: update sub2api to v0.1.171`.
- [ ] Create local annotated tag `v0.1.171` at the verified update commit.

### Task 7: Integrate locally and report

**Files:**
- Modify: local Git branch and tag references only

- [ ] Fast-forward local `main` to the verified update branch without pushing.
- [ ] Confirm `.superpowers/` remains untracked and excluded.
- [ ] Confirm local `main`, `VERSION`, README, and tag all report v0.1.171.
- [ ] Report commits, verification evidence, and any residual test or deployment risk.
