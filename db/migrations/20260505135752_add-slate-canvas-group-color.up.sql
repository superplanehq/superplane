ALTER TABLE canvas_groups
  DROP CONSTRAINT canvas_groups_background_color_check;

ALTER TABLE canvas_groups
  ADD CONSTRAINT canvas_groups_background_color_check
  CHECK (background_color IN ('blue-800', 'green-800', 'slate-700', 'violet-800', 'yellow-800'));
