BEGIN;

DO $$
BEGIN
  IF to_regclass('public.workflow_staging') IS NOT NULL
     AND to_regclass('public.workflow_staged_files') IS NULL THEN
    ALTER TABLE workflow_staging RENAME TO workflow_staged_files;
    ALTER INDEX idx_workflow_staging_version_id RENAME TO idx_workflow_staged_files_version_id;
  END IF;
END $$;

COMMIT;
