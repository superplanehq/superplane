BEGIN;

-- Remove default role policies (custom role policies remain).
DELETE FROM casbin_rule
WHERE ptype = 'p'
  AND v0 IN (
    'role:org_owner',
    'role:org_admin',
    'role:org_viewer'
  );

-- Remove default role inheritance (custom role inheritance remains).
DELETE FROM casbin_rule
WHERE ptype = 'g'
  AND v0 IN (
    'role:org_owner',
    'role:org_admin',
    'role:org_viewer'
  )
  AND v1 IN (
    'role:org_admin',
    'role:org_viewer'
  );

-- Update subject prefixes in casbin rules (roles, groups, users).
UPDATE casbin_rule
SET v0 = regexp_replace(v0, '^role:', '/roles/')
WHERE v0 LIKE 'role:%';

UPDATE casbin_rule
SET v0 = regexp_replace(v0, '^group:', '/groups/')
WHERE v0 LIKE 'group:%';

UPDATE casbin_rule
SET v0 = regexp_replace(v0, '^user:', '/users/')
WHERE v0 LIKE 'user:%';

-- Update parent role/group prefixes in grouping policies.
UPDATE casbin_rule
SET v1 = regexp_replace(v1, '^role:', '/roles/')
WHERE v1 LIKE 'role:%';

UPDATE casbin_rule
SET v1 = regexp_replace(v1, '^group:', '/groups/')
WHERE v1 LIKE 'group:%';

UPDATE casbin_rule
SET v1 = regexp_replace(v1, '^user:', '/users/')
WHERE v1 LIKE 'user:%';

-- Normalize custom role permissions to plural resource names.
UPDATE casbin_rule
SET v2 = 'roles'
WHERE ptype = 'p'
  AND v2 = 'role'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'groups'
WHERE ptype = 'p'
  AND v2 = 'group'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'members'
WHERE ptype = 'p'
  AND v2 = 'member'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'canvases'
WHERE ptype = 'p'
  AND v2 = 'canvas'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'blueprints'
WHERE ptype = 'p'
  AND v2 = 'blueprint'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'integrations'
WHERE ptype = 'p'
  AND v2 = 'integration'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

UPDATE casbin_rule
SET v2 = 'secrets'
WHERE ptype = 'p'
  AND v2 = 'secret'
  AND v0 NOT IN ('/roles/org_owner', '/roles/org_admin', '/roles/org_viewer');

-- Update domain format (e.g., org:<id> -> /org/<id>, project:<id> -> /project/<id>).
UPDATE casbin_rule
SET v1 = regexp_replace(v1, '^([a-zA-Z0-9_-]+):', E'/\\1/')
WHERE ptype = 'p'
  AND v1 IS NOT NULL
  AND v1 NOT LIKE '/%'
  AND v1 LIKE '%:%';

UPDATE casbin_rule
SET v2 = regexp_replace(v2, '^([a-zA-Z0-9_-]+):', E'/\\1/')
WHERE ptype = 'g'
  AND v2 IS NOT NULL
  AND v2 NOT LIKE '/%'
  AND v2 LIKE '%:%';

COMMIT;
