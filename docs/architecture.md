# Architecture

## Product invariant

xcontext only claims delivered savings when a trusted integration proves that compressed content replaced the source artifact in the model context. Manual ingestion records potential savings.

## Data path

```text
agent tool / CI / SDK
        |
trusted interceptor (MCP, hook, SDK middleware)
        |
redact -> classify -> deterministic compressor -> compact result + ctx ref
        |                                      |
raw storage policy                        model context
        |
tenant-scoped retrieval and search
```

The control plane records immutable usage events, compressor version, source-line provenance, and the SHA-256 hash of the original input. Raw storage supports `redacted`, `original`, and `none`; `redacted` is the default.

## Deployment modes

- Local: processing, raw storage, and index remain on the user's machine.
- Cloud: the Agumbe service processes and stores according to tenant policy.
- Hybrid: raw artifacts remain local while summaries, hashes, and usage receipts sync to the control plane.

Storage and identity are interfaces. The in-memory adapters are development-only. Production adapters will use durable metadata storage, object storage for raw artifacts, and Agumbe JWT/API-key identity.

## Trust boundaries

- A context ref is an identifier, never authorization.
- Every lookup is constrained by authenticated tenant and workspace.
- Only a trusted interceptor can attest delivered savings.
- Raw retrieval is audited and never returned by list/detail endpoints.
- Production starts fail-closed when no identity resolver is configured.

