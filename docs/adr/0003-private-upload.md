# ADR DM-0003: private text uploads and durable cleanup intents

Status: accepted within the approved MVP. Date: 2026-10-31.

DM-01 enables first-party identity, private UTF-8 text uploads and durable job registration. PDF validation stays in DM-03's isolated parser; reject PDFs before storage/queueing until that boundary exists. This is an incremental delivery, not a reduction of the approved text/PDF MVP or completion of all DM-A02 cases.

Use opaque random sessions with only token digests in PostgreSQL, eight-hour expiry and revocation. Passwords use the already pinned x/crypto Argon2id implementation: 64 MiB, three passes, four lanes, 16-byte salt, 32-byte key. Limit concurrent derivations to two and local auth attempts to 30/minute across the process. Require exact configured Origin on every mutation; do not trust forwarded host headers. HttpOnly/SameSite cookies are Secure for HTTPS. A shared process budget is adequate only for the loopback milestone; public deployment requires abuse controls, recovery/verification, TLS and quotas.

S3 and PostgreSQL cannot share an atomic transaction. Before object creation, commit a random-key upload intent. Acquire a PostgreSQL transaction and owner/key advisory lock, replay an existing accepted document if its bytes match, otherwise put and stat-confirm the object and atomically insert document+queued job and remove the intent. A unique owner/key constraint backs idempotency; a partial unique index prevents multiple active job generations. Intents are inserted before taking the transaction connection to avoid pool exhaustion when concurrent callers wait on advisory locks.

On a failed or ambiguous commit, do not blindly delete the object: the commit may have succeeded. Reconcile only intents older than one hour whose objects are unreferenced, under row locks; storage deletion is idempotent and the intent remains on failure. Uploads have a bounded deadline far shorter than the grace period. Cleanup runs explicitly with `npm run uploads:cleanup`, at most 100 candidates per invocation. A scheduler and retention policy remain release work. A rejected replay can leave an intent without an object; the same reconciler safely removes it.

Hold the transaction during the bounded S3 write for this low-volume local milestone; reconsider reservations/shorter transactions when throughput warrants the extra protocol. The HTTP upload path has two slots and bounded body/text sizes. Downloads authorize before storage access, force attachment disposition and sandbox the response. No document content, passwords or session tokens are logged.

Validation: real database/S3 concurrency, fault after storage write, deferred constraint failure at database COMMIT, orphan grace and reconciliation, owner-negative routes, session revocation/expiry and browser lost-acknowledgement recovery. PDF, worker claims, live AI and distributed load are unverified here.

References consulted: [Go Argon2 package](https://pkg.go.dev/golang.org/x/crypto/argon2), [RFC 9106](https://datatracker.ietf.org/doc/html/rfc9106.html), [OWASP exact-origin CSRF guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html).
