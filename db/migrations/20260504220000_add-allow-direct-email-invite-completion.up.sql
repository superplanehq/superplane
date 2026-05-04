BEGIN;

ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS allow_direct_email_invite_completion boolean NOT NULL DEFAULT true;

COMMENT ON COLUMN organizations.allow_direct_email_invite_completion IS
  'When false, pending email invitations are not auto-accepted after non-OAuth sign-in (e.g. email and password).';

COMMIT;
