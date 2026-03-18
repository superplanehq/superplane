ALTER TABLE organizations
  RENAME COLUMN canvas_versioning_enabled TO versioning_enabled;

ALTER TABLE workflows
  RENAME COLUMN canvas_versioning_enabled TO versioning_enabled;
