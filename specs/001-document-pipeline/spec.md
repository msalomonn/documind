# DM-01: Upload a document and track extraction

Status: MVP scope approved by Manuel on 2026-10-25. Requirements are not implementation evidence.

## User outcome

A signed-in user uploads a supported document, sees progress and receives structured extraction results or an actionable failure without losing track of the document.

## MVP boundary

Owner-scoped upload/list/detail; private storage; durable queued processing; worker retries; extraction into a fixed result schema; status polling and failure display. Support UTF-8 text and text-based PDF, up to 10 MiB and 30 PDF pages. Results include title, concise summary and key facts with supporting excerpts. OCR, scanned PDFs, RAG/chat, arbitrary office formats, multi-tenant organizations and a standalone QueueCore are excluded.

States: `queued → processing → succeeded`; transient failure goes to `retry_wait → processing`; invalid input or exhausted attempts ends at `failed`. Users can retry a failed job explicitly; each retry generation remains traceable. Details show safe error codes and attempt counts, never provider secrets.

## Acceptance criteria

- DM-A01: unauthenticated requests fail with 401; users cannot list, download, inspect results or retry another user's documents (404).
- DM-A02: unsupported, oversized, corrupt, encrypted or scanned-only documents fail clearly. Validate actual content as well as name/MIME and impose parser time/resource limits.
- DM-A03: upload completion persists document and job records atomically after object storage confirmation. A stable owner-scoped idempotency key prevents duplicate accepted jobs; reusing a key for different bytes returns 409.
- DM-A04: queued work survives API restart; a worker crash is recovered after its lease expires. Duplicate delivery cannot produce conflicting final results.
- DM-A05: transient provider/time-out/rate-limit errors retry at most three total automatic attempts with bounded backoff; permanent input failures do not retry. Exhaustion yields `failed` with a safe reason.
- DM-A06: a worker with an expired claim cannot overwrite a newer worker's result. A successful job stores one versioned result; malformed provider output never becomes a valid result.
- DM-A07: results conform to the documented JSON schema; supporting excerpts must occur in the extracted source text. Unsupported claims or missing fields are reported as extraction failure/uncertainty, not fabricated facts.
- DM-A08: the UI reflects progress, success and failure after refresh; users can manually retry failed work without creating concurrent active generations.
- DM-A09: the local flow works with a deterministic fake provider clearly labeled as such. A live AI-provider smoke test is a separate release criterion and must be recorded honestly.
- DM-A10: document contents and prompts are not logged. Model output cannot execute code/tools or change authorization; document deletion/retention and public-demo quotas are documented before deployment.

## Proposed defaults

File/page limits, extraction schema and retry policy above are proposed MVP boundaries. Identity/provider choice, deployment, retention duration and spending limits must be decided before public operation. Synthetic fixtures are sufficient for local validation.
