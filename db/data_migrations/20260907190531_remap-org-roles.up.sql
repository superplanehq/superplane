BEGIN;

-- Mark current org_owner principals as owners before the role is removed.
UPDATE users
SET is_owner = true
WHERE id IN (
  SELECT replace(v0, '/users/', '')::uuid
  FROM casbin_rule
  WHERE ptype = 'g'
    AND v1 = '/roles/org_owner'
    AND v0 LIKE '/users/%'
);

-- Avoid unique collisions when an owner already has org_admin.
DELETE FROM casbin_rule AS owner_rule
USING casbin_rule AS admin_rule
WHERE owner_rule.ptype = 'g'
  AND admin_rule.ptype = 'g'
  AND owner_rule.v0 = admin_rule.v0
  AND owner_rule.v2 IS NOT DISTINCT FROM admin_rule.v2
  AND owner_rule.v3 IS NOT DISTINCT FROM admin_rule.v3
  AND owner_rule.v4 IS NOT DISTINCT FROM admin_rule.v4
  AND owner_rule.v5 IS NOT DISTINCT FROM admin_rule.v5
  AND owner_rule.v1 = '/roles/org_owner'
  AND admin_rule.v1 = '/roles/org_admin';

UPDATE casbin_rule
SET v1 = '/roles/org_admin'
WHERE ptype = 'g'
  AND v1 = '/roles/org_owner';

-- Avoid unique collisions when a viewer already has org_operator.
DELETE FROM casbin_rule AS viewer_rule
USING casbin_rule AS operator_rule
WHERE viewer_rule.ptype = 'g'
  AND operator_rule.ptype = 'g'
  AND viewer_rule.v0 = operator_rule.v0
  AND viewer_rule.v2 IS NOT DISTINCT FROM operator_rule.v2
  AND viewer_rule.v3 IS NOT DISTINCT FROM operator_rule.v3
  AND viewer_rule.v4 IS NOT DISTINCT FROM operator_rule.v4
  AND viewer_rule.v5 IS NOT DISTINCT FROM operator_rule.v5
  AND viewer_rule.v1 = '/roles/org_viewer'
  AND operator_rule.v1 = '/roles/org_operator';

UPDATE casbin_rule
SET v1 = '/roles/org_operator'
WHERE ptype = 'g'
  AND v1 = '/roles/org_viewer';

-- Drop obsolete default policies. Startup reloads the new CSV policies.
DELETE FROM casbin_rule
WHERE ptype = 'p'
  AND v0 IN (
    '/roles/org_viewer',
    '/roles/org_owner',
    '/roles/org_admin'
  );

DELETE FROM casbin_rule
WHERE ptype = 'g'
  AND v0 IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer')
  AND v1 IN ('/roles/org_admin', '/roles/org_viewer');

UPDATE role_metadata
SET role_name = 'org_operator',
    display_name = 'Operator',
    description = 'Can create tasks and interact with them.',
    updated_at = NOW()
WHERE role_name = 'org_viewer';

DELETE FROM role_metadata
WHERE role_name = 'org_owner';

INSERT INTO role_metadata (
  role_name,
  domain_type,
  domain_id,
  display_name,
  description
)
SELECT
  'org_maintainer',
  domain_type,
  domain_id,
  'Maintainer',
  'Can create and edit automations, set integrations, and change models.'
FROM role_metadata
WHERE role_name = 'org_admin';

UPDATE role_metadata
SET display_name = 'Admin',
    description = 'Can manage members, billing, and organization settings.',
    updated_at = NOW()
WHERE role_name = 'org_admin';

COMMIT;
