WITH default_fleets(slug, architecture, cpu_millicores, memory_mb) AS (
  VALUES
    ('e1-large-amd64', 'amd64', 2000, 8192),
    ('e1-large-arm64', 'arm64', 2000, 8192),
    ('e1-tiny-amd64', 'amd64', 2000, 1024),
    ('e1-tiny-arm64', 'arm64', 2000, 1024)
)
INSERT INTO runner_fleets (
  id,
  scope_type,
  scope_id,
  slug,
  enabled,
  spec,
  runner_version,
  created_at,
  updated_at
)
SELECT
  uuid_generate_v4(),
  'installation',
  NULL,
  defaults.slug,
  true,
  jsonb_build_object(
    'operating_system', 'linux',
    'architecture', defaults.architecture,
    'cpu_millicores', defaults.cpu_millicores,
    'memory_mb', defaults.memory_mb,
    'disk_gb', 30,
    'capabilities', jsonb_build_array('docker'),
    'max_execution_timeout_seconds', 3600
  ),
  'dev',
  now(),
  now()
FROM default_fleets AS defaults
WHERE EXISTS (
  SELECT 1
  FROM accounts
  WHERE installation_admin = true
    AND deleted_at IS NULL
)
AND NOT EXISTS (
  SELECT 1
  FROM runner_fleets
  WHERE scope_type = 'installation'
    AND scope_id IS NULL
    AND slug = defaults.slug
    AND deleted_at IS NULL
);
