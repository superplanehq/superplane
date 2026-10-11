ALTER TABLE public.factory_velocity_repository_merges
	ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 'github';

UPDATE public.factory_velocity_repository_merges
SET provider = 'github'
WHERE provider IS NULL OR provider = '';

DROP INDEX IF EXISTS public.idx_factory_velocity_repository_merges_factory_repo_number;

CREATE UNIQUE INDEX IF NOT EXISTS idx_factory_velocity_merges_factory_provider_repo_num
	ON public.factory_velocity_repository_merges USING btree (factory_id, provider, repository, number);
