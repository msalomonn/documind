CREATE TABLE documind.users (
 id text PRIMARY KEY, email text UNIQUE NOT NULL CHECK(length(email) <= 254),
 password_salt bytea NOT NULL CHECK(octet_length(password_salt)=16),
 password_hash bytea NOT NULL CHECK(octet_length(password_hash)=32),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE documind.sessions (
 token_hash text PRIMARY KEY, user_id text NOT NULL REFERENCES documind.users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
CREATE INDEX ON documind.sessions(user_id);
CREATE TABLE documind.upload_intents (
 object_key text PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE documind.documents (
 id text PRIMARY KEY, owner_id text NOT NULL REFERENCES documind.users(id),
 idempotency_key text NOT NULL, object_key text UNIQUE NOT NULL,
 content_hash text NOT NULL, name text NOT NULL, content_type text NOT NULL,
 size bigint NOT NULL CHECK(size > 0 AND size <= 10485760), created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_id,idempotency_key)
);
CREATE TABLE documind.jobs (
 document_id text NOT NULL REFERENCES documind.documents(id), generation integer NOT NULL CHECK(generation > 0),
 state text NOT NULL CHECK(state IN ('queued','processing','succeeded','retry_wait','failed')),
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 3),
 next_run_at timestamptz NOT NULL DEFAULT now(), lease_expires_at timestamptz, claim_token text, error_code text,
 created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(document_id,generation)
);
CREATE UNIQUE INDEX one_active_generation ON documind.jobs(document_id) WHERE state IN ('queued','processing','retry_wait');
CREATE INDEX eligible_jobs ON documind.jobs(next_run_at) WHERE state IN ('queued','retry_wait');
CREATE TABLE documind.results (
 document_id text NOT NULL, generation integer NOT NULL,
 schema_version integer NOT NULL CHECK(schema_version > 0), result jsonb NOT NULL,
 provider text NOT NULL, model text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(document_id,generation), FOREIGN KEY(document_id,generation) REFERENCES documind.jobs(document_id,generation)
);
REVOKE ALL ON ALL TABLES IN SCHEMA documind FROM PUBLIC;
GRANT SELECT,INSERT,UPDATE,DELETE ON documind.users,documind.sessions,documind.upload_intents,documind.documents,documind.jobs,documind.results TO documind_app;
