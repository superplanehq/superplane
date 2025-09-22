BEGIN;

ALTER TABLE organizations DROP COLUMN display_name;
ALTER TABLE canvases DROP CONSTRAINT unique_canvas_in_organization;

COMMIT;