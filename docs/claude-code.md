# Claude Code integration

XContext supports Claude Code through stdio MCP and an installable plugin. The plugin adds the XContext MCP server, skill, and `/xcontext:status`, `/xcontext:search`, and `/xcontext:stats` commands.

## Prerequisites

- Claude Code
- `agumbe-ctl` with XContext support on `PATH`

## Install with agumbe-ctl

Local-first mode is the default:

```bash
agumbe-ctl xcontext init --claude-code --local
claude mcp get xcontext
```

Use `--claude-scope user`, `project`, or `local` to control where Claude Code records the MCP server. `user` is the default.

Managed mode keeps credentials in the environment rather than Claude configuration:

```bash
export AGUMBE_XCONTEXT_API_URL=https://api.agumbe.ai/xcontext/v1
export AGUMBE_XCONTEXT_API_KEY=xctx_live_...
agumbe-ctl xcontext init --claude-code --cloud
```

Start Claude Code from an environment that has these variables.

## Develop the plugin locally

```bash
claude plugin validate ./plugins/claude-code --strict
claude plugin marketplace add .
claude plugin install xcontext@agumbe --scope user
```

Open `/plugin` to inspect the installation and `/mcp` to confirm the XContext server. The plugin defaults to local mode; set `AGUMBE_XCONTEXT_MODE=cloud` for managed mode.

For the public repository, users can replace `.` with `agumbe-ai/xcontext` when adding the marketplace.

## Verify

Ask Claude:

> Use XContext to run a verbose test command. Show the receipt, search for the failure, and report the redactions and token savings.

The result must contain a `ctx://` reference, an exit code, and no unredacted fixture secret. `xcontext_stats` should show a new object and redaction.

## Remove

```bash
agumbe-ctl xcontext init --claude-code --remove
```

Removal preserves local XContext data.

## Automatic Bash hooks

The plugin does not automatically capture every Bash result. Its skill directs Claude to use the shell-free `xcontext_execute` tool before noisy commands. An optional post-execution example lives in `plugins/claude-code/hooks/xcontext-opt-in.json`; review its retention implications before adding it to Claude Code settings.
