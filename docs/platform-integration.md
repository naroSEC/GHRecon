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

Never pass a token on the command line. Supply GITHUB_TOKEN through the platform's secret environment mechanism and redact the environment from diagnostic collection.
