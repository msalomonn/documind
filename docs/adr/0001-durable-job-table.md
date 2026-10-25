# ADR DM-0001: PostgreSQL-backed jobs first

Status: accepted with the initial architecture approval. Date: 2026-10-25.
Context: demonstrate queue semantics and recovery within a three-week portfolio budget.
Options: managed queue/broker; Redis-backed queue; transactional PostgreSQL job table.
Decision proposal: use a job table with atomic claims, leases, bounded retries and fenced completion. Keep the worker separate from the API and isolate the queue behind an interface.
Trade-offs: fewer services and atomic document/job registration, but polling, indexing, lease recovery and retention require explicit implementation. At-least-once work may repeat external provider charges; do not promise exactly-once extraction.
Validation: concurrent claims, duplicate acceptance, worker crash, lease expiry, stale completion and transient/permanent provider failures.
Revisit: measured queue throughput/latency needs, multi-service consumers, or a justified QueueCore extraction after Nov 16.
