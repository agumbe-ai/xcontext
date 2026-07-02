# XContext

XContext is the context control layer for AI agents. It captures large tool outputs before they reach a model, redacts secrets, returns compact summaries with provenance, and keeps source material retrievable through durable `ctx://` references.

Use XContext locally with no account or network connection, or connect to Agumbe Cloud for shared sessions, API keys, usage telemetry, and the web console.

## Install

Install the Agumbe CLI from npm:

```bash
npm install -g @agumbe/ctl
agumbe-ctl version
```

Run without a global install when evaluating XContext:

```bash
npx -y @agumbe/ctl@latest xcontext --help
```

The XContext local-mode initializer requires `agumbe-ctl` 0.9.1 or newer. If `xcontext init` is unavailable, check `agumbe-ctl version` and upgrade the npm package.

## Local quickstart

Local mode needs no account, API key, network connection, or Kubernetes cluster.

```bash
agumbe-ctl xcontext init --codex --local
agumbe-ctl xcontext ingest --local --file ./test-output.log --content-type test_output
agumbe-ctl xcontext sessions --local
agumbe-ctl xcontext objects --local
agumbe-ctl xcontext stats --local
```

The initializer registers the XContext MCP server with Codex. Restart Codex after changing its MCP configuration.

Local metadata is stored transactionally in SQLite at:

```text
~/.agumbe/xcontext/xcontext.db
```

Redacted raw artifacts are stored separately with restricted permissions. Existing first-pass JSON stores are imported automatically and retained as timestamped backups.

### Search and retrieve

```bash
agumbe-ctl xcontext search "provider timeout" --local
agumbe-ctl xcontext retrieve ctx://local/ctxs_.../ctxo_... --local
```

### Raw-content policy

Set `AGUMBE_XCONTEXT_STORE_RAW_MODE` to one of:

- `redacted` — retain only redacted raw artifacts; this is the default.
- `original` — retain original content. Use only with an explicit data policy.
- `none` — do not retain raw artifacts.

Local commands never fall through to the network. Set `AGUMBE_XCONTEXT_LOCAL_DIR` to move the local data directory.

## Agumbe Cloud quickstart

Cloud mode provides durable shared storage and visibility in Agumbe Console.

1. Sign in at [console.agumbe.ai](https://console.agumbe.ai/xcontext).
2. Open **XContext → API Keys** and create a key.
3. Export the key through your shell or secret manager.
4. Initialize the agent integration in cloud mode.

```bash
export AGUMBE_XCONTEXT_API_URL=https://api.agumbe.ai/xcontext/v1
export AGUMBE_XCONTEXT_API_KEY=xctx_live_...

agumbe-ctl xcontext init --codex --cloud
agumbe-ctl xcontext status --cloud
```

The initializer does not write the API key into Codex configuration. The key must remain available in the environment when Codex starts.

Mode resolution is: explicit `--local` or `--cloud`, `AGUMBE_XCONTEXT_MODE`, saved configuration, then cloud when an API key exists or local otherwise.

## MCP and agent integrations

Run the stdio MCP server directly with:

```bash
agumbe-ctl xcontext mcp --local
```

XContext exposes these tools:

- `xcontext_execute` — run an argv array without a shell and return a protected receipt.
- `xcontext_ingest` — redact, summarize, and store context.
- `xcontext_search` — search stored summaries and protected context.
- `xcontext_retrieve` — retrieve context by `ctx://` reference.
- `xcontext_stats` — inspect object, redaction, retrieval, and token metrics.

Codex setup is automated through `agumbe-ctl xcontext init --codex`. Claude Code setup is automated through `agumbe-ctl xcontext init --claude-code`; an installable Claude Code plugin with skills and commands is available in [`plugins/claude-code`](plugins/claude-code). See the [Claude Code integration guide](docs/claude-code.md).

Install the public Claude Code plugin marketplace with `/plugin marketplace add agumbe-ai/xcontext`, then `/plugin install xcontext@agumbe`.

The standalone `xcontext-mcp` service also supports intercepted command execution. `xcontext_execute` accepts an argv array and never invokes a shell; full output is ingested while the agent receives a compact summary and context reference.

## Verify the integration

After restarting the agent client:

1. Confirm the four XContext MCP tools are available.
2. Ingest a log containing repeated lines and a test secret.
3. Confirm the response contains a `ctx://` reference and does not expose the secret.
4. Run `agumbe-ctl xcontext stats --local` or inspect the cloud dashboard.
5. Search for a distinctive line and retrieve its context reference.

Potential compression and verified delivered savings are reported separately. Manual uploads can produce `potentialTokensSaved`; delivered savings are accepted only from a trusted interceptor.

## Remove the Codex integration

```bash
agumbe-ctl xcontext init --codex --remove
```

Removing the integration preserves local data. Delete `~/.agumbe/xcontext` separately only when you intend to remove the stored context permanently.

## Develop the service

The hosted API and control plane live in this repository. Service development requires Go 1.23+ or Docker and PostgreSQL.

```bash
cp .env.example .env
set -a; source .env; set +a
make run
```

Production requires PostgreSQL and an Agumbe-compatible HS256 JWT secret. Startup fails closed without both. Migrations are embedded, serialized across replicas with a PostgreSQL advisory lock, and run at startup by default.

## Product surfaces

- Product page: [console.agumbe.ai/xcontext](https://console.agumbe.ai/xcontext)
- Cloud dashboard: [console.agumbe.ai/xcontext/dashboard](https://console.agumbe.ai/xcontext/dashboard)
- Hosted API: `https://api.agumbe.ai/xcontext/v1`
- CLI and MCP client: `agumbe-ctl xcontext`
- Architecture: [docs/architecture.md](docs/architecture.md)
- Claude Code integration: [docs/claude-code.md](docs/claude-code.md)
- Security policy: [SECURITY.md](SECURITY.md)
- Roadmap: [docs/roadmap.md](docs/roadmap.md)

## License

Apache-2.0
