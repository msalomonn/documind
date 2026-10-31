# DM-01 private upload checkpoint

Date: 2026-10-31, America/Argentina/Buenos_Aires. Branch: `feat/2-private-upload`, based on DM-00 `79a36ea6cf2fa2ab205c79d656e2ea76cd8f712e`. Source revision: the commit containing this record; exact tested head and remote CI are recorded on the PR. Issue: #2. DM-00 PR #8 remains a prerequisite; this checkpoint does not claim integration or public deployment.

Completed T003–T005: real registration/login/logout and opaque sessions; bounded owner-scoped text upload/list/detail/download; idempotent document/job transaction; persisted orphan intents and bounded reconciler; minimal desktop/mobile user flow. DM-A01 and DM-A03 have evidence. DM-A02 is partial: text limits/invalid inputs verified, PDFs deliberately rejected until T008. T006–T012 remain open; T010 is not complete just because the initial upload UI exists.

SDD sequence: contract and validation test first (initial compile failed because Validate did not exist), then schema/services/routes/UI, real integration and browser checks, self-review and evidence. The first browser run found ambiguous test locators matching both visible filenames and accessible download labels; narrowed assertions to visible filename elements. No application failure was hidden by that test correction.

Self-review covered owner predicates, session lifecycle/origin enforcement, body/hash resource limits, concurrent key replay, pool exhaustion avoidance, storage/COMMIT boundary, orphan cleanup grace and protection of referenced objects. Downloads declare the expected content length so interrupted streams are detectable. A final review found that a session rejection could retain a selected file across sign-in; the UI now clears that selection/key on 401, with desktop/mobile regression coverage. This is implementing-agent self-review, not independent approval.

Local verification: `npm run check`, `go vet ./...`, real dependency checks, synthetic failure cases, eight Chromium scenarios across desktop/mobile. See docs/verification.md for precise limits and mapping. No live AI calls, PDF parser or worker claims have been executed. Browser fixtures remain synthetic records in local volumes; integration fixtures are removed. Existing credentials and volumes were preserved.

Next: integrate prerequisite PR #8 and this delivery through review; DM-02 implements leased claims, crash recovery, bounded retry and stale-worker fencing. Do not imply queued jobs already process.
