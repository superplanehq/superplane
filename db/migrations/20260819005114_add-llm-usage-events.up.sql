-- Historical version. Phase 1 later retimestamped this file so it would
-- apply after other 20260819 migrations. golang-migrate stores only the
-- current version, so a database that already applied 20260819005114
-- cannot migrate unless this version still exists as a file.
--
-- Keep this body as a no-op. The table is created by
-- 20260819201039_add-llm-usage-events.

SELECT 1;
