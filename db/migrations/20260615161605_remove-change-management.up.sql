BEGIN;

DROP TABLE IF EXISTS public.workflow_change_request_approvals;
DROP TABLE IF EXISTS public.workflow_change_requests;

ALTER TABLE public.workflow_versions
  DROP COLUMN IF EXISTS change_management_enabled,
  DROP COLUMN IF EXISTS change_request_approvers;

ALTER TABLE public.workflows
  DROP COLUMN IF EXISTS change_management_enabled,
  DROP COLUMN IF EXISTS change_request_approvers;

ALTER TABLE public.organizations
  DROP COLUMN IF EXISTS change_management_enabled;

COMMIT;