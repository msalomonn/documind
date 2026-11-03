# ADR DM-0004: fenced leases and bounded worker attempts

Status: accepted within the approved MVP. Date: 2026-11-03.

## Context and decision

A durable queued row alone cannot recover a crashed worker. Use the existing PostgreSQL job table instead of adding a broker: claim eligible queued/retry rows or expired processing rows under `FOR UPDATE SKIP LOCKED`, generate a new random token and increment attempts in the same short transaction. Commit before external work. SKIP LOCKED supports competing queue consumers; it is not a general consistent-read strategy ([PostgreSQL SELECT documentation](https://www.postgresql.org/docs/current/sql-select.html)).

Default lease is 30 seconds with 10-second heartbeats. A 60-second attempt deadline caps all renewals; reserve min(5 seconds, one fifth of the attempt budget) for finalization. PostgreSQL wall-clock time is authoritative for eligibility and fencing. Processors honor context cancellation. Their local deadline is advisory; DB fencing prevents late writes even if the process clock differs. Configuration bounds permit shorter integration tests.

Renewal, failure and completion first acquire the job row lock, then check state, token, lease and deadline against the current wall clock. Checking time before a blocking lock could incorrectly accept an expired worker. Completion and insertion of the unique result for the generation share one transaction. Ambiguous/failed persistence does not trigger an unfenced follow-up write; restart/reclaim resolves durable state.

Count crashes as attempts to prevent an endlessly crashing document. Transient failures use 5s then 30s backoff plus 0–20% jitter; Retry-After can increase this up to 60s. Permanent invalid input/result stops immediately. The third failed attempt is terminal; the next claim scan makes a third expired attempt terminal. Persist only allowlisted error codes. Infrastructure errors stop the CLI and require restart; there is no supervisor configured in this local milestone.

## Alternatives and consequences

Holding locks during processing would serialize slow external calls and complicate crash recovery. A lease without tokens permits stale completion; tokens without expiry checks permit writes after ownership has expired. Adding a broker would still require idempotent results and fencing while increasing operational scope. These alternatives are unnecessary for the current workload.

Processing is at least once. A future provider call can be repeated after a crash and billed more than once; one stored result does not imply one external invocation. Manual retry/new generations, automatic polling, result views, parser isolation, source-quote checks and live AI remain later tasks. Structural result validation here does not prove semantic accuracy.

The worker engine accepts a Processor interface. Until DM-03 exists, the executable requires an explicitly selected synthetic document and validates a diagnostic prefix. This lets real S3/queue recovery be tested without consuming arbitrary uploads or claiming AI extraction. No ordinary queue drain or public deployment is enabled.

## Validation and revisit trigger

Real PostgreSQL tests cover competing consumers, skip-locked/future eligibility, token rotation, expired and lock-delayed writes, bounded renewal, retry exhaustion and deferred COMMIT failure. Real child processes are killed with SIGKILL or stopped with SIGTERM, then restarted after natural lease expiry. Browser tests show success/failure and attempt metadata for synthetic S3-backed diagnostic jobs on desktop/mobile.

Revisit queue indexing, polling, supervision and external-side-effect idempotency when adding the general processor or when measured queue size/throughput demands it. Public operation additionally needs retention, quotas, secrets/hosting decisions and operational monitoring.
