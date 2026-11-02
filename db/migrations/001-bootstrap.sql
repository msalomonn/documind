CREATE SCHEMA documind;
REVOKE ALL ON SCHEMA documind FROM PUBLIC;
GRANT USAGE ON SCHEMA documind TO documind_app;
-- Document/job/result tables and owner constraints belong to DM-01 (T003).
