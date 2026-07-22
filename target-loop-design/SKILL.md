---
name: target-loop-design
description: Design a measurable target plus the coupled iterative loop that reaches it. Use whenever the user wants to "set a goal and figure out how to get there", design a repeatable improvement process, build an OKR / success criteria, structure a feedback loop, define done-conditions, or asks for "a loop or a target", "an iterative process toward an outcome", "a way to keep making progress on X", or "how do I actually hit this goal". Produces a written Target + Loop design artifact. NOT for scheduling recurring prompt runs (that is /loop) or cron cloud agents (that is /schedule) — this designs the goal and process; those execute it.
---

# Target-Loop Design

Turn an intention into two coupled artifacts:

- **Target** — the desired outcome, stated as a measurable state of the world with an unambiguous done-condition.
- **Loop** — the repeatable iterative process that closes the gap to the Target, with a feedback signal that adjusts each cycle.

A loop without a target spins forever. A target without a loop is a wish. This skill forces both, and forces them to be *coupled*: every iteration of the loop must move a metric named in the target. If it can't, one of the two is wrong.

## When to use vs. skip

Use when the user wants to *design* a goal and the process to reach it — planning, not execution.

Do **not** use this to run something on an interval — that is the `/loop` skill (recurring prompt/slash-command execution) or `/schedule` (cron cloud agents). This skill produces the design that those tools can then execute. When the design is done, offer to hand the loop off to `/loop` or `/schedule` for actual running.

## Method

Work the three phases in order. Interview the user for anything you can't ground from context or the vault/codebase — but resolve what you *can* infer yourself first, and bring back only genuinely open decisions. Ask one question at a time when interviewing; batched questions are bewildering.

### Phase 1 — Form the Target

A well-formed target is a *state*, not an activity ("p95 latency under 200ms", not "work on performance"). Pin down every element below. Push back on any that are vague, unmeasurable, or actually a task in disguise.

1. **Outcome** — one sentence describing the world when this is achieved. A state, not a verb-phrase of effort.
2. **Why** — the motivating pressure. If there's no real "why", the loop won't survive contact with the first hard week.
3. **Metric** — the single number that most honestly tracks the outcome. Prefer a leading, observable signal over a lagging vanity one.
4. **Baseline** — the metric's current value. No baseline ⇒ you can't measure progress, so establish one before anything else.
5. **Threshold** — the value of the metric that counts as success (e.g. `≥ 95%`, `≤ 3 open`, `= 0`).
6. **Horizon** — the deadline or time window. "Someday" is not a horizon.
7. **Non-goals** — what this explicitly does *not* include. Prevents scope creep from silently redefining success.
8. **Done-condition** — the unambiguous stop. Usually "metric crosses threshold and holds for N cycles", but state it plainly so there's no arguing later.

If the user gives you a target that has no measurable metric, do not proceed — help them find a proxy metric first. An unmeasurable target cannot be closed by a loop.

### Phase 2 — Form the Loop

Model the loop as a feedback controller. Every effective loop has all eight elements; a loop missing any one of them is where iterative processes silently fail. Name each explicitly.

1. **Action** — the concrete work performed in one iteration. Small enough to finish within a single cadence.
2. **Cadence** — how often one iteration runs (per day, per PR, per sprint). Match it to how fast the metric can actually move.
3. **Signal** — how you *measure* the result of the action each cycle. This is the sensor; it must read the same metric named in the Target.
4. **Gap** — result compared against the threshold → the error signal. "How far are we still from done?"
5. **Adjustment** — the rule for how the *next* iteration changes based on the gap. This is the actual feedback; without it you have repetition, not a loop.
6. **Exit conditions** — three kinds, all required:
   - *Success:* the Target's done-condition.
   - *Abort:* the kill-switch — the condition under which you stop even though you haven't succeeded (cost, deadline blown, metric moving the wrong way N cycles running).
   - *Max iterations:* a hard cap so a stalled loop can't run forever.
7. **Guardrails** — invariants that must hold every cycle regardless of progress (don't break the build, don't exceed budget, don't ship without review). The loop optimizes the metric; guardrails stop it from cheating.
8. **Owner** — who or what runs each iteration and reads the signal (a person, an agent, a CI job).

### Phase 3 — Couple them

The two artifacts are only valid together. Verify the coupling before finishing:

- The Loop's **Signal** reads the Target's **Metric**. Same number, not a cousin of it.
- The Loop's **Adjustment** provably reduces the **Gap** — if a successful iteration wouldn't move the metric, the action is wrong.
- The Loop's success **Exit** *is* the Target's **Done-condition** — they must be the same line.
- The **Cadence** is fast enough to reach the **Threshold** before the **Horizon** (rough math: `(baseline→threshold distance) ÷ per-cycle movement ≤ cycles remaining`). If it isn't, either the horizon, the action size, or the cadence has to change — surface that now, not at the deadline.

## Output

ALWAYS use this exact template for the final artifact. Fill every field; if a field is genuinely unknowable, write `TBD — <what's needed to resolve it>` rather than deleting it.

```markdown
# Target: <short name>

**Outcome:** <one-sentence desired state of the world>
**Why:** <the motivating pressure>
**Metric:** <the single number that tracks it>
**Baseline → Threshold:** <current value> → <success value>
**Horizon:** <deadline / window>
**Non-goals:** <what's explicitly out of scope>
**Done when:** <unambiguous stop condition>

# Loop: <short name>

| Element | Definition |
|---|---|
| Action | <the work in one iteration> |
| Cadence | <how often one iteration runs> |
| Signal | <how the metric is measured each cycle> |
| Gap | <result vs. threshold> |
| Adjustment | <how the next iteration changes from the gap> |
| Guardrails | <invariants that must hold every cycle> |
| Owner | <who/what runs it> |

**Exit — success:** <= Target done-condition>
**Exit — abort:** <kill-switch condition>
**Exit — max iterations:** <hard cap>

## Coupling check
- Signal reads the Target metric: <yes + how>
- A good iteration moves the metric: <yes + why>
- Cadence reaches threshold before horizon: <rough math>
```

After presenting the artifact, offer to operationalize it: wire the Loop's Action + Cadence into `/loop` (recurring in-session runs) or `/schedule` (cron cloud agent), or capture the Target into the vault if the work lives there.

## Depth

For anti-patterns to catch during the interview (activity-as-outcome, vanity metrics, missing kill-switch, un-coupled loops) and worked examples across domains, see `references/checklist.md`. Load it when a target is subtle, contested, or the user wants their draft stress-tested.
