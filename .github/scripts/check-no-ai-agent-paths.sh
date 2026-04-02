#!/usr/bin/env bash
# Fail CI if agent/IDE assistant paths are tracked (guards against git add -f).
set -euo pipefail
root=$(git rev-parse --show-toplevel)
cd "$root"
bad=0
while IFS= read -r f; do
  case "$f" in
  .cursor/*|*/.cursor/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .cursorrules|*/.cursorrules) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .github/copilot-instructions.md) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .github/instructions/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .github/chatmodes/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .copilot/*|*/.copilot/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .qodo/*|*/.qodo/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .amazonq/*|*/.amazonq/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .windsurf/*|*/.windsurf/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .continue/*|*/.continue/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .specstory/*|*/.specstory/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .kilocode/*|*/.kilocode/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .augment/*|*/.augment/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  .openhands/*|*/.openhands/*) echo "tracked (forbidden): $f" >&2; bad=1 ;;
  esac
  if [[ "$f" == .aider* || "$f" == */.aider* ]]; then
    echo "tracked (forbidden): $f" >&2
    bad=1
  fi
done < <(git ls-files)

if [[ "$bad" -ne 0 ]]; then
  echo "Remove these paths from the index (see CONTRIBUTING.md)." >&2
  exit 1
fi
