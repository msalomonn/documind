# Initial backlog

GitHub Issues are the live status source. The descriptions below define scope; they are not a second status board.

## DM-00: Select interfaces and bootstrap React/Go local stack

Depends on ENG-01 approval. Record identity, parser, storage and extractor contracts in ADRs; pin dependencies and define a bounded local setup.

## DM-01: Implement owner-scoped upload and job registration

Depends on DM-00. Verify DM-A01–A03: file limits, ownership, idempotency, confirmed storage and transactional job registration, including object/DB failure boundaries.

## DM-02: Implement recoverable worker claims and retries

Depends on DM-01. Verify DM-A04–A06 with real database tests: worker kill/restart, leases, stale completion, bounded retries and unique result.

## DM-03: Implement extraction and validate provider results

Depends on DM-02; live adapter requires provider decision. Verify DM-A02, DM-A05–A07, DM-A09–A10. Keep fake-provider evidence distinct from live smoke results.

## DM-04: Build document status, result and retry UI

Depends on DM-01 and agreed result contract; may progress using fixtures in parallel with worker work. Verify DM-A01, DM-A08–A09 via browser tests.

## DM-05: Harden, deploy and document DocuMind demo

Depends on DM-02–DM-04 and provider/hosting/cost/retention decisions. Fulfill release checklist, enforce quotas, verify a live synthetic extraction and prepare English docs/screenshots. Target 2026-11-13.

## Issue index

- [DM-00: Select interfaces and bootstrap React/Go local stack](https://github.com/msalomonn/documind/issues/1)
- [DM-01: Implement owner-scoped upload and job registration](https://github.com/msalomonn/documind/issues/2)
- [DM-02: Implement recoverable worker claims and retries](https://github.com/msalomonn/documind/issues/3)
- [DM-03: Implement extraction and validate provider results](https://github.com/msalomonn/documind/issues/4)
- [DM-04: Build document status, result and retry UI](https://github.com/msalomonn/documind/issues/5)
- [DM-05: Harden, deploy and document DocuMind demo](https://github.com/msalomonn/documind/issues/6)
