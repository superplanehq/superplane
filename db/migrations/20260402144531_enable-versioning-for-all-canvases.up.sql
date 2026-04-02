-- Versioning is now always enabled for all canvases (draft-first model).
UPDATE workflows SET versioning_enabled = true WHERE versioning_enabled = false;
