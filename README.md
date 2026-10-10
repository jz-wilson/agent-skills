# agent-skills

Personal agent skills, installable via the [`skills`](https://skills.sh) CLI:

```bash
npx skills add jz-wilson/agent-skills -g -y
```

## Skills

| Skill | Purpose |
|---|---|
| [handoff](handoff/SKILL.md) | Compact the current conversation into a handoff doc for the next session — every claim about live state labelled VERIFIED or UNVERIFIED, with a re-runnable check per claim. |
| [pickup](pickup/SKILL.md) | Resume from a handoff doc — re-establish context, then re-verify every live-state claim against the real cluster/repo and gate on anything STALE before starting work. |
| [target-loop-design](target-loop-design/SKILL.md) | Design a measurable target plus the coupled iterative loop that reaches it — goal-to-loop scaffolding with an explicit coupling check. |

`handoff` and `pickup` are a matched pair: `handoff` emits a `## Live-State Claims` table, and `pickup` re-runs every row in it before touching anything. Neither trusts a claim it did not see proven.
