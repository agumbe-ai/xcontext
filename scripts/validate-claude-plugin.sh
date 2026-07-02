#!/usr/bin/env bash
set -euo pipefail

plugin="plugins/claude-code"
test -f "$plugin/.claude-plugin/plugin.json"
test -f "$plugin/.mcp.json"
test -f "$plugin/skills/xcontext/SKILL.md"
test -f ".claude-plugin/marketplace.json"

python3 -m json.tool "$plugin/.claude-plugin/plugin.json" >/dev/null
python3 -m json.tool "$plugin/.mcp.json" >/dev/null
python3 -m json.tool "$plugin/hooks/xcontext-opt-in.json" >/dev/null
python3 -m json.tool ".claude-plugin/marketplace.json" >/dev/null

if command -v claude >/dev/null 2>&1; then
  claude plugin validate "$plugin" --strict
  claude plugin validate . --strict
fi
