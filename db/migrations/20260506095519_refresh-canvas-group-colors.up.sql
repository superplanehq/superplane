ALTER TABLE canvas_groups
  DROP CONSTRAINT canvas_groups_background_color_check;

UPDATE canvas_groups
SET background_color = CASE background_color
  WHEN 'blue-800' THEN 'blue-500'
  WHEN 'green-800' THEN 'green-500'
  WHEN 'violet-800' THEN 'violet-500'
  WHEN 'yellow-800' THEN 'yellow-950'
  ELSE background_color
END;

ALTER TABLE canvas_groups
  ALTER COLUMN background_color SET DEFAULT 'blue-500';

ALTER TABLE canvas_groups
  ADD CONSTRAINT canvas_groups_background_color_check
  CHECK (background_color IN ('blue-500', 'green-500', 'orange-500', 'slate-700', 'violet-500', 'yellow-950'));
