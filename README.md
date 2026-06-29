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

## License

Apache-2.0.
