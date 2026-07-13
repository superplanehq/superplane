BEGIN;

DELETE FROM casbin_rule
WHERE ptype = 'p'
  AND v2 = 'canvases'
  AND v3 = 'publish';

COMMIT;
