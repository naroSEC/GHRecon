# Architecture

GHRecon is an independent discovery CLI. It converts explicitly authorized target identifiers into deduplicated, provenance-rich public GitHub assets. It does not clone repositories, validate cloud permissions, or detect/print secrets.

```text
CLI/config -> target normalizer -> deterministic query planner
           -> adaptive scheduler -> GitHub provider
           -> extractor pipeline -> correlation/deduplication
           -> text | JSON | JSONL
```

## Boundaries

- `internal/model` owns the stable interchange types.
- `internal/target` normalizes untrusted user input.
- `internal/query` creates bounded deterministic queries from embedded/configured rules.
- `internal/github` contains REST endpoints, authentication, pagination, response parsing, and rate-limit headers. GitHub API changes should stop here.
- `internal/scheduler` bounds concurrency and reduces it after throttling/transient failures.
- `internal/extractor` turns small GitHub text-match fragments into candidate assets. Extractors never receive tokens.
- `internal/correlation` deduplicates by canonical `(type, provider, value)` while merging all provenance.
- `internal/output` is the only stdout writer. Diagnostics and progress go to stderr.

The provider surface is deliberately small enough for a future GitLab/Bitbucket implementation without imposing a generalized provider framework on v1.

## Search and failure model

Queries are capped before execution. Every query owns its pagination and result cap. Response rate-limit headers update shared state; workers wait until reset when exhausted. `Retry-After`, bounded exponential backoff, and jitter handle secondary throttling. One query failure produces a typed partial error and does not discard successful findings. Authentication or invalid initial configuration fails fast.

## Security and data minimization

Tokens may be supplied explicitly, read from a local token file, or inherited from `GITHUB_TOKEN`. They are held in memory only and are never serialized, cached, or logged. GHRecon stores only text-match fragments returned by GitHub long enough to extract assets. Default output records minimal evidence; candidate credential files produce only a reason and path, never a suspected value.

## ADR-001: aggregate provenance in each event

An output event represents one canonical asset and has a non-null `sources` array. This makes deduplication explicit and preserves every observation without emitting twenty indistinguishable asset rows. JSONL remains stream-friendly: one complete finding per line. The JSON envelope additionally carries completion state, typed errors, and summary metrics.

## ADR-002: REST API and standard library HTTP

The GitHub REST API is isolated behind a client rather than using `gh`. This keeps the binary independent, makes headers and retry behavior observable, and permits mock-server integration tests.
