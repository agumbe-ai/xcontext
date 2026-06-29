# Contributing

Create an issue before substantial changes. Pull requests must be focused, include tests, and preserve tenant isolation and the delivered-savings invariant.

Run:

```bash
make fmt
make vet
make test-race
```

Do not commit real logs, credentials, customer data, or generated raw-context databases. Synthetic fixtures belong under `evals/fixtures`.

