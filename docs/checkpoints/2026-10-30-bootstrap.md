# DM-00 bootstrap checkpoint — 2026-10-30

Repository: DocuMind. Base: `6ac65ebec4e74274b938ef9f4052d1eb22377e9a`. Branch: `feat/1-local-bootstrap`. Source: the commit containing this checkpoint. Issue #1, T001/T002; acceptance DM-B01–B06 in the bootstrap contract.

Delivered: Go net/http API, explicitly idle worker, PostgreSQL with restricted runtime role, transactional/checksummed migration runner, private SeaweedFS S3 with distinct initialization/bucket-scoped runtime credentials, React readiness/retry screen, integration interfaces and CI. No identity bypass or document-processing claim. Ports 4320/4321/55433/58333 avoid TenantFlow.

SDD: inspected constitution/governance/spec/plan/tasks, selected `specs/001-document-pipeline`, and ran Spec Kit prerequisites. No checklists or pre/post implementation extension hooks exist. Added the bootstrap contract and integration ADR before implementation. The initial API test failed with the handler undefined, then passed after implementation. T003–T012 remain outside this delivery.

Verified locally: TypeScript; web/Go builds; Go race tests and vet; real DB/S3 integration; repeated migrations; preserved credentials on setup rerun; worker dependency probe; four browser scenarios with screenshots. See docs/verification.md for acceptance mapping and the injected UI failure distinction. The original Docker host-mount failure was resolved by internal container configuration. The restart write timeout exposed storage warm-up; service health gating and an authenticated readiness-object read now cover that boundary.

Self-review: separate initialization/runtime credentials, loopback published ports, denied protected routes, sanitized readiness failures, bounded probes and HTTP server timeouts, migration transactions/checksums, shutdown, preservation of `.env`/volumes and honest UI state. Dependencies resolved from official sources and pinned; downloaded Go archive checksum was verified before unpacking to the workspace. PostgreSQL and S3 fixtures are local and synthetic. Remote CI/integration status lives on the PR/issue.

Resume: keep local `.env` and Docker volumes. Stop local web/API before browser checks. DM-01 must implement sessions and ownership before enabling upload endpoints, then add document/job/result schema and storage-to-transaction failure handling. Provider selection, parser execution sandbox and public deployment are not authorized by this bootstrap completion.
