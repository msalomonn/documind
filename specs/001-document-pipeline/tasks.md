# DocuMind implementation decomposition

These are planned tasks, not completion claims. GitHub Issues will own live status.

| ID | Work | Depends on | Acceptance/evidence |
| --- | --- | --- | --- |
| DM-00 | Choose identity, parser/storage/provider interfaces; scaffold React/Go, DB and local object storage | Approved constitution/MVP | ADRs, repeatable startup and bounded parser contract |
| DM-01 | Owner-scoped upload and transactional job registration | DM-00 | DM-A01–A03; failure-boundary and idempotency tests |
| DM-02 | Worker claims, leases, retries and result fencing | DM-01 | DM-A04–A06; kill/restart and stale-worker tests |
| DM-03 | Text/PDF extraction and schema validation, fake + live adapter | DM-02; provider decision for live adapter | DM-A02, A05–A07, A09–A10; malformed/injected document cases |
| DM-04 | Progress/results/retry UI | DM-01; result contract | DM-A01, A08–A09; browser E2E with fake provider |
| DM-05 | Hardening, CI, deployment and demo documentation | DM-02–DM-04; cost/hosting/retention decisions | All acceptance criteria, live-provider smoke and public-demo limits |

DM-04 can proceed against agreed response fixtures while worker implementation progresses. Do not extract QueueCore in this milestone.

## Executable task checklist

All unchecked. These steps refine the backlog; issue state remains authoritative. Target source paths are proposed and will be created during bootstrap.

- [ ] T001 Record identity, storage/parser and extractor contracts in docs/adr/0002-integrations.md; pin versions after initial approval. [DM-00]
- [ ] T002 Scaffold web, cmd/api, cmd/worker and local database/object-store setup with migrations and reproducible README commands. Depends T001. [DM-00]
- [ ] T003 Define document/job/result schema, idempotency and active-generation constraints in db/migrations. Depends T002. [DM-A03,A06]
- [ ] T004 Implement owner-scoped bounded upload and storage-to-DB failure handling in internal/documents; expose registration and lookup routes. Depends T003. [DM-A01–A03]
- [ ] T005 Add real DB/storage upload tests in tests/integration for foreign owners, duplicate keys and storage/commit failures. Depends T004. [DM-A01–A03]
- [ ] T006 Implement job claims, leases, retry scheduling and fenced completion in internal/jobs and cmd/worker. Depends T003. [DM-A04–A06]
- [ ] T007 Add concurrent-claim, worker-kill, lease-expiry and stale-result tests in tests/integration. Depends T006. [DM-A04–A06]
- [ ] T008 Implement bounded PDF/text parsing and deterministic fake extractor in internal/extraction with invalid/malicious fixture tests. Depends T001,T006. [DM-A02,A06,A07,A09,A10]
- [ ] T009 Implement live provider adapter with schema/quote validation, deadlines and typed failures after provider decision in internal/extraction. Depends T008. [DM-A05–A07,A09,A10]
- [ ] T010 Build upload/progress/result/retry UI in web/src using agreed fixtures; add browser E2E against local fake provider. Depends T004 and response contract; integrate with T006,T008 before completion. [DM-A01,A08,A09]
- [ ] T011 Add CI, dependency checks and acceptance mapping under .github/workflows and docs/verification.md. Depends T005,T007–T010. [DM-05]
- [ ] T012 Add retention cleanup, quotas, deployed smoke and synthetic live-provider demo evidence after cost/hosting decisions. Document in docs/demo.md and release checklist. Depends T011. [DM-A09,A10; DM-05]
