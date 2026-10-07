BEGIN;

CREATE TABLE public.factory_work_order_comments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    factory_id uuid NOT NULL,
    work_order_id uuid NOT NULL,
    author_user_id uuid,
    author_kind text NOT NULL,
    automation jsonb,
    source_run_id uuid,
    body text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT factory_work_order_comments_pkey PRIMARY KEY (id),
    CONSTRAINT factory_work_order_comments_author_kind_check CHECK ((author_kind = ANY (ARRAY['user'::text, 'automation'::text]))),
    CONSTRAINT factory_work_order_comments_factory_id_fkey FOREIGN KEY (factory_id) REFERENCES public.factories(id) ON DELETE RESTRICT,
    CONSTRAINT factory_work_order_comments_work_order_id_fkey FOREIGN KEY (work_order_id) REFERENCES public.factory_work_orders(id) ON DELETE RESTRICT,
    CONSTRAINT factory_work_order_comments_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES public.users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_factory_work_order_comments_work_order_created
    ON public.factory_work_order_comments USING btree (work_order_id, created_at, id);

CREATE TABLE public.factory_work_order_comment_mentions (
    comment_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT factory_work_order_comment_mentions_pkey PRIMARY KEY (comment_id, user_id),
    CONSTRAINT factory_work_order_comment_mentions_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.factory_work_order_comments(id) ON DELETE CASCADE,
    CONSTRAINT factory_work_order_comment_mentions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_factory_work_order_comment_mentions_user_id
    ON public.factory_work_order_comment_mentions USING btree (user_id);

-- Copy historical comments from timeline events. Reuse the event id so
-- `order().comments[].id` stays stable for existing expressions.
INSERT INTO public.factory_work_order_comments (
    id,
    organization_id,
    factory_id,
    work_order_id,
    author_user_id,
    author_kind,
    automation,
    source_run_id,
    body,
    created_at,
    updated_at
)
SELECT
    e.id,
    o.organization_id,
    o.factory_id,
    e.work_order_id,
    CASE
        WHEN (e.data -> 'author' ->> 'userId') ~ '^[0-9a-fA-F-]{36}$'
             AND EXISTS (
                 SELECT 1
                 FROM public.users u
                 WHERE u.id = ((e.data -> 'author' ->> 'userId')::uuid)
             )
        THEN (e.data -> 'author' ->> 'userId')::uuid
        ELSE NULL
    END,
    CASE
        WHEN (e.data -> 'author' ->> 'kind') = 'automation' THEN 'automation'
        ELSE 'user'
    END,
    e.data -> 'author' -> 'automation',
    CASE
        WHEN (e.data -> 'run' ->> 'id') ~ '^[0-9a-fA-F-]{36}$'
        THEN (e.data -> 'run' ->> 'id')::uuid
        ELSE NULL
    END,
    COALESCE(e.data ->> 'body', ''),
    e.created_at,
    e.created_at
FROM public.factory_work_order_events e
INNER JOIN public.factory_work_orders o ON o.id = e.work_order_id
WHERE e.type = 'order.comment.added';

COMMIT;