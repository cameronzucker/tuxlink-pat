# tuxlink-pat

A tuxlink-maintained fork of [`la5nta/pat`](https://github.com/la5nta/pat) — the Pat Winlink client.

This fork exists to support [tuxlink](https://github.com/cameronzucker/tuxlink) (a Linux-native Tauri Winlink client that wraps Pat). Tuxlink consumes `tuxlink-pat` as a git submodule and builds Pat into its AppImage release artifact.

## Why a fork?

See [tuxlink ADR 0011 — Fork Pat as `tuxlink-pat`](https://github.com/cameronzucker/tuxlink/blob/feat/v0.0.1/docs/adr/0011-fork-pat-for-tuxlink.md) for the full architectural decision and reasoning.

Short version: upstream Pat has known limitations (e.g., plaintext WL2K passwords in `config.json`) that tuxlink works around case-by-case in the tuxlink call sites. As those workarounds accumulate, fixing the limitations at the engine layer becomes cheaper than continuing to bandage tuxlink. The fork is the workshop for those engine-layer fixes; patches that fit upstream's accepted scope are submitted to `la5nta/pat` as PRs after they ship here.

## Repository conventions

- **Default branch:** `master` (inherited from upstream; tracks upstream's default-branch name).
- **Per-patch branches:** `patch-<slug>` or `<bd-id>/<slug>` (mirrors tuxlink's per-task-branch convention).
- **Merge mode:** merge-commit (no fast-forward); no squash; **branches RETAINED on merge** (NOT deleted — needed for upstream-PR cherry-pick portability per ADR 0011 §4).
- **Issue tracker:** GitHub Issues (this repo; `bd` is tuxlink-only).

## Workflow per fork patch

The full pipeline per patch is `superpowers:build-robust-features` (brainstorm → 5-round adrev with ≥1 cross-provider Codex → `writing-plans-enhanced` → `plan-review-cycle` → TDD impl → Codex on impl diff → PR). See tuxlink's ADR 0011 §3 for the discipline.

The opportunistic-sync model means upstream is merged into `master` at patch time, not on a separate schedule. Per-patch workflow:

```bash
# 1. Claim the patch's bd issue + create a worktree on tuxlink-pat
bd update <issue-id> --claim
git worktree add -b patch-<slug> /path/to/worktree origin/master

# 2. Add upstream remote if missing (idempotent)
cd /path/to/worktree
git remote get-url upstream > /dev/null 2>&1 \
  || git remote add upstream https://github.com/la5nta/pat.git

# 3. Verify upstream's current default-branch name (do NOT hardcode 'master')
UPSTREAM_BRANCH=$(gh api repos/la5nta/pat --jq '.default_branch')
echo "Upstream default branch: $UPSTREAM_BRANCH"

# 4. Opportunistic sync — fetch + merge upstream into this patch branch
git fetch upstream
git merge upstream/"$UPSTREAM_BRANCH"
# Resolve conflicts here (within the patch's brainstorm/plan, NOT auto-rollback)

# 5. Run the full build-robust-features pipeline on the patch
# (brainstorm → 5-round adrev → writing-plans-enhanced → plan-review-cycle →
#  TDD impl → Codex on impl diff)

# 6. Push + open PR against master (NOT --delete-branch on merge)
git push -u origin patch-<slug>
gh pr create --base master --head patch-<slug> ...

# 7. Operator merges via gh UI or:
gh pr merge <PR#> --merge   # NO --delete-branch (cherry-pick needs the branch)

# 8. Tuxlink side: update the submodule pin to the new tuxlink-pat commit
#    (this is a separate PR against tuxlink/feat/v0.0.1)
```

## Upstream contribution policy

For each fork patch:

- **If the patch is a bug fix or generally-useful feature:** submit a PR to upstream `la5nta/pat` after the patch ships here. Wait for upstream review.
- **If upstream accepts:** drop the fork-side patch on the next upstream-merge cycle.
- **If upstream declines** or the patch is tuxlink-specific by design (e.g., a tuxlink-IPC primitive Pat upstream wouldn't want): keep the patch in the fork indefinitely.

See tuxlink ADR 0011 §4 for the full contribution policy.

## Build

Pat builds via `bash make.bash` (NOT bare `go build`). Requires Go 1.24+ per `go.mod` and libax25-dev on Linux for full AX.25 hardware modem support (optional; Pat builds without it but with reduced functionality).

```bash
# Linux (Debian/Ubuntu):
apt install golang-go libax25-dev

# Build:
SKIP_TESTS=1 bash make.bash
# Produces ./pat in the repo root.
```

For tuxlink consumers: tuxlink's `src-tauri/build.rs` invokes this same `make.bash` from the submodule when building tuxlink's release profile.

## License

This fork preserves upstream Pat's MIT license. See [LICENSE](LICENSE).
