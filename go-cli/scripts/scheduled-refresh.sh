#!/usr/bin/env bash
# Monthly check that the go-cli skill still works with the latest Go libraries.
# Runs refresh.sh (dry run: nothing in the skill changes) and sends an ntfy push:
#   - refresh FAILED            -> high priority, so a broken example gets fixed
#   - passed, upgrades pending  -> low priority, "run refresh.sh --write"
#   - passed, nothing new       -> silent
# Installed as a systemd user timer (see SKILL.md "Keeping the skill current").
#
# Env: NTFY_URL (server + topic, e.g. https://ntfy.example.com/alerts; set it
# in the systemd unit), NTFY_TOKEN (default: read from NTFY_ENV, which
# defaults to ~/.config/vault-nightly/ntfy.env), REFRESH (default: ./refresh.sh).
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
REFRESH=${REFRESH:-$HERE/refresh.sh}
NTFY_URL=${NTFY_URL:-}
NTFY_ENV=${NTFY_ENV:-$HOME/.config/vault-nightly/ntfy.env}
STATE=${XDG_STATE_HOME:-$HOME/.local/state}/go-cli-refresh
mkdir -p "$STATE"
LOG=$STATE/last.log

if [[ -z ${NTFY_TOKEN:-} && -f $NTFY_ENV ]]; then
  NTFY_TOKEN=$(sed -n 's/^NTFY_TOKEN=//p' "$NTFY_ENV")
fi

notify() { # title priority message
  if [[ -z $NTFY_URL || -z ${NTFY_TOKEN:-} ]]; then
    echo "scheduled-refresh: NTFY_URL or NTFY_TOKEN unset; would have sent: $1 - $3" >&2
    return 1
  fi
  curl -fsS -o /dev/null -H "Authorization: Bearer $NTFY_TOKEN" -H "Title: $1" \
    -H "Priority: $2" -H "Tags: go" -d "$3" "$NTFY_URL"
}

date -Is > "$LOG"
if ! "$REFRESH" >> "$LOG" 2>&1; then
  notify "go-cli skill: refresh FAILED" high \
    "An example no longer passes with the latest dependencies. Log: $LOG. Fix the asset, then scripts/refresh.sh --write."
  exit 1
fi

upgrades=$(grep -c 'upgraded:' "$LOG" || true)
# Toolchain lines read "  go  go1.27.2 -> go1.27.3"; compare without the go/v prefix.
newer=$(awk '/^  / && $3 == "->" && $1 != "goreleaser" && $1 != "task" {a=$2; b=$4; sub(/^(go|v)/,"",a); sub(/^(go|v)/,"",b); if (b != "" && a != b) print $1}' "$LOG" | tr '\n' ' ')
if (( upgrades > 0 )) || [[ -n $newer ]]; then
  notify "go-cli skill: updates available" low \
    "${upgrades} module upgrade(s) pass all checks${newer:+; newer toolchain: $newer}. Run scripts/refresh.sh --write and update the Versions table."
fi
exit 0
