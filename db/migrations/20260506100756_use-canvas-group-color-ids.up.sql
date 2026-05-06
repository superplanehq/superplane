ALTER TABLE canvas_groups
  DROP CONSTRAINT canvas_groups_background_color_check;

UPDATE canvas_groups
SET background_color = CASE background_color
  WHEN 'blue-800' THEN 'color_1'
  WHEN 'blue-500' THEN 'color_1'
  WHEN 'green-800' THEN 'color_2'
  WHEN 'green-500' THEN 'color_2'
  WHEN 'violet-800' THEN 'color_3'
  WHEN 'violet-500' THEN 'color_3'
  WHEN 'yellow-800' THEN 'color_4'
  WHEN 'yellow-950' THEN 'color_4'
  WHEN 'slate-700' THEN 'color_5'
  WHEN 'orange-500' THEN 'color_6'
  ELSE background_color
END;

ALTER TABLE canvas_groups
  ALTER COLUMN background_color SET DEFAULT 'color_1';

ALTER TABLE canvas_groups
  ADD CONSTRAINT canvas_groups_background_color_check
  CHECK (background_color IN ('color_1', 'color_2', 'color_3', 'color_4', 'color_5', 'color_6'));
