# Cursor integration

XContext supports Cursor through a marketplace-ready plugin that bundles the XContext MCP server, an automatic-use skill, and status, search, and stats commands.

## Prerequisites

- Cursor
- `agumbe-ctl` with XContext support on `PATH`

## Install locally for review

Use a symlink while developing or testing the marketplace package:

```bash
mkdir -p ~/.cursor/plugins/local
ln -s "$(pwd)/plugins/cursor" ~/.cursor/plugins/local/xcontext
```

Restart Cursor, or run **Developer: Reload Window**. Open **Customize** and confirm the XContext plugin and MCP server appear.

The plugin defaults to local mode:

```bash
agumbe-ctl xcontext mcp --local
```

Local mode needs no account, API key, network connection, or Kubernetes cluster.

## Managed cloud mode

For managed mode, configure the MCP server to pass `--cloud` instead of `--local` and start Cursor from an environment that contains:

```bash
export AGUMBE_XCONTEXT_API_URL=https://api.agumbe.ai/xcontext/v1
export AGUMBE_XCONTEXT_API_KEY=xctx_live_...
```

Do not commit API keys to Cursor plugin manifests or MCP configuration.

## Verify

Ask Cursor:

> Use XContext to run a verbose test command. Show the receipt, search for the failure, and report the redactions and token savings.

The result must contain a `ctx://` reference, an exit code, and no unredacted fixture secret. `xcontext_stats` should show a new object and redaction.

## Publish checklist

- `plugins/cursor/.cursor-plugin/plugin.json` is valid JSON.
- `.cursor-plugin/marketplace.json` is valid JSON and points to `./plugins/cursor`.
- Manifest paths are relative and do not contain `..`.
- `plugins/cursor/mcp.json` is valid JSON.
- Commands and skills include frontmatter metadata.
- The plugin is tested from `~/.cursor/plugins/local/xcontext`.

Submit the public repository URL at [cursor.com/marketplace/publish](https://cursor.com/marketplace/publish).
