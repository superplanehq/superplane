BEGIN;

ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS name CHARACTER VARYING(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS change_management_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS change_request_approvers JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE public.workflow_versions AS v
SET
  name = COALESCE(w.name, ''),
  description = COALESCE(w.description, ''),
  change_management_enabled = COALESCE(w.change_management_enabled, false),
  change_request_approvers = COALESCE(w.change_request_approvers, '[]'::jsonb)
FROM public.workflows AS w
WHERE w.id = v.workflow_id;

ALTER TABLE public.workflows
  DROP CONSTRAINT IF EXISTS unique_canvas_in_organization;

ALTER TABLE public.workflows
  DROP CONSTRAINT IF EXISTS workflows_organization_id_name_key;

ALTER TABLE public.workflows
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS description,
  DROP COLUMN IF EXISTS change_management_enabled,
  DROP COLUMN IF EXISTS change_request_approvers;

COMMIT;
