# agent-skills

Personal agent skills, installable via the [`skills`](https://skills.sh) CLI:

```bash
npx skills add jz-wilson/agent-skills -g -y                  # every skill
npx skills add jz-wilson/agent-skills --skill go-cli -g -y   # one skill
npx skills add jz-wilson/agent-skills --list                 # see what's here
```

It works with Claude Code, Codex, OpenCode, Cursor and other harnesses: without `-a` it asks which agents to install into, or name them with `-a claude-code -a opencode` (`-a '*'` for all). `-g` installs for your user rather than the current project, and `bunx` works in place of `npx`.

`go-cli`'s scripts and templates also need Go 1.27, golangci-lint v2 and a POSIX shell.

## Skills

| Skill | Purpose |
|---|---|
| [handoff](handoff/SKILL.md) | Compact the current conversation into a handoff doc for the next session — every claim about live state labelled VERIFIED or UNVERIFIED, with a re-runnable check per claim. |
| [pickup](pickup/SKILL.md) | Resume from a handoff doc — re-establish context, then re-verify every live-state claim against the real cluster/repo and gate on anything STALE before starting work. |
| [target-loop-design](target-loop-design/SKILL.md) | Design a measurable target plus the coupled iterative loop that reaches it — goal-to-loop scaffolding with an explicit coupling check. |
| [go-cli](go-cli/SKILL.md) | Write Go scripts, CLIs and TUIs the current way (Kong, testscript, golangci-lint v2, Charm v2 under charm.land), with verified templates, a scaffolder (`scripts/new.sh`) and a monthly dependency refresh. |

`handoff` and `pickup` are a matched pair: `handoff` emits a `## Live-State Claims` table, and `pickup` re-runs every row in it before touching anything. Neither trusts a claim it did not see proven.
