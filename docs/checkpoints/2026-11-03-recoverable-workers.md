# DM-02 recoverable-worker checkpoint

Date: 2026-11-03, America/Argentina/Buenos_Aires. Branch: `feat/3-recoverable-workers`, based on main `bf1b164ca83e6ea30f5afdd69aa3760c1d4c7143` (merged DM-01 PR #9). Issue #3 owns live status; the PR records exact source head, CI and integration outcome. This record does not claim merge or public deployment.

## SDD and implementation

W01–W07 were written before worker implementation. The initial policy test failed because the worker types/functions did not exist, then passed after implementation. T006/T007 now have the scoped code and verification described in `docs/verification.md`. T008–T012 remain open. ADR DM-0004 records timing, fencing, failure classification and the synthetic diagnostic boundary. `.specify/extensions.yml` is absent; there are no before/after implementation hooks to dispatch.

Migration 003 adds attempt deadlines and lease consistency. Claims use short SKIP LOCKED transactions, fresh tokens, bounded attempts and no processor I/O under the claim lock. Mutation checks happen after acquiring the row lock; atomic finalization stores exactly one result per generation. The CLI has explicit diagnostics only until DM-03. List/detail expose actual state, attempts and safe error codes; manual refresh shows diagnostic completion or failure.

## Verification and self-review

On this source, `npm run check` passed: TypeScript, web/Go builds, race-enabled tests, real PostgreSQL/S3 integration, worker dependency check and 14 desktop/mobile E2E cases. `go vet ./...` and `git diff --check` passed. Real subprocess tests cover SIGKILL and SIGTERM followed by restart after natural lease expiry. Failure injections cover transient retry/exhaustion, heartbeat loss, lock-wait expiry and deferred database COMMIT failure. No live AI was called.

The implementing agent reviewed locking/fencing, retry bounds, cancellation, unique/atomic results, document ownership, diagnostic scoping and disclosure of limitations. Visual review caught the stale upload notice saying processing was disabled after a successful diagnostic; the notice now reports only upload acceptance, while the library shows current status. Added graceful SIGTERM coverage alongside forced termination. No outstanding blocker was identified for this bounded local worker delivery. This is self-review, not independent approval.

## Handoff

Review the linked PR and its CI/evidence before integration. Manual retry/result views belong to DM-04; general extraction, bounded PDF parsing and fake-provider output validation belong to DM-03. Ordinary documents remain queued; never imply the diagnostic processor is AI extraction. Local credentials, existing uploads and persistent service volumes were preserved. Paid provider, hosting and public release decisions remain separate.
