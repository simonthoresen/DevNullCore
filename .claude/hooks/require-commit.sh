#!/bin/bash
# Stop hook: block task completion if there are uncommitted changes.
# Reads JSON from stdin; checks stop_hook_active to prevent infinite loops.

INPUT=$(cat)

# Check stop_hook_active (prevent infinite loop after Claude already acted)
STOP_HOOK_ACTIVE=$(echo "$INPUT" | python3 -c \
  "import sys,json; d=json.load(sys.stdin); print(str(d.get('stop_hook_active',False)).lower())" 2>/dev/null || echo "false")

if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  exit 0
fi

# No git repo — nothing to check
TOPLEVEL=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$TOPLEVEL" ]; then
  exit 0
fi

# Check for any uncommitted or untracked changes
HAS_CHANGES=false
if ! git diff --quiet 2>/dev/null; then HAS_CHANGES=true; fi
if ! git diff --staged --quiet 2>/dev/null; then HAS_CHANGES=true; fi
if [ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]; then HAS_CHANGES=true; fi

if [ "$HAS_CHANGES" = "true" ]; then
  echo '{"decision":"block","reason":"There are uncommitted changes in the repository. Stage the relevant files, commit with a descriptive message, and push to the remote before completing this task."}'
  exit 2
fi

exit 0
