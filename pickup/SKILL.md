---
name: pickup
description: Resume work from a handoff document written by /handoff — locate it, re-establish context, verify every claim it makes about live state, and continue.
argument-hint: "Optional: path to a specific handoff doc, or a keyword to match one"
---

Locate and load the handoff document so the current session can continue where the last one stopped.

**1. Find the doc.**
- If the user passed a file path, use it directly.
- If they passed a keyword, match it against `/tmp/handoff-*.md` filenames (case-insensitive substring).
- Otherwise, pick the most recent `/tmp/handoff-*.md` by mtime.
- If none found, say so and stop.
- If multiple plausible matches exist, list them and ask which to use.

**2. Read everything.**
Read the handoff doc in full. Then read any artifacts it references by path or URL (plans, PRDs, ADRs, vault notes) rather than assuming their content.

**3. Re-establish context** (CLAUDE.md recap rule — always do this):
- Run `git log --oneline -10` on the relevant repo(s).
- Read each note listed under `Relevant Vault Notes`.
- Scan `agents/*/reports/` for any active agent logs tied to this work.

**4. Verify the handoff against live state — mandatory gate.**
Handoff docs go stale between sessions and have contained wrong conclusions. Never begin work on an unverified claim.

*4a. Extract the claims.* If the doc has a `## Live-State Claims` table (written by `/handoff`), start from it — take its rows and its `Check to re-run` commands directly, and treat any row already marked `UNVERIFIED` as suspect. Then scan the rest of the doc anyway for claims the table missed.

Either way, end with every **factual claim about live system state** in a numbered checklist. Claims are things that could have changed since the doc was written:
- Cluster/infra state — a workload is running/healthy, a CRD or namespace exists, an IP/VIP/DNS record resolves, a cert is valid, an Argo app is Synced
- Repo state — a branch exists, a commit is on `main`, a PR is open/merged, a file contains X, CI is green
- External state — a ticket is open, a service is reachable, a version is deployed

Exclude intent, rationale, plans, and history ("we decided X", "next session should Y") — those are not verifiable state.

*4b. One command per claim.* For each claim, name the single cheapest **read-only** command that confirms or refutes it (`kubectl get`, `git log origin/main --oneline`, `gh pr list`, `dig`, `curl -s -o /dev/null -w '%{http_code}'`, `Read`). Run them. Read-only only — no `apply`, `delete`, `push`, or edits during verification. Batch independent checks in parallel.

*4c. Print the table.* Before any other output:

| # | Claim (from handoff) | Verification command | Result | Evidence |
|---|---|---|---|---|
| 1 | ... | `kubectl -n x get deploy y` | PASS | `1/1 Ready` |
| 2 | ... | `git log origin/main --oneline -5` | STALE | commit absent from `origin/main` |

`Result` is exactly `PASS` or `STALE`. `Evidence` is raw command output (trimmed), never a paraphrase. If a check cannot be run (host down, no credentials), mark it `STALE` — unverifiable is not passing.

*4d. Gate on STALE.* If every row is PASS, continue to step 5. If **any** row is STALE:
- Call out each STALE claim explicitly with what the handoff asserted vs. what is actually true.
- Ask the user how to proceed (e.g. re-derive the plan from live state / repair the drift first / proceed anyway and ignore that claim).
- Do not touch any file, config, or cluster resource until the user answers.

**5. Invoke suggested skills.**
For each skill listed under `Suggested Skills` in the doc, invoke it before starting work.

**6. Confirm the plan.**
Summarise back to the user:
- What Was Done (from the doc)
- What Was NOT Done (from the doc)
- Next Session Focus (from the doc)
- Any claims corrected by step 4, and how that changes the plan

Then proceed with the next session focus without waiting for further confirmation unless something is ambiguous — **provided step 4 produced no STALE rows, or the user has already resolved them**.

If the user passed an argument that is not a file path, treat it as a keyword selector for step 1 and also as a focus hint — weight the next steps toward that topic.
