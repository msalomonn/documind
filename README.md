# DocuMind

A document-processing application demonstrating durable jobs, worker recovery and schema-validated AI extraction.

**Status:** DM-01 private text uploads implemented, pending PR integration on top of DM-00. Register/sign in, upload UTF-8 text, list and download your own originals. Each accepted upload has a durable queued job. Workers, PDF parsing and AI extraction are not implemented yet. No public deployment or paid provider is configured.

![Local workspace](docs/screenshots/upload-queued-desktop.png)

## Run locally

Requirements: **Go 1.27.3**, **Node 24.22.1**, Docker with Compose. Versions are recorded in `.go-version`, `.node-version`, `go.mod`/`go.sum` and `package-lock.json`.

```sh
npm ci
go mod download
npm run setup
npm run services:up
npm run db:migrate
```

In separate terminals, from the repository root:

```sh
npm run api
```

```sh
npm run dev
```

Open **http://127.0.0.1:4320** and create your own local account (email and a password of at least 12 characters). There are no seeded user credentials. Upload a `.txt` file, up to 10 MiB and 200,000 characters. Its status stays **Queued** until worker implementation. Download returns the private original; PDF uploads are explicitly rejected until DM-03. The screen also checks PostgreSQL and private storage. If unavailable, start services/API and use **Check again**. The frontend is a development server; production serving belongs to deployment work.

The worker scaffold can validate its dependencies with `npm run worker:check`. `npm run worker` starts an explicitly idle process and shuts down on SIGINT/SIGTERM. It does not claim or process jobs yet.

## Local services and credentials

| Component | Loopback port | Purpose |
| --- | --- | --- |
| React/Vite | 4320 | Local UI, proxies `/api` to Go |
| Go API | 4321 | Sessions, owner-scoped uploads/downloads and health |
| PostgreSQL 18 | 55433 | Separate bootstrap and restricted runtime roles |
| SeaweedFS 4.51 S3 | 58333 | Private `documind-local` bucket |

Ports are separate from TenantFlow. Compose uses digest-pinned images and persistent volumes; no host-directory mounts are required. Initial storage startup/recovery may take tens of seconds; `services:up` waits for volume registration. Memory limits: 512 MiB PostgreSQL and 768 MiB storage.

Setup generates `.env` with random credentials and restrictive file permissions; rerunning it preserves that file. Keep it with the existing volumes. Do not delete/regenerate credentials to restart services. The application uses bucket-scoped S3 credentials; separate initialization credentials create the bucket and the synthetic readiness object. The storage admin/filer interfaces are not published to the host. This is local HTTP development, not production TLS or a security certification.

`db:migrate` initializes the local runtime role if missing, applies checksum-guarded SQL migrations and initializes the private bucket/readiness object. It is repeatable and does not reset existing passwords. The second migration adds users, hashed sessions, upload intents, documents, jobs and result constraints. Existing migration checksums remain unchanged.

Cookies are HttpOnly and SameSite=Lax. Every mutation requires an exact `APP_ORIGIN`, default `http://127.0.0.1:4320`; an optional HTTPS origin enables Secure cookies. Local HTTP origins must be loopback. Do not switch to `localhost` without configuring the matching origin and restarting the API. Passwords use Argon2id. Password recovery, email verification, distributed rate limiting, retention and public quotas remain release work.

Run `npm run uploads:cleanup` to reconcile at most 100 unreferenced upload intents older than one hour. Repeat to drain larger backlogs. Storage/DB failures preserve cleanup candidates; successful document originals are excluded. This command is not a user-document deletion or retention feature.

Stop services without deleting data:

```sh
npm run services:stop
```

After a restart, run `npm run services:up` to wait for health. Do not run `docker compose down -v` unless you explicitly intend to erase this local data.

## Verify

With services running and initialization complete:

```sh
npx playwright install chromium
npm run check
```

Stop the local API and Vite processes first: browser tests own ports 4320/4321. Checks run TypeScript, production web/Go builds, Go tests with the race detector, real PostgreSQL/S3 integration, the worker dependency check and eight desktop/mobile browser scenarios. Integration tests use synthetic objects and remove their document/account fixtures. Browser scenarios create synthetic `@example.test` accounts and documents in the local database; they are intentionally not user data and are not a demo login. No real documents are needed. The health error test injects a 503 response. The lost-acknowledgement scenario commits a real upload, replaces its response with 503 and then retries the same key. These are response fault injections, not real network outages.

[Verification and limits](docs/verification.md) · [DM-01 checkpoint](docs/checkpoints/2026-10-31-private-upload.md) · [Bootstrap checkpoint](docs/checkpoints/2026-10-30-bootstrap.md)

## Engineering records

- [Approved MVP specification](specs/001-document-pipeline/spec.md)
- [DM-01 private upload contract](specs/001-document-pipeline/contracts/private-upload.md)
- [DM-00 acceptance and integration boundaries](specs/001-document-pipeline/contracts/bootstrap.md)
- [Plan](specs/001-document-pipeline/plan.md) and [tasks](specs/001-document-pipeline/tasks.md)
- [Integration ADR](docs/adr/0002-integrations.md) and [durable-job ADR](docs/adr/0001-durable-job-table.md)
- [Constitution](.specify/memory/constitution.md), [governance](docs/governance.md), [Spec Kit](docs/spec-kit.md)
- [Release checklist](docs/release-checklist.md) and [issues](https://github.com/msalomonn/documind/issues)

Next: DM-02 implements worker claims, leases, bounded retries and fenced completion. DM-03 adds isolated PDF/text parsing and extraction. Full DM-A02 remains incomplete while PDFs are rejected. Identity has no development bypass. Costs, retention and public hosting remain decisions before release.

The cross-project roadmap lives in [portfolio-engineering](https://github.com/msalomonn/portfolio-engineering). November 6 for a local MVP and November 13 for public release remain tentative targets.
