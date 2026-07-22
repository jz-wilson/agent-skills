# Target-Loop Design — Anti-Patterns & Worked Examples

Load this when a target is vague, contested, or the user asks you to stress-test their draft. Use it to interrogate a Target + Loop, not to lecture — turn each anti-pattern into a specific question.

## Anti-patterns to catch

### In the Target

- **Activity dressed as outcome.** "Improve onboarding" is work, not a state. Ask: *"What's true about the world when this is done that isn't true now?"* Force a state with a number.
- **Vanity metric.** The number goes up while the thing you care about doesn't (page views vs. activations, lines of code vs. shipped value). Ask: *"If this metric hit target but the real goal didn't move, what metric would have caught that?"* Use that one instead.
- **No baseline.** Progress is unmeasurable without a starting value. If unknown, the loop's first iteration is "establish the baseline" — nothing else.
- **Unfalsifiable done-condition.** "When it feels solid" can never trigger an exit, so the loop runs forever. Demand a threshold + hold-time.
- **Metric with no proxy for a fuzzy goal.** Some outcomes (morale, taste, clarity) resist direct measurement. Don't abandon the target — find an observable proxy (survey score, review-cycle count, time-to-first-response) and name it as a proxy.

### In the Loop

- **Repetition, not iteration.** Same action every cycle with no adjustment rule = a while-loop with no feedback. The Adjustment element is what makes it a *loop*; if it's missing, progress is luck.
- **No kill-switch.** A loop with only a success exit will grind resources forever on an unreachable target. Every loop needs an abort condition and a max-iteration cap.
- **Signal ≠ metric.** The loop measures something adjacent to the target metric ("we shipped features" vs. "activation rate rose"). Drift here means the loop can succeed while the target fails.
- **Cadence mismatch.** Iterating daily on a metric that only moves monthly wastes cycles; iterating quarterly on a fast metric wastes the horizon. Match cadence to the metric's natural response time.
- **Guardrail-free optimization.** A loop pointed at a single metric will cheat to hit it (gaming, cutting corners, breaking things off-metric). Guardrails are the invariants that make the win real.

## Worked examples

### Latency (engineering)

- **Target:** p95 API latency ≤ 200ms (baseline 640ms) by end of Q3. Non-goal: p99 tail, cost reduction. Done when p95 ≤ 200ms for 7 consecutive days.
- **Loop:** Action = profile the slowest endpoint and ship one fix. Cadence = per weekday. Signal = p95 from the metrics backend. Gap = current p95 − 200ms. Adjustment = next iteration targets whatever endpoint now dominates p95. Guardrail = no regression in error rate or throughput. Exit-abort = if p95 hasn't dropped in 10 straight cycles, escalate to an architectural review instead of more point fixes.
- **Coupling:** signal *is* p95; a fix that doesn't lower p95 is reverted; success exit = the done-condition.

### Weekly writing habit (personal)

- **Target:** publish one essay per week for 12 weeks (baseline: 0). Metric = essays shipped. Threshold = 12. Non-goal: length, virality. Done when 12 are published.
- **Loop:** Action = draft + publish one piece. Cadence = weekly. Signal = count in the archive. Gap = 12 − shipped. Adjustment = if a week slips, next week's piece is scoped smaller (protect the streak over the polish). Guardrail = never publish without one editing pass. Abort = 3 consecutive missed weeks ⇒ the goal is mis-sized; renegotiate.

### Backlog reduction (ops)

- **Target:** open triage-needed issues ≤ 3 (baseline 41) within 6 weeks. Done when ≤ 3 held for 3 days.
- **Loop:** Action = triage a fixed batch each morning. Cadence = daily. Signal = open count with the label. Gap = count − 3. Adjustment = if inflow exceeds triage rate two days running, raise the batch size. Guardrail = every triaged issue gets a real label, not a rubber-stamp close. Max iterations = 30 working days.

## Interview flow reminder

1. Get the outcome as a *state* first; refuse to move on until it's a number.
2. Establish the baseline before designing any action.
3. Design the action to be one-cadence-sized, then derive cadence from the metric's response time.
4. Name all three exits out loud — people forget abort and max-iterations.
5. Run the coupling check last; it catches the mismatches the earlier phases hide.
