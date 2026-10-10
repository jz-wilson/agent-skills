---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up, with every claim about live state marked VERIFIED or UNVERIFIED.
argument-hint: "What will the next session be used for?"
disable-model-invocation: true
---

Write a handoff document summarising the current conversation so a fresh agent can continue the work. Save to the temporary directory of the user's OS - not the current workspace.

Include a "suggested skills" section in the document, which suggests skills that the agent should invoke.

Do not duplicate content already captured in other artifacts (PRDs, plans, ADRs, issues, commits, diffs). Reference them by path or URL instead.

Redact any sensitive information, such as API keys, passwords, or personally identifiable information.

## Mark every live-state claim

A handoff is read by an agent that cannot tell your observations from your assumptions. Past handoffs have carried wrong conclusions into the next session. So every **factual claim about live system state** in the doc must be labelled, and the doc must never assert something you did not check.

**What counts as a live-state claim** — anything that could have changed since you observed it:
- Cluster/infra — a workload is running/healthy, a namespace or CRD exists, a VIP/DNS record resolves, a cert is valid, an Argo app is Synced
- Repo — a branch exists, a commit is on `origin/main`, a PR is open/merged, a file contains X, CI is green
- External — a ticket is open, a service is reachable, a version is deployed

Intent, rationale, decisions, and next steps are **not** live-state claims — write those normally.

**Label each one inline** at the point it appears:
- `[VERIFIED: <command> → <raw evidence, trimmed> @ <UTC timestamp>]` — you actually ran the check this session and saw the output.
- `[UNVERIFIED: <why>]` — you inferred it, read it in an older doc, were told it, or observed it early in a long session and it may have drifted since. Say which.

Do not label something VERIFIED because it was true when you set it up hours ago. If the evidence is not in this session's transcript, it is UNVERIFIED.

**Then add a `## Live-State Claims` section** near the top of the doc — a single table so the next session can act on it mechanically:

| # | Claim | Status | Check to re-run | Evidence / why unverified |
|---|---|---|---|---|
| 1 | app StatefulSet running on node-a | VERIFIED | `kubectl -n app get sts` | `1/1` @ 2026-07-27T14:02Z |
| 2 | integration commit is on origin/main | UNVERIFIED | `git log origin/main --oneline -5` | assumed from local branch; never fetched |

`Check to re-run` must be a **read-only** command the next session can run verbatim. Every claim gets one, VERIFIED included — verification decays with time, and `/pickup` re-runs the whole table before starting work.

If you can cheaply verify an UNVERIFIED claim right now with a read-only command, do it and promote the row instead of shipping the doubt forward. If you cannot (host unreachable, no credentials), leave it UNVERIFIED and say so — never guess a status.

Call out the UNVERIFIED rows again in the section describing next steps, so anything the next session is about to build on is visibly unproven.

If the user passed arguments, treat them as a description of what the next session will focus on and tailor the doc accordingly.
