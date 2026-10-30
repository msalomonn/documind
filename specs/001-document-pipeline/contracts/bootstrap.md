# DM-00: local bootstrap contract

Scope: issue #1, T001/T002, approved document-pipeline feature. This delivery prepares the application and integration boundaries; it does not implement document upload, user registration, extraction or job execution.

## Acceptance

- DM-B01: Go API liveness returns 200 independently of dependencies. Readiness returns 200 only when PostgreSQL and the configured private S3 bucket are reachable, including an authenticated read of a synthetic readiness object from the storage volume; otherwise 503 with safe component states, no credentials or backend errors. Requests have bounded deadlines.
- DM-B02: setup creates random, ignored local credentials without overwriting existing files; migration is transactional, checksum-guarded and repeatable. Runtime DB credentials cannot alter schema. Ports do not collide with TenantFlow: web 4320, API 4321, PostgreSQL 55433, S3 58333; published services bind to loopback.
- DM-B03: real S3 integration proves authenticated put/get/delete of synthetic data and rejects anonymous/invalid-credential object reads. No anonymous identity or public bucket policy. Local storage and database volumes persist across restarts.
- DM-B04: React reports bootstrap readiness, handles unavailable dependencies with a retry action and clearly labels document processing as upcoming. Browser checks cover desktop/mobile, real readiness and failed readiness recovery.
- DM-B05: unauthenticated `/api/me` and document routes return 401 even with forged identity headers. No dev-auth bypass or simulated user session. Worker starts, probes dependencies and remains explicitly idle until DM-02; it cannot claim or pretend to process jobs.
- DM-B06: Go tests/race checks, build, TypeScript and browser checks run in CI with synthetic local services. No AI network call or paid service. Record actual versions, commands, results and limitations.

## Boundaries for later slices

Identity: first-party opaque sessions in PostgreSQL, random tokens stored only as hashes, HttpOnly SameSite cookies, exact-origin CSRF checks and Argon2id password hashing. DM-01 must implement and test registration, session expiry/sign-out and owner resolution before enabling any document route; no caller-controlled owner ID is authoritative.

Storage: private S3, random keys unrelated to filenames, bounded streams and context deadlines. Admin credentials only initialize the local bucket; app credentials are scoped to that bucket. Post-upload object confirmation precedes document/job transaction in DM-01.

Parser: UTF-8 text or text PDF, 10 MiB / 30 pages, maximum 200,000 extracted characters and five-second parsing deadline. PDF parsing will run in an isolated worker subprocess/container with memory/output limits; reject encrypted, corrupt or scanned-only input. Select and pin its executable in DM-03 before enabling PDF processing.

Extractor: `Extract(context, sourceText)` returns versioned title/summary/facts with exact supporting excerpts, or a typed permanent/transient failure. Source text is untrusted data and cannot authorize actions. Fake adapter and result validation arrive in DM-03; a live provider remains a separate decision.
