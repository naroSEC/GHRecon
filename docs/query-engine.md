# Query engine

The planner is deterministic. It emits repository and code queries for domains, companies, keywords, known organizations, and repositories; company/organization targets also produce organization searches. Standard/deep profiles add configured filename queries. custom_queries are appended last.

Queries are deduplicated by type and text and truncated at search.max_queries. Each GitHub search is independently capped by search.max_results and GitHub's search pagination ceiling.

Deep mode may create an exact, quoted code query for a newly extracted target subdomain. Expansion is breadth-by-depth and stops at both max_depth and max_queries. It never expands unrelated domains or cloud assets.

The GitHub search grammar is not rewritten by GHRecon. Invalid custom queries become typed partial failures rather than terminating successful work.
