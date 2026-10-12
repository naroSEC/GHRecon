# GitHub rate limits

GHRecon does not assume numeric GitHub quotas. Every response updates shared state from X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Used, X-RateLimit-Reset, and X-RateLimit-Resource.

When remaining capacity is zero and a reset time is present, new requests wait until reset or context cancellation. HTTP 429, exhausted primary limits, server failures, and secondary/abuse responses use a maximum of three retries with exponential backoff and jitter. Retry-After is a lower bound; a secondary-limit response without Retry-After waits at least one minute. Authentication and invalid-query failures are not retried.

The scheduler starts at configured concurrency, halves its dispatch limit after a retryable API failure, and recovers gradually after successful queries. Pagination remains inside each scheduled query, so one query cannot cause an unbounded job graph.

GitHub may change quotas and abuse controls. Operators should choose conservative caps and consult current GitHub documentation. The /rate_limit endpoint is intentionally not polled for real-time decisions.
