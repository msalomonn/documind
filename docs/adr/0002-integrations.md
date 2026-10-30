# ADR DM-0002: local integration boundaries

Status: accepted within the approved DM-00 scope. Date: 2026-10-30.

Use Go net/http, pgx/v5 and the minio-go/v7 S3 client; React/TypeScript/Vite supplies the separate web process. Pin resolved modules in go.mod/go.sum and npm lockfile. The Go SDK is distinct from the MinIO server: the latter's repository is archived, so use SeaweedFS 4.51 as the local S3 service with a digest-pinned image. PostgreSQL uses the already verified pinned PostgreSQL 18 image. Neither app depends on TenantFlow at runtime.

Identity uses first-party opaque database sessions with Argon2id passwords, implemented in DM-01 before any document endpoint is enabled. OIDC would avoid password ownership but add an identity service and provider setup for this bounded local MVP. Password recovery, verification and public-operation controls must be decided before release. DM-00 has no pretend login: protected routes reject every request with 401.

Keep parser and extractor behind explicit interfaces. DM-03 will pin a maintained PDF text extraction executable and enforce the bootstrap contract's time/memory/page/output limits in an isolated worker environment. UTF-8 and PDF content are data, never provider instructions. An AI provider is not selected or contacted in DM-00.

Local object storage has separate initialization and bucket-scoped application credentials; only S3 is published, on loopback. PostgreSQL has separate admin and runtime credentials. Generated configuration is ignored and preserved on rerun. Production hosting, TLS, identity recovery, parser isolation image, retention and AI budget are still release decisions.

Validation: readiness failure handling, real private S3 round trip, anonymous/wrong-key rejection, runtime DB privilege checks, repeated migrations and desktop/mobile bootstrap checks. This validates infrastructure, not the full document-processing MVP.

References consulted 2026-10-30: [Go distribution](https://go.dev/dl/), [pgx](https://github.com/jackc/pgx), [S3 client API](https://github.com/minio/minio-go/blob/master/docs/API.md), [SeaweedFS S3](https://github.com/seaweedfs/seaweedfs/wiki/Amazon-S3-API), [bucket-scoped credentials](https://github.com/seaweedfs/seaweedfs/wiki/S3-Credentials), [archived MinIO server](https://github.com/minio/minio).
