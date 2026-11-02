# DM-01 delivery contract — 2026-10-31

Scope: T003–T005 and the minimum registration/login/upload/list/download UI. Based on DM-00 PR #8; no worker or extraction is advertised. This implements DM-A01/A03 and the text-upload portion of A02. PDF parsing, page limits, encrypted/scanned detection stay in T008/DM-03. Until then **all PDF uploads fail with 415 `pdf_not_enabled`**, never enter the queue. Full A02 is not complete.

## Identity

POST `/api/auth/register` and `/api/auth/login`: JSON `{email,password}`; normalized email, 12–128 byte passwords. Registration returns 201 and login 200, setting a random opaque HttpOnly, SameSite=Lax, path=/ cookie with an eight-hour expiry. Only a SHA-256 token digest is stored. Argon2id uses a 16-byte random salt, 64 MiB, three passes, four lanes and a 32-byte key. At most two password derivations run concurrently; a process-local auth request budget is a local safeguard, not a distributed production abuse-control system. GET `/api/me` returns identity or 401. POST `/api/auth/logout` revokes the session and expires the cookie. Every mutation requires an exact configured Origin; missing/null/foreign origins fail 403. Secure cookies are enabled for HTTPS origins; the default is loopback HTTP only.

## Upload and reads

Authenticated POST `/api/documents` accepts exactly one multipart `file` and a 16–128 character ASCII idempotency key. Bound the entire body, file (10 MiB), filename (180 bytes), and UTF-8 text (200,000 code points). Require `.txt`, a text/plain or application/octet-stream part MIME, nonempty valid UTF-8 without binary controls; reject PDF signatures even when renamed. Reject all other extensions/types. Owner identity is derived from the session only.

Return 202 with `{id,name,size,contentType,status,generation,attempts,createdAt}` after confirmed S3 storage and an atomic document/job commit. Same owner/key/bytes replays that document; different bytes returns 409. Filename is not part of idempotency identity; replay retains the first accepted filename. Concurrent duplicates serialize with a transaction advisory lock, backed by a unique owner/key constraint. Different owners can use the same key independently.

GET `/api/documents` returns `{documents:[...]}` (newest first, at most 100). GET `/api/documents/{id}` and `/result` authorize before returning data; result is 409 `result_not_ready`. POST `/retry` returns 409 `job_active` for queued work. GET `/download` streams the private original as an attachment after authorization. Foreign and nonexistent IDs return the same 404 on every subroute. Original object keys, hashes and credentials are never exposed.

## Failure and persistence

Persist an upload intent before writing an object. Object keys are random and private. Upload uses a 30-second context and no retry outside that deadline. Storage write/stat failure never enqueues. Document + initial job + intent removal commit atomically. An ambiguous database commit must never trigger blind object deletion. `npm run uploads:cleanup` deletes only unreferenced intent objects older than one hour under row locks, then removes the intent; failures retain the intent for retry. Successful documents are never selected. This is orphan reconciliation, not user-document retention.

Schema includes one active job generation per document, composite result/job identity, bounded states/attempts and lease/token columns for DM-02. Queued work persists across API restarts. Worker processing/recovery, result validation and manual retries are later deliveries.

## Required evidence

Real PostgreSQL/S3 tests: registration/login/logout/expired sessions, origin rejection, unauthenticated/foreign access, input bounds, concurrent replay and conflict, owner-scoped keys, object failure, real deferred database commit failure, orphan grace/cleanup and persisted queue after handler restart. Desktop/mobile browser tests: registration, upload, refresh, original download, logout/login, invalid upload, visible error and recovery. Tests use synthetic data. Review is explicitly self-review; no independent approval is implied.

## Authentication entry screen refinement — 2026-10-31

Requested by Manuel during DM-01 review. UX-A01: while session verification is pending, render only a neutral loading screen; do not mount the document workspace or service health panel. UX-A02: signed-out users enter a standalone login screen with a registration alternative; neither documents nor infrastructure panels appear there. UX-A03: successful login/registration replaces the entire entry screen with the document application; a valid session survives reload without a login flash. UX-A04: logout or a protected API 401 replaces the application with the login screen and clears selected files/documents. Verify on desktop/mobile, including delayed session checks and invalid credentials. Server authorization remains unchanged.

UX-A05 (integration review, 2026-11-02): cancel the initial document-list request when its session is replaced or cleared. A late response from an earlier session must not clear a newly authenticated session or populate its document list. Verify a held document request is aborted on logout, then a fresh login loads normally.
