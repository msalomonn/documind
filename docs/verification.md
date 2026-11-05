# DM-02 verification

Date: 2026-11-03, America/Argentina/Buenos_Aires. Source: the commit containing this record; the PR identifies the exact source head and CI run. Environment: macOS arm64, Go 1.27.3, Node 24.22.1, PostgreSQL 18, SeaweedFS 4.51 and Playwright 1.64.1 Chromium.

`npm run check` and `go vet ./...` passed. The full check includes TypeScript, production web/Go builds, Go race tests, real PostgreSQL/S3 integration, the read-only worker dependency probe and **14 desktop/mobile browser scenarios**. No real documents or live AI requests were used.

| Scope | Reproducible evidence |
| --- | --- |
| W01 / A04 claims | `tests/integration/workers_test.go`: eight competing consumers yield one claim; locked rows are skipped and future retries are ineligible. |
| W02–W03 / A04 recovery | Real compiled worker child processes receive SIGKILL and SIGTERM, stop, wait for natural PostgreSQL lease expiry, restart and succeed on attempt 2. The previous token cannot complete. A third expired claim becomes terminal. |
| W02–W03 / A06 fencing | Old-token renew/fail/complete are rejected; completion waiting behind a row lock is rejected after expiry; renewal is capped at the hard deadline. Slow processing stays owned through heartbeat; lost heartbeat cancels the processor without writing success. |
| W03 / A06 atomic result | Duplicate completion leaves exactly one result. A real deferred PostgreSQL constraint trigger fails COMMIT: both succeeded state and result insertion roll back. Removing the fault permits completion. |
| W04–W05 / A05 retry mechanics | Typed transient rate-limit failure, Retry-After cap, no early retry, three-attempt exhaustion, immediate permanent failure, sanitized unknown errors and processor timeout. `internal/jobs/policy_test.go` covers jitter bounds and structural outcome checks. These are injected typed failures, not live provider 429 responses. |
| W05 / A06 structural subset | Runner rejects malformed outcomes. Version, required fields, sizes and provenance are checked before result persistence. Raw provider-payload and source-excerpt validation remain DM-03. |
| W06–W07 / partial A08 | `tests/e2e/workers.spec.ts`: real synthetic upload → selected CLI worker → persisted diagnostic success, then permanent failure; manual refresh displays status, attempt and safe code on desktop/mobile. Existing authentication, privacy and upload scenarios still pass. |

Some integration tests accelerate eligibility/expiry with targeted admin SQL updates. **The SIGKILL/SIGTERM subprocess scenarios wait for actual lease expiry without those updates.** All queue test claims are scoped to their own generated document IDs. Integration fixtures are cleaned up; browser fixtures remain synthetic `@example.test` accounts, as in DM-01.

Visual evidence inspected: [success desktop](screenshots/worker-success-desktop.png), [success mobile](screenshots/worker-success-mobile.png), [failure desktop](screenshots/worker-failure-desktop.png), [failure mobile](screenshots/worker-failure-mobile.png). Screenshots show diagnostic processing, not AI output. Upload screenshots were refreshed for the current explanatory text. They demonstrate UI state; backend recovery is evidenced by integration tests.

Limits: no general extraction adapter, PDF parsing, semantic quote validation, live provider, full result view, manual retry, automatic UI polling, public deployment, load/security certification or Go vulnerability scan. General uploads stay queued. Result retrieval remains a later endpoint implementation even though diagnostic results are persisted. A05 provider classification and A06 provider-output validation are not complete end-to-end until DM-03. A02, A07–A10 remain partially or wholly pending. Self-review is not independent reviewer approval. See the [checkpoint](checkpoints/2026-11-03-recoverable-workers.md) for integration boundaries.

---

## Historical DM-01 verification

The following records earlier revisions; their limits do not describe the newer worker implementation above.

Date: 2026-10-31, America/Argentina/Buenos_Aires. Source: the commit containing this record; PR records the exact tested head and CI result. Environment: macOS arm64, Go 1.27.3, Node 24.22.1, PostgreSQL 18, SeaweedFS 4.51, Playwright 1.64.1 Chromium.

| Acceptance | Reproducible evidence |
| --- | --- |
| DM-A01 | `tests/integration/upload_test.go`: real registration/login, token hashes, origin rejection, logout/expiry; unauthenticated and foreign detail/download/result/retry; foreign list empty. No development identity bypass. |
| DM-A02 (text portion only) | `internal/documents/validation_test.go` and integration: empty, invalid UTF-8, binary controls, unsupported type, 10 MiB and 200,000-character limits. Every PDF and renamed PDF signature rejected. **PDF page/corruption/encryption/scanning checks remain DM-03, so full A02 is incomplete.** |
| DM-A03 | Real DB/S3: eight concurrent service calls produce one document/job; byte conflict 409; same key scoped independently per owner; storage confirmation fault and real deferred COMMIT failure leave no document/job; durable intents survive and grace-period cleanup preserves fresh and committed originals. |
| Future schema invariants | DB rejects a second active generation and a result without its job. These checks do not prove worker fencing or extraction validation. |
| Durable queue subset of A04 | Handler recreation reads the persisted queued document. This is not a worker kill/recovery test. |
| Initial UI (partial A08) | Twelve desktop/mobile browser scenarios: readiness, injected outage/recovery, registration/upload/refresh/download/logout/login, invalid PDF, session-rejection selection reset and lost-acknowledgement retry without duplication. No processing/results UI claimed. |

Commands: `npm run check` (typecheck, web/Go builds, race tests, real integration, worker dependency check, twelve E2E scenarios); `go vet ./...`. All data is synthetic. S3 failure injection happens after a real write; database failure is a deferred PostgreSQL constraint trigger at COMMIT, not a mocked transaction. Browser acknowledgement loss is simulated by replacing a real 202 response with 503, then retrying the original idempotency key. This does not simulate a physical network outage.

Visual evidence inspected: [queued desktop](screenshots/upload-queued-desktop.png), [queued mobile](screenshots/upload-queued-mobile.png), [rejected PDF mobile](screenshots/upload-rejected-mobile.png), [registration desktop](screenshots/upload-signin-desktop.png). Ten screenshots include standalone login/registration and authenticated queue/rejection/unavailable states on both viewports. They show real local UI; health unavailable is injected.

Limits: no public deployment, live AI, PDF parser, worker claims/recovery/retries, provider-result validation, load/security certification, or Go vulnerability scan. HTTP upload concurrency and password derivation are bounded; public abuse controls, session/fixture retention, user deletion and quotas remain release work. Reconciler handles failed-upload orphans only. Reviewer evidence is self-review, not independent approval.

---

### Authentication entry refinement (UX-A01–UX-A04)

Manuel requested a standalone registration/login screen before entering the system. The session gate now renders one of three mutually exclusive screens: verification in progress, authentication, or the document application. Document/service components do not mount before authentication; the login defaults to sign-in with a registration alternative. Logout and protected API 401 return to login.

Validation on 2026-10-31: `npm run typecheck`, `npm run build` and all **10 desktop/mobile browser scenarios** passed. The entry tests assert no document/health requests while signed out, delay session resolution to inspect the neutral loading state, switch registration/login and reject wrong credentials. Existing real registration/upload/download/reload/logout and 401/reset cases verify entry to and exit from the system. Login, registration and application screenshots were regenerated and visually inspected. Backend code and authorization are unchanged; current-head CI is recorded on the PR.

### Integration review — 2026-11-02

Review reproduced an uncancelled initial document request surviving logout. Added UX-A05 and a regression test that first failed on the previous implementation. The session-bound document query now aborts when its effect is cleaned up, and the API helper checks cancellation before handling a response. Desktop/mobile tests hold the old request, observe its cancellation on logout, then verify a new login and refresh succeed. The suite now has 12 browser cases. Full verification and exact-head CI for integration into main are recorded on PR #9; this is self-review, not independent approval.

## Historical DM-00 verification (2026-10-30)

The following records the bootstrap revision before DM-01; its limitations describe that earlier revision.


Date: 2026-10-30, America/Argentina/Buenos_Aires. Source: the commit containing this record and its PR head. Environment: macOS arm64; Go 1.27.3, Node 24.22.1, PostgreSQL 18, SeaweedFS 4.51, Playwright 1.64.1 Chromium. CI uses Linux and runs the same `npm run check`.

| Acceptance | Evidence |
| --- | --- |
| DM-B01 | `internal/httpapi/server_test.go`: healthy/unavailable probes, safe status bodies, no-store and deadlines. Live readiness reads a synthetic S3 object, not just bucket metadata. |
| DM-B02 | Setup rerun preserves the `.env` hash; real migrations run twice; integration inspects restricted runtime-role privileges. Images and dependencies pinned. |
| DM-B03 | `internal/platform/integration_test.go`: authenticated put/get/delete, anonymous and wrong-key read rejection, denial of unrelated bucket creation. Existing schema/bucket survived a service restart without reinitialization after waiting for service health. |
| DM-B04 | Four desktop/mobile Playwright scenarios: real ready state, no misleading upload controls, no horizontal overflow, injected 503 state and retry against real API. |
| DM-B05 | Forged identity headers do not bypass 401; worker dependency probe exits successfully and normal worker is explicitly idle. |
| DM-B06 | Go race tests, TypeScript, web/Go build and worker probe; workflow added for PR/main. Remote CI result is tracked on the PR, not inferred from local success. |

The first API test failed before its handler existed. Bootstrap initially encountered a Docker Desktop host-mount restriction; configuration now originates inside containers, with persisted Docker volumes. A first immediate post-restart write timed out during storage leader/volume recovery. Compose now waits for volume registration, allows graceful shutdown, and readiness verifies real object reads. The integration suite then passed after recovery without recreating the bucket/schema.

Screenshots: [ready desktop](screenshots/bootstrap-desktop.png), [ready mobile](screenshots/bootstrap-mobile.png), [injected unavailable desktop](screenshots/bootstrap-unavailable-desktop.png), [injected unavailable mobile](screenshots/bootstrap-unavailable-mobile.png). They show local scaffold states, not completed document processing.

Limits: no implemented user sessions, document uploads, parser, queue claims/leases, AI provider, deployed smoke or load test. Fake-provider and live-provider evidence do not exist yet. PDF isolation/binary version and public-operation identity controls remain later decisions. No application secrets or document contents are logged by the scaffold. Review performed by the implementing agent is self-review, not independent reviewer approval.
