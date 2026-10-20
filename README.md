# GHRecon

GHRecon is a GitHub OSINT and asset discovery tool for authorized security assessments.

Give it a domain, company name, GitHub organization, or repository and it searches public GitHub data for related repositories, source files, domains, URLs, endpoints, and cloud assets. Results are normalized, deduplicated, and emitted with their original GitHub provenance.

GHRecon is intentionally focused on discovery. It does not clone repositories, verify credentials, test cloud permissions, or replace tools such as Gitleaks, TruffleHog, and Semgrep.

## Features

- Repository, code, user, and organization search
- Domain, subdomain, URL, endpoint, and cloud asset extraction
- Repository metadata and clone URL export
- Deduplication with all source locations preserved
- Response-header-based GitHub rate-limit handling
- Bounded retries and adaptive concurrency
- Text, JSON, and JSONL output
- Quick, standard, and deep scan profiles
- YAML configuration and custom search queries
- Linux, macOS, and Windows builds

## Installation

Install the latest version with Go 1.23 or newer:

```sh
go install github.com/naroSEC/GHRecon/cmd/ghrecon@latest
```

The executable is installed in `$(go env GOPATH)/bin`. Make sure that directory is included in `PATH`.

PowerShell for the current session:

```powershell
$env:Path += ";$(go env GOPATH)\bin"
ghrecon --version
```

Prebuilt binaries are also available on the [Releases page](https://github.com/naroSEC/GHRecon/releases) for:

- Linux amd64 and arm64
- Windows amd64
- macOS amd64 and arm64

## Authentication

GHRecon accepts a GitHub token from a file, a command-line option, or the `GITHUB_TOKEN` environment variable. Token values are never written to logs or result files.

The recommended approach is a token file containing only the token and an optional trailing newline. Keep this file outside the repository and restrict access to your user account.

```text
github_pat_...
```

```sh
ghrecon --token-file /secure/path/github.token --domain example.com
```

PowerShell:

```powershell
ghrecon `
  --token-file "$env:USERPROFILE\.ghrecon\github.token" `
  --domain example.com
```

For short-lived manual use, a token can be supplied directly:

```sh
ghrecon --token "github_pat_..." --domain example.com
```

`--token` can expose the token through shell history or the operating system's process list. Prefer `--token-file` on shared systems and in automation.

The environment variable remains supported for compatibility:

```sh
export GITHUB_TOKEN=github_pat_...
```

PowerShell:

```powershell
$env:GITHUB_TOKEN = "github_pat_..."
```

Use the least privilege necessary. Public repository and code discovery does not require write access.

Token source precedence is `--token` or `--token-file`, followed by `GITHUB_TOKEN`. The two command-line token options cannot be used together.

## Usage

Search by domain:

```sh
ghrecon --domain example.com
```

Combine several identifiers:

```sh
ghrecon \
  --domain example.com \
  --company "Example Corporation" \
  --keyword examplecorp
```

Write JSONL for another tool or pipeline:

```sh
ghrecon --domain example.com --jsonl --output findings.jsonl
```

Export repository candidates only:

```sh
ghrecon --domain example.com --repos-only --json
```

Run `ghrecon --help` for the complete option list.

## Target files

Use `--input` to load one target per line:

```text
domain: example.com
company: Example Corporation
keyword: examplecorp
org: example
repo: example/backend
```

Blank lines and lines beginning with `#` are ignored. Bare domains are treated as domains; other bare values are treated as keywords.

## Scan profiles

### Quick

Repository and code search with domain and URL extraction. Useful for a first pass or a limited API budget.

### Standard

Adds interesting-file queries, endpoint extraction, cloud asset patterns, and credential-file candidates.

### Deep

Adds bounded follow-up searches for newly discovered target subdomains. Expansion always respects `--max-depth`, `--max-queries`, and `--max-results`.

```sh
ghrecon --domain example.com --profile deep --max-depth 2
```

## Configuration

Pass a YAML file with `--config`. Values override the embedded defaults.

```yaml
github:
  api_url: https://api.github.com

search:
  profile: standard
  max_depth: 1
  max_queries: 50
  max_results: 100
  concurrency: 4
  timeout: 20s

custom_queries:
  - type: code
    query: '"example.com" filename:custom.conf'
    reason: organization-specific configuration
```

The default interesting-file and candidate rules are available in [rules/default.yaml](rules/default.yaml).

## Output

Human-readable output is the default. Use `--json` for a complete report envelope or `--jsonl` for one deduplicated finding per line.

Machine-readable output is written only to stdout. Progress and verbose logs are written to stderr, so piping JSONL does not require log filtering.

Each finding includes:

- Schema version
- Asset type and normalized value
- Discovery confidence
- Repository, path, URL, and search query provenance
- Type-specific metadata where applicable

Exit codes:

- `0`: completed
- `1`: fatal input, configuration, or output error
- `2`: partial completion; successful findings are still available

The v1 schema and compatibility rules are documented in [docs/output-schema.md](docs/output-schema.md).

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/ghrecon
```

The default test suite uses mock GitHub servers and does not consume API quota. A live integration test is available separately:

```sh
go test -tags=integration ./tests/integration
```

## Documentation

- [Architecture](docs/architecture.md)
- [Output schema](docs/output-schema.md)
- [Query engine](docs/query-engine.md)
- [Extractors](docs/extractors.md)
- [Rate-limit handling](docs/rate-limits.md)
- [Platform integration](docs/platform-integration.md)

## Responsible use

Use GHRecon only for targets you own or are explicitly authorized to assess. Search results can reference infrastructure and configuration that was published unintentionally. Handle findings according to the rules of engagement and the disclosure policy that applies to the target.
