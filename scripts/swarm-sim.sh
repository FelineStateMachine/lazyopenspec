#!/usr/bin/env bash
# Simulates a small swarm of agents working through OpenSpec changes so you can
# watch lazyopenspec react. Works with macOS bash 3.2 and BSD tools.
#
#   scripts/swarm-sim.sh [target-dir]      (default /tmp/lazyopenspec-demo)
#   go run . [target-dir] --follow         (in another pane)
set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
target="${1:-/tmp/lazyopenspec-demo}"
agents="${AGENTS:-3}"
changes="$target/openspec/changes"
topics="add-recurring-tasks:task-management add-api-tokens:user-auth add-audit-log:audit-log add-bulk-edit:task-management improve-search-ranking:search add-slack-alerts:notifications"

rm -rf "$target"
mkdir -p "$target"
cp -Rp "$here/testdata/fixture/." "$target/"
git -C "$target" init -q
git -C "$target" add -A
git -C "$target" -c user.name=sim -c user.email=sim@example.com commit -qm "baseline"

echo "swarm-sim: $agents agents working in $target (ctrl+c to stop)"
echo "watch with: go run $here $target --follow"
echo

say() { printf '%s [agent-%s] %s\n' "$(date +%H:%M:%S)" "$1" "$2"; }

# Atomic write, the way editors and agents save: write a temp file, then rename.
atomic() { # atomic <file> (content on stdin)
	local tmp
	tmp="$(dirname "$1")/.$(basename "$1").$$.$RANDOM.tmp"
	cat >"$tmp"
	mv -f "$tmp" "$1"
}

active() { find "$changes" -mindepth 1 -maxdepth 1 -type d ! -name archive 2>/dev/null; }
pick() { awk -v r="$RANDOM" 'NF { a[n++] = $0 } END { if (n) print a[r % n] }'; }

check_task() { # check off the next unchecked top-level task somewhere
	local f
	f="$(grep -l '^- \[ \]' $(active | sed 's|$|/tasks.md|') 2>/dev/null | pick || true)"
	[ -n "$f" ] || return 1
	local task
	task="$(grep -m1 '^- \[ \]' "$f" | sed 's/^- \[ \] //' | cut -c1-50)"
	awk '!done && /^- \[ \]/ { sub(/\[ \]/, "[x]"); done = 1 } { print }' "$f" | atomic "$f"
	say "$1" "checked \"$task\" in $(basename "$(dirname "$f")")"
}

add_requirement() { # append an ADDED requirement to some delta spec
	local f n
	f="$(active | while read -r d; do ls "$d"/specs/*/spec.md 2>/dev/null; done | pick || true)"
	[ -n "$f" ] || return 1
	n=$((RANDOM % 900 + 100))
	if ! grep -q '^## ADDED Requirements' "$f"; then printf '\n## ADDED Requirements\n' >>"$f"; fi
	awk -v n="$n" '
		{ print }
		/^## ADDED Requirements/ {
			print ""
			print "### Requirement: Generated behavior " n
			print "The system SHALL handle case " n " consistently."
			print ""
			print "#### Scenario: Case " n
			print "- **WHEN** case " n " occurs"
			print "- **THEN** the system handles it"
		}' "$f" | atomic "$f"
	say "$1" "added requirement $n to $(echo "$f" | sed "s|$changes/||")"
}

advance_change() { # move one simulated change to its next artifact
	local entry name cap dir
	entry="$(echo $topics | tr ' ' '\n' | pick)"
	name="${entry%%:*}" cap="${entry##*:}" dir="$changes/$name"
	if [ ! -d "$dir" ]; then
		[ -d "$changes/archive" ] && ls "$changes/archive" | grep -q -- "-$name\$" && return 1
		mkdir -p "$dir"
		printf 'schema: spec-driven\ncreated: %s\n' "$(date +%F)" | atomic "$dir/.openspec.yaml"
		say "$1" "openspec new change $name"
	elif [ ! -f "$dir/proposal.md" ]; then
		printf '## Why\n\nUsers keep asking for %s; doing it by hand is slow and error-prone.\n\n## What Changes\n\n- Introduce %s\n- Update the %s capability\n\n## Impact\n\n- Affected specs: %s\n' \
			"${name//-/ }" "${name//-/ }" "$cap" "$cap" | atomic "$dir/proposal.md"
		say "$1" "wrote proposal for $name"
	elif [ ! -d "$dir/specs" ]; then
		mkdir -p "$dir/specs/$cap"
		printf '## ADDED Requirements\n\n### Requirement: %s\nThe system SHALL support %s.\n\n#### Scenario: Happy path\n- **WHEN** a member uses %s\n- **THEN** it works\n' \
			"${name//-/ }" "${name//-/ }" "${name//-/ }" | atomic "$dir/specs/$cap/spec.md"
		say "$1" "drafted spec delta $name/specs/$cap"
	elif [ ! -f "$dir/design.md" ] && [ $((RANDOM % 2)) -eq 0 ]; then
		printf '## Context\n\nSketching %s.\n\n## Decisions\n\n- Keep it behind a feature flag\n\n## Open Questions\n\n- Rollout order?\n' "$name" | atomic "$dir/design.md"
		say "$1" "wrote design for $name"
	elif [ ! -f "$dir/tasks.md" ]; then
		{
			printf '## 1. Implementation\n\n'
			for i in 1 2 3 4 5; do printf -- '- [ ] 1.%s Step %s of %s\n' "$i" "$i" "$name"; done
		} | atomic "$dir/tasks.md"
		say "$1" "planned tasks for $name"
	else
		return 1
	fi
}

edit_design() {
	local f
	f="$(active | sed 's|$|/design.md|' | while read -r p; do [ -f "$p" ] && echo "$p"; done | pick || true)"
	[ -n "$f" ] || return 1
	printf -- '- Note from agent-%s at %s\n' "$1" "$(date +%T)" >>"$f"
	say "$1" "noted a decision in $(basename "$(dirname "$f")")/design.md"
}

archive_done() { # archive a change whose tasks are all checked
	local d name f cap main
	d="$(active | while read -r c; do
		[ -f "$c/tasks.md" ] && grep -q '^- \[x\]' "$c/tasks.md" && ! grep -q '^- \[ \]' "$c/tasks.md" && echo "$c"
	done | pick || true)"
	[ -n "$d" ] || return 1
	name="$(basename "$d")"
	for f in "$d"/specs/*/spec.md; do
		[ -f "$f" ] || continue
		cap="$(basename "$(dirname "$f")")" main="$target/openspec/specs/$cap/spec.md"
		mkdir -p "$(dirname "$main")"
		[ -f "$main" ] || printf '# %s Specification\n\n## Purpose\nTBD - created by archiving change %s.\n\n## Requirements\n' "$cap" "$name" >"$main"
		awk '/^## ADDED Requirements/ { on = 1; next } /^## / { on = 0 } on' "$f" >>"$main"
	done
	mkdir -p "$changes/archive"
	mv "$d" "$changes/archive/$(date +%F)-$name"
	say "$1" "archived $name"
}

agent() {
	local id="$1"
	while true; do
		sleep "$(awk -v r="$RANDOM" 'BEGIN { printf "%.1f", 0.5 + (r % 26) / 10 }')"
		case $((RANDOM % 10)) in
			0 | 1 | 2 | 3) check_task "$id" || advance_change "$id" || true ;;
			4 | 5) advance_change "$id" || check_task "$id" || true ;;
			6) add_requirement "$id" || true ;;
			7) edit_design "$id" || true ;;
			8) archive_done "$id" || check_task "$id" || true ;;
			9) [ $((RANDOM % 3)) -eq 0 ] && git -C "$target" add -A && git -C "$target" -c user.name=sim -c user.email=sim@example.com commit -qm "agent-$id checkpoint" && say "$id" "committed checkpoint" || true ;;
		esac
	done
}

trap 'trap - INT TERM EXIT; kill 0 2>/dev/null' INT TERM EXIT
for i in $(seq 1 "$agents"); do agent "$i" & done
wait
