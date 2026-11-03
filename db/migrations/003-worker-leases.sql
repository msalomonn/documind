ALTER TABLE documind.jobs ADD COLUMN attempt_deadline timestamptz;
ALTER TABLE documind.jobs ADD CONSTRAINT processing_has_lease CHECK (
 (state='processing' AND claim_token IS NOT NULL AND lease_expires_at IS NOT NULL AND attempt_deadline IS NOT NULL AND lease_expires_at<=attempt_deadline)
 OR (state<>'processing' AND claim_token IS NULL AND lease_expires_at IS NULL AND attempt_deadline IS NULL)
);
CREATE INDEX expired_job_leases ON documind.jobs(lease_expires_at) WHERE state='processing';
