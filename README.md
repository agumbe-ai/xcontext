# xcontext

xcontext is the context control plane for AI agents. It captures large artifacts before they reach a model, redacts secrets, returns a compact representation with provenance, preserves retrievable source material, and distinguishes potential compression from verified delivered savings.

> Status: active development. The current service implements the secure ingest/retrieval vertical and an in-memory development store. Do not use the in-memory store for production.

## Run locally

Requires Go 1.23 or Docker.

```bash
cp .env.example .env
set -a; source .env; set +a
make run
```

In another terminal:

```bash
curl -sS http://localhost:8080/xcontext/v1/objects \
  -H 'content-type: application/json' \
  -d '{"contentType":"test_output","source":"go test","text":"PASS one\nFATAL: provider timeout\nAuthorization: Bearer abcdefghijklmnopqrstuvwxyz"}'
```

`deliveryVerified` is honored only for an identity authenticated as a trusted interceptor. Manual uploads produce `potentialTokensSaved` but never claim `deliveredTokensSaved` or cost avoided.

## Product surfaces

- API/control plane: this repository.
- CLI and MCP entry point: `agumbe-ctl xcontext` (integration in progress).
- Console: shared Agumbe console under `/xcontext` (integration in progress).
- Gateway: remains an independent product and service.

See [architecture](docs/architecture.md), [security](SECURITY.md), and [roadmap](docs/roadmap.md).

## MCP interception

Build `xcontext-mcp` and configure an API key carrying `ingest`, `read`, `retrieve`, and `intercept` scopes:

```json
{
  "mcpServers": {
    "xcontext": {
      "command": "xcontext-mcp",
      "env": {
        "AGUMBE_XCONTEXT_API_URL": "https://api.agumbe.ai/xcontext/v1",
        "AGUMBE_XCONTEXT_API_KEY": "xctx_live_..."
      }
    }
  }
}
```

`xcontext_execute` accepts an argv array and never invokes a shell. Full command output is ingested and kept out of model context; the tool returns a compressed summary and context reference. Delivered savings are accepted by the API only when the authenticating key has the `intercept` scope.

## Production configuration

Production requires PostgreSQL and an Agumbe-compatible HS256 JWT secret. Startup fails closed without both. Migrations are embedded in the service image and run at startup by default.

Release publication uses GitHub OIDC, produces an SBOM and provenance attestation, pins the immutable image digest, and opens a promotion PR in `manifests-index`. Configure repository environments and these secrets:

- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_SERVICE_ACCOUNT`
- `GKE_PROJECT`
- `GKE_REGION`
- `MANIFESTS_REPO_TOKEN`

## License

Apache-2.0.
