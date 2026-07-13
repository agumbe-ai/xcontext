#!/usr/bin/env bash
set -euo pipefail

plugin="plugins/cursor"

test -f "$plugin/.cursor-plugin/plugin.json"
test -f "$plugin/mcp.json"
test -f "$plugin/skills/xcontext/SKILL.md"
test -f "$plugin/commands/status.md"
test -f "$plugin/commands/search.md"
test -f "$plugin/commands/stats.md"
test -f "$plugin/assets/xcontext-logo.svg"
test -f ".cursor-plugin/marketplace.json"

python3 -m json.tool "$plugin/.cursor-plugin/plugin.json" >/dev/null
python3 -m json.tool "$plugin/mcp.json" >/dev/null
python3 -m json.tool ".cursor-plugin/marketplace.json" >/dev/null

python3 - "$plugin/.cursor-plugin/plugin.json" ".cursor-plugin/marketplace.json" <<'PY'
import json
import pathlib
import sys

for path in sys.argv[1:]:
    data = json.loads(pathlib.Path(path).read_text())
    values = []

    def walk(value):
        if isinstance(value, dict):
            for key, nested in value.items():
                if key in {"source", "logo", "mcpServers", "skills", "commands", "rules", "agents", "hooks"}:
                    values.append(nested)
                walk(nested)
        elif isinstance(value, list):
            for nested in value:
                walk(nested)

    walk(data)
    for value in values:
        candidates = value if isinstance(value, list) else [value]
        for candidate in candidates:
            if not isinstance(candidate, str):
                continue
            if candidate.startswith(("/", "~")) or ".." in pathlib.PurePosixPath(candidate).parts:
                raise SystemExit(f"{path}: invalid relative manifest path {candidate!r}")
PY

for file in "$plugin"/commands/*.md "$plugin"/skills/*/SKILL.md; do
  head -1 "$file" | grep -qx -- "---"
done
