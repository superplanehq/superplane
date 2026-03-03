ALTER TABLE ONLY public.workflows
    DROP CONSTRAINT IF EXISTS workflows_live_version_id_fkey;

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_live_version_id_fkey
    FOREIGN KEY (live_version_id)
    REFERENCES public.workflow_versions(id)
    ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED;
