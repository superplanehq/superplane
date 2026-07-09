INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5)
SELECT ptype, v0, v1, v2, 'update', v4, v5
FROM casbin_rule old_rule
WHERE ptype = 'p'
  AND v2 = 'canvases'
  AND v3 = 'update_version'
  AND NOT EXISTS (
    SELECT 1
    FROM casbin_rule existing_rule
    WHERE existing_rule.ptype = old_rule.ptype
      AND existing_rule.v0 IS NOT DISTINCT FROM old_rule.v0
      AND existing_rule.v1 IS NOT DISTINCT FROM old_rule.v1
      AND existing_rule.v2 IS NOT DISTINCT FROM old_rule.v2
      AND existing_rule.v3 = 'update'
      AND existing_rule.v4 IS NOT DISTINCT FROM old_rule.v4
      AND existing_rule.v5 IS NOT DISTINCT FROM old_rule.v5
  );

DELETE FROM casbin_rule
WHERE ptype = 'p'
  AND v2 = 'canvases'
  AND v3 = 'update_version';
