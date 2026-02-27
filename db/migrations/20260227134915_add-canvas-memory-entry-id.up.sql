ALTER TABLE canvas_memories
ADD COLUMN IF NOT EXISTS id UUID;

UPDATE canvas_memories
SET id = gen_random_uuid()
WHERE id IS NULL;

ALTER TABLE canvas_memories
ALTER COLUMN id SET DEFAULT gen_random_uuid(),
ALTER COLUMN id SET NOT NULL;

ALTER TABLE canvas_memories
ADD CONSTRAINT canvas_memories_pkey PRIMARY KEY (id);
