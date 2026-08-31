BEGIN;

CREATE TABLE IF NOT EXISTS public.user_api_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL,
  name text NOT NULL,
  token_hash text NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT now(),
  last_used_at timestamp without time zone,
  CONSTRAINT user_api_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_api_tokens_token_hash
  ON public.user_api_tokens (token_hash);

CREATE INDEX IF NOT EXISTS idx_user_api_tokens_user_id
  ON public.user_api_tokens (user_id);

-- Preserve every existing personal token as one named row so current
-- integrations keep working after the migration to the new table.
INSERT INTO public.user_api_tokens (user_id, name, token_hash, created_at)
SELECT id, 'Default token', token_hash, now()
FROM public.users
WHERE type = 'human' AND token_hash IS NOT NULL AND token_hash <> '';

COMMIT;
