ALTER TABLE workflows ADD COLUMN change_management_enabled BOOLEAN NOT NULL DEFAULT false;

-- Canvases that had versioning enabled are assumed to have had approvals configured,
-- so they become change management enabled.
UPDATE workflows SET change_management_enabled = true WHERE versioning_enabled = true;
