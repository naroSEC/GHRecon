# Extractors

Extractors consume GitHub text-match fragments and file metadata. They do not fetch or persist full repository files.

- Domain extraction accepts only the configured root domains and their label-boundary subdomains.
- URL extraction finds HTTP(S) candidates and validates them with Go's URL parser. Scheme, hostname, escaped path, optional port, and query presence are structured attributes.
- Endpoint extraction accepts API/admin/auth/health/debug-style relative paths and rejects common static assets.
- Cloud extraction identifies public AWS S3/API Gateway, Azure Storage, and GCP Storage/Cloud Run naming patterns. It does not test existence or permissions.
- Credential-file candidates are based on configurable filenames and expose no suspected value.

All values include repository/path/search provenance. Text fragments are deliberately discarded during correlation.

Rules live in the embedded YAML configuration rather than being scattered across the query planner. Regex behavior is covered by regression fixtures under tests/fixtures.
