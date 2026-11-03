# DM-01 implementation plan

## Shape

React web app, Go HTTP API and Go worker; separate processes in one repository. PostgreSQL stores document metadata, jobs and results; S3-compatible storage holds private original objects. Start with a PostgreSQL-backed queue to avoid another operational dependency. Provider access lives behind an extractor interface, with a deterministic fake adapter for tests.

Proposed layout: `web/`, `cmd/api`, `cmd/worker`, `internal/{auth,documents,jobs,extraction}`, `db/migrations`, `tests/fixtures`. Pin versions during DM-00. DM-00 is the bounded bootstrap delivery described in [bootstrap contract](contracts/bootstrap.md); application processing remains in later tasks.

## Persistence and job contract

Document: ID, owner, random object key, content hash, detected type, size, timestamps. Job: document, generation, state, attempts, next run, lease expiry, claim token and safe error code. Result: unique job generation, schema version, extraction JSON and provider/model metadata without secrets. Enforce at most one active generation per document with a database constraint.

API streams and validates a bounded upload to private storage, then commits document+job metadata in a transaction. A failed database commit can leave an orphan object: track temporary keys and reconcile/delete unreferenced objects after a safe grace period. Do not enqueue when storage fails.

Workers atomically claim eligible rows with a short transaction using row locks and SKIP LOCKED, then commit before doing slow parsing or provider calls. They renew leases during work. Completion/retry updates compare claim token and lease ownership. An expired claim can be reclaimed; stale workers cannot finalize. State transitions and result insertion occur in one transaction. This is at-least-once processing, not exactly-once provider billing.

Default: 3 total automatic attempts; backoff 5s then 30s with jitter, plus bounded Retry-After handling; per-attempt deadlines and maximum lease duration defined during implementation. Permanent invalid documents fail immediately. Manual retry creates a new generation only when no active generation exists.

## API draft

- `POST /api/documents` with multipart file and `Idempotency-Key`; returns 202 after persistence.
- `GET /api/documents` and `GET /api/documents/:id` expose owner-scoped status/results.
- `GET /api/documents/:id/download` authorizes before a short-lived signed URL or streamed response.
- `POST /api/documents/:id/retry` returns 202 or 409 for an active generation.

## Extraction contract

Interface accepts bounded extracted text and deadline, returns schema-validated `{schemaVersion,title,summary,keyFacts:[{text,sourceExcerpt}]}` or a typed failure. Verify supporting excerpts against source text. Treat documents as data, delimit them from provider instructions and allow no tools. Provide clear unknown/error states. A schema check is necessary but does not prove factual accuracy; review synthetic evaluation cases and record failures.

## Verification

Real database/storage integration tests: duplicate keys, foreign-owner access, corrupt input, object/DB failure boundary, worker kill/restart, lease expiry and stale completion, provider 429/timeouts, malformed output and exhausted retries. Fake provider tests are deterministic. A separately authorized live-provider smoke uses synthetic content and recorded cost bounds. Browser scenario covers upload → progress → result and failure → retry.

## Dependencies and release decisions

DM-00 chooses identity integration, storage/parser libraries and provider contract. Deployment additionally needs hosting, retention/cleanup policy, quotas and cost cap. Local fixtures do not require real credentials. TenantFlow is not a runtime dependency.

## DM-00 execution

Implement T001/T002 against DM-B01–B06: contracts/ADR first, failing API readiness tests, then Go API/idle worker, private local services, migration tool, React status UI and CI. Real S3/database and browser checks must pass before reporting bootstrap ready. T003–T012 remain separate.

## DM-01 execution

See [private upload contract](contracts/private-upload.md) and [failure-boundary ADR](../../docs/adr/0003-private-upload.md). Implement T003–T005 with first-party sessions and the minimum text-upload UI. Full PDF content validation remains T008; reject PDF until then. Durable intents precede storage writes and transaction acquisition; object confirmation precedes document/job insertion. Exact origin checks, owner predicates and idempotency/failure tests are part of this delivery.

## DM-02 execution

T006/T007 follow [W01–W07](contracts/workers.md) and [ADR DM-0004](../../docs/adr/0004-recoverable-workers.md). Add migration 003 without changing historical checksums. Claim under a short SKIP LOCKED transaction, commit before S3/processor work, and lock then recheck wall-clock ownership on each mutation. Persist result and final state atomically. Every claim, including crash recovery, consumes one of three attempts. A hard deadline bounds heartbeat extensions; reserve finalization time inside it.

Keep the queue independent of extraction through a cancellation-aware Processor interface. Until T008 supplies an adapter, require an explicit synthetic document ID in the CLI, with a prefix-validated diagnostic outcome. General uploads remain queued. Surface real status, attempts and safe codes on manual refresh without closing T010. Validate with real PostgreSQL/S3, both SIGKILL and SIGTERM followed by restart, malformed outcome rejection, deferred COMMIT failure and desktop/mobile diagnostics. Provider failures are typed injections, not live provider requests.
