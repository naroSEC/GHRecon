# Platform integration

Use JSONL for streaming ingestion:

~~~sh
ghrecon --domain example.com --profile standard --jsonl
~~~

Parse only stdout. Logs are written to stderr. Each line is a complete schema-v1 deduplicated finding with all known sources. Consumers should ignore unknown additive fields and use type, provider, and value as the stable identity.

Exit codes:

- 0: completed
- 1: fatal input, configuration, or output error
- 2: partial completion; keep stdout findings and record missing queries

Use JSON instead when the adapter needs errors and summary metrics in-band. The JSON envelope exposes partial, typed query errors, counts, API request count, and duration.

Repository findings contain clone URLs for handoff, but GHRecon never clones them. A typical controlled pipeline is GHRecon -> policy/relevance review -> clone sandbox -> Gitleaks/TruffleHog/Semgrep.

Prefer `--token-file` with a runtime-mounted secret file, or use `GITHUB_TOKEN` through the platform's secret environment mechanism. `--token` is supported for manual use but may be visible in process listings and shell history. Redact arguments and the environment from diagnostic collection whenever direct token input is used.
