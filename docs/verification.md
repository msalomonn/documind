# DM-01 verification

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
