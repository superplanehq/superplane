BEGIN;

UPDATE public.canvas_folders
SET background_color = 'slate'
WHERE background_color = 'yellow';

ALTER TABLE public.canvas_folders
  DROP CONSTRAINT canvas_folders_background_color_check;

ALTER TABLE public.canvas_folders
  ADD CONSTRAINT canvas_folders_background_color_check
  CHECK (background_color IN ('blue', 'green', 'purple', 'slate', 'orange'));

COMMIT;
