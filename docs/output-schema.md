# Output schema v1

`schema_version` is the string `"1"`. Additive fields may appear within v1; consumers must ignore unknown fields. Removing fields, changing their meaning/type, or changing identity rules requires a new major schema version.

## Finding

Every finding contains:

- `schema_version`, `type`, `value`, and discovery `confidence` (0..1)
- `sources`, an array of provenance objects (possibly empty only for malformed third-party data)
- optional `provider`, `reason`, `attributes`, and `repository_details`

Supported v1 types are `repository`, `organization`, `user`, `domain`, `subdomain`, `url`, `endpoint`, `cloud_asset`, and `secret_candidate`. The canonical identity is `type + provider + normalized value`.

```json
{
  "schema_version": "1",
  "type": "subdomain",
  "value": "api.example.com",
  "confidence": 0.95,
  "sources": [{
    "source": "github_code_search",
    "repository": "example/backend",
    "path": "config/prod.js",
    "url": "https://github.com/example/backend/blob/main/config/prod.js",
    "query": "\"example.com\""
  }]
}
```

Repository metadata is nested under `repository_details`; `value` is its `owner/name`. Cloud findings set `provider` to `aws`, `azure`, or `gcp`.

## JSON

JSON output is one report envelope with `status` (`completed` or `partial`), `partial`, `findings`, typed `errors`, and `summary`. This is the format to use when the caller needs explicit completion metadata.

## JSONL

JSONL writes one deduplicated Finding per line. No logs or summary are written to stdout. Exit code 2 indicates partial completion, so an adapter can retain findings while detecting missing searches. Fatal setup failures use exit code 1.

## Text

Text output is for people and may evolve within a major release. It is not a machine interface.
