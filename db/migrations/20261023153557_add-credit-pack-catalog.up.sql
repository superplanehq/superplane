CREATE TABLE credit_packs (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (length(trim(name)) > 0),
    amount_cents bigint NOT NULL CHECK (amount_cents = 0 OR amount_cents BETWEEN 50 AND 99999999),
    enabled boolean NOT NULL DEFAULT false,
    polar_product_id text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX credit_packs_polar_product_id_idx
    ON credit_packs (polar_product_id) WHERE deleted_at IS NULL;
