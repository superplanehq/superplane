ALTER TABLE public.factory_velocity_repository_merges
	ADD COLUMN IF NOT EXISTS author_uuid text NOT NULL DEFAULT '';
