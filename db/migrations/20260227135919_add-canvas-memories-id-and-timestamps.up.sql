ALTER TABLE canvas_memories
  ADD COLUMN IF NOT EXISTS id UUID DEFAULT gen_random_uuid(),
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE canvas_memories
SET id = gen_random_uuid()
WHERE id IS NULL;

ALTER TABLE canvas_memories
  ALTER COLUMN id SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'canvas_memories_pkey'
  ) THEN
    ALTER TABLE canvas_memories
      ADD CONSTRAINT canvas_memories_pkey PRIMARY KEY (id);
  END IF;
END $$;
