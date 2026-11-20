ALTER TABLE workflow_versions
    ADD COLUMN automation_managed boolean NOT NULL DEFAULT false;
