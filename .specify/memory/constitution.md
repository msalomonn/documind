# Engineering Constitution v1

Status: DRAFT — proposed for Manuel's review on 2026-10-25. Ratification is not implied by committing this file.
Scope: portfolio-engineering, TenantFlow and DocuMind. Roadmap: 2026-10-26 through 2026-11-13.

## 1. Behavior before implementation

Every feature has a versioned specification with user outcomes, explicit exclusions, numbered acceptance criteria and negative cases. Plans describe technical choices separately. Resolve material ambiguity before implementing the affected behavior. Small fixes may use a short issue with reproduction and acceptance criteria instead of a full feature package.

## 2. One accountable source per concern

GitHub Issues own task status, priorities and dependencies. Git owns specs, plans, ADRs, code and durable verification evidence. PRs own review and integration. A checkpoint records observations at a commit; it is not a competing task board. Before creating work, search for existing issues and changes. Offline backlogs become issue indexes once published.

## 3. Independent, small deliveries

Each app must build, test and release independently. Prefer a modular monolith for TenantFlow and an API plus worker for DocuMind. A PR should address one coherent behavior and reference its acceptance criteria. Do not create a shared runtime package or QueueCore until demonstrated duplication justifies extraction. QueueCore is deferred until at least 2026-11-16.

## 4. Security invariants are acceptance criteria

Tenant identity and permissions are verified on the server for every relevant operation. Tests must attempt cross-tenant reads and writes. Document access is owner-scoped. Files and extracted content are untrusted input; neither file text nor model output may grant permissions or execute tools. No secrets or real customer documents belong in Git, fixtures, screenshots or public demos.

## 5. Evidence is proportional and reproducible

Verify each changed behavior with appropriate unit, integration or browser checks. Authorization and job recovery require real integration tests, including negative cases. A mocked provider test does not prove a live AI integration. Record command or scenario, commit, environment, date, result and limitations. Never substitute a passing build for a demonstrated user flow.

## 6. Reliable background work

DocuMind assumes at-least-once job execution. Claims have leases; completion is fenced by a claim token; retries are bounded; final results are idempotent. Document any external side effect that cannot be made exactly-once, including repeated AI charges. A crashed worker must not leave a document permanently processing.

## 7. Explicit architecture decisions

Record consequential choices as ADRs: context, options, decision, consequences, validation and revisit trigger. Label proposals and accepted decisions accurately. Do not introduce infrastructure simply to demonstrate more tools. Pin tool and dependency versions and retain required third-party notices.

## 8. Agent work has boundaries

An implementation brief names the repository, issue, spec revision, allowed scope, dependencies and acceptance tests. Parallel work uses separate branches/checkouts and agreed contracts; agents must not overwrite each other's changes. At handoff record the observed HEAD, worktree state, evidence and next action. Review findings against actual code before accepting them.

## 9. Review, release and public presentation

Manuel approves this constitution and the initial MVP scopes before feature implementation, as required by the roadmap. Routine implementation choices within approved scope do not need repeated approval. Review changes through PRs. Reviewer assignments for personal repos are Manuel's responsibility; the Braundin-specific colleague policy is not silently inherited. Never claim branch protection is enabled without checking it.

Release each app when its own criteria pass: critical tests and CI, deployed smoke test, documented limitations, synthetic demo data, operational setup and recovery steps. Integration, deployment and portfolio publication are separate facts. Paid hosting, provider choice and spending limits remain decisions to resolve before incurring costs.

## 10. Keep the process useful

Target 4–6 hours per weekday across the entire portfolio: 60–90 hours total over 15 workdays. Reserve time for integration and fixes. Reduce optional features before weakening authorization, isolation or recovery. An app may ship without the other. Dates are targets, not delivery promises.

## Governance

Version: 1.0.0-draft. Owner: Manuel (`msalomonn`). Ratified: pending.
Change through a PR stating rationale, affected specs and migration needs. Applications carry a versioned snapshot in `.specify/memory/constitution.md`, with the source version recorded in `docs/governance.md`. Adopt updates deliberately; do not require another repo to exist on disk to use an app's constitution.
