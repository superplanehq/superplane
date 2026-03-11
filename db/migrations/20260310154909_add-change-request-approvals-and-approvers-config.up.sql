BEGIN;

ALTER TABLE public.workflows
  ADD COLUMN IF NOT EXISTS change_request_approvers jsonb NOT NULL DEFAULT '[{"type":"anyone"}]'::jsonb;

CREATE TABLE IF NOT EXISTS public.workflow_change_request_approvals (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  workflow_id uuid NOT NULL,
  workflow_change_request_id uuid NOT NULL,
  approver_index integer NOT NULL,
  approver_type character varying(32) NOT NULL,
  approver_user_id uuid,
  approver_role character varying(255),
  actor_user_id uuid,
  state character varying(32) NOT NULL,
  invalidated_at timestamp without time zone,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT workflow_change_request_approvals_pkey PRIMARY KEY (id),
  CONSTRAINT workflow_change_request_approvals_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE,
  CONSTRAINT workflow_change_request_approvals_change_request_id_fkey FOREIGN KEY (workflow_change_request_id) REFERENCES public.workflow_change_requests(id) ON DELETE CASCADE,
  CONSTRAINT workflow_change_request_approvals_approver_user_id_fkey FOREIGN KEY (approver_user_id) REFERENCES public.users(id) ON DELETE SET NULL,
  CONSTRAINT workflow_change_request_approvals_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES public.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_change_request_approvals_change_request
  ON public.workflow_change_request_approvals (workflow_change_request_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_change_request_approvals_active
  ON public.workflow_change_request_approvals (workflow_change_request_id, invalidated_at, approver_index);

COMMIT;
