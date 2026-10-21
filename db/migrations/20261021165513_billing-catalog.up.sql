CREATE TABLE billing_plans (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 key text NOT NULL UNIQUE CHECK (key ~ '^[a-z][a-z0-9_]*$'),
 name text NOT NULL CHECK (btrim(name) <> ''),
 category text NOT NULL CHECK (category IN ('none','trial','paid')),
 visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('public','private')),
 price_cents bigint NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
 per_seat boolean NOT NULL DEFAULT false,
 polar_product_id text UNIQUE,
 seat_limit integer CHECK (seat_limit > 0),
 parallel_task_limit integer CHECK (parallel_task_limit > 0),
 credit_amount_cents bigint NOT NULL DEFAULT 0 CHECK (credit_amount_cents >= 0),
 credit_frequency text NOT NULL DEFAULT 'none' CHECK (credit_frequency IN ('none','once','billing_period')),
 credit_expiration text NOT NULL DEFAULT 'never' CHECK (credit_expiration IN ('period_end','after_days','never')),
 credit_validity_days integer,
 trial_credit_cents bigint NOT NULL DEFAULT 0 CHECK (trial_credit_cents >= 0),
 trial_length_days integer NOT NULL DEFAULT 14 CHECK (trial_length_days > 0),
 markup_bps integer CHECK (markup_bps >= 0),
 features jsonb NOT NULL DEFAULT '[]',
 require_payment_method boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK (credit_expiration <> 'after_days' OR credit_validity_days > 0)
);

INSERT INTO billing_plans (key,name,category,visibility) VALUES ('none','None','none','private');
INSERT INTO billing_plans (key,name,category,visibility,trial_credit_cents)
 VALUES ('trial','Trial','trial','private', COALESCE((SELECT welcome_grant_cents FROM installation_llm_settings WHERE id=1),5000));
INSERT INTO billing_plans (key,name,category,visibility,price_cents,credit_amount_cents,credit_frequency,credit_expiration)
 VALUES ('business_legacy','Business','paid','private',19900,5000,'billing_period','period_end');

ALTER TABLE organization_billing_plans ADD COLUMN plan_id uuid REFERENCES billing_plans(id);
UPDATE organization_billing_plans o SET plan_id=p.id FROM billing_plans p WHERE p.key=CASE o.plan WHEN 'business' THEN 'business_legacy' ELSE o.plan END;
ALTER TABLE organization_billing_plans ALTER COLUMN plan_id SET NOT NULL;
ALTER TABLE organization_billing_plans DROP CONSTRAINT organization_billing_plans_plan;
CREATE INDEX organization_billing_plans_catalog ON organization_billing_plans(plan_id);

CREATE TABLE organization_feature_grants (
 organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
 feature_key text NOT NULL,
 enabled boolean NOT NULL DEFAULT true,
 PRIMARY KEY (organization_id,feature_key)
);

ALTER TABLE organization_llm_credit_grants DROP CONSTRAINT organization_llm_credit_grants_kind;
ALTER TABLE organization_llm_credit_grants DROP CONSTRAINT organization_llm_credit_grants_amount_sign;
ALTER TABLE organization_llm_credit_grants ADD CONSTRAINT organization_llm_credit_grants_kind CHECK (kind IN ('welcome','admin','included','topup','topup_refund','trial_adjustment','topup_adjustment','admin_adjustment','included_adjustment'));
ALTER TABLE organization_llm_credit_grants ADD CONSTRAINT organization_llm_credit_grants_amount_sign CHECK (
 (kind='topup_refund' AND amount_micros<0) OR
 (kind IN ('trial_adjustment','topup_adjustment','admin_adjustment','included_adjustment') AND amount_micros<>0) OR
 (kind IN ('welcome','admin','included','topup') AND amount_micros>0));

CREATE TABLE plan_feature_definitions (
 key text PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
 label text NOT NULL CHECK (btrim(label) <> ''),
 gated boolean NOT NULL DEFAULT true
);

INSERT INTO plan_feature_definitions (key, label, gated) VALUES
 ('byo_subscription', 'Bring your own subscription', false),
 ('byo_key', 'Bring your own API key', false),
 ('centralized_billing', 'Centralized billing', false);
