-- Seat quantity operations record an explicit next-renewal seat change.
-- A row is written only when an Admin removes seats, a purchase supersedes
-- that removal, protective recovery cancels it, or legacy cleanup records a
-- verified automatic reduction. Renewal never inserts a row.
--
-- Forward recovery: stop the seat-removal worker, then repair rows from
-- Polar evidence. Do not drop the table to undo a bad row.

ALTER TABLE organization_billing_plans
    ADD COLUMN seat_change_version bigint NOT NULL DEFAULT 0,
    ADD COLUMN seat_admission_limit integer,
    ADD COLUMN seat_proved_target integer,
    ADD COLUMN seat_quantity_operation_id uuid,
    ADD COLUMN seat_bound_period_end timestamp with time zone;

ALTER TABLE organization_billing_plans
    ADD CONSTRAINT organization_billing_plans_seat_admission_limit_check
        CHECK (seat_admission_limit IS NULL OR seat_admission_limit >= 1),
    ADD CONSTRAINT organization_billing_plans_seat_proved_target_check
        CHECK (seat_proved_target IS NULL OR seat_proved_target >= 1);

CREATE TABLE seat_quantity_operations (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_api_key_id uuid REFERENCES user_api_tokens(id) ON DELETE SET NULL,
    origin text NOT NULL,
    action text NOT NULL,
    subscription_id text NOT NULL,
    period_start timestamp with time zone NOT NULL,
    period_end timestamp with time zone NOT NULL,
    source_paid_quantity integer NOT NULL,
    target_quantity integer NOT NULL,
    effective_at timestamp with time zone NOT NULL,
    operation_key text NOT NULL,
    request_fingerprint text NOT NULL,
    revision integer NOT NULL,
    predecessor_id uuid,
    superseder_id uuid,
    seat_purchase_id uuid REFERENCES organization_seat_purchases(id) ON DELETE SET NULL,
    provider_evidence jsonb,
    status text NOT NULL,
    cause text NOT NULL DEFAULT '',
    seat_change_version bigint NOT NULL,
    dispatched_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT seat_quantity_operations_origin_check CHECK (
        origin IN ('admin', 'purchase', 'protective', 'legacy_cleanup')
    ),
    CONSTRAINT seat_quantity_operations_action_check CHECK (
        action IN ('schedule', 'change', 'cancel', 'supersede', 'protective_cancel', 'legacy_cleanup')
    ),
    CONSTRAINT seat_quantity_operations_status_check CHECK (
        status IN (
            'submitting', 'unknown', 'scheduled', 'rejected',
            'applied', 'canceled', 'superseded', 'repair_required'
        )
    ),
    CONSTRAINT seat_quantity_operations_quantities_check CHECK (
        source_paid_quantity >= 1 AND target_quantity >= 1
    ),
    CONSTRAINT seat_quantity_operations_revision_check CHECK (revision >= 1),
    CONSTRAINT seat_quantity_operations_key_length CHECK (
        char_length(operation_key) BETWEEN 1 AND 128
    ),
    CONSTRAINT seat_quantity_operations_fingerprint_check CHECK (
        char_length(request_fingerprint) > 0
    ),
    CONSTRAINT seat_quantity_operations_predecessor_fk
        FOREIGN KEY (predecessor_id) REFERENCES seat_quantity_operations(id),
    CONSTRAINT seat_quantity_operations_superseder_fk
        FOREIGN KEY (superseder_id) REFERENCES seat_quantity_operations(id)
);

CREATE UNIQUE INDEX uq_seat_quantity_operations_operation_key
    ON seat_quantity_operations (organization_id, operation_key);

CREATE UNIQUE INDEX uq_seat_quantity_operations_inflight
    ON seat_quantity_operations (organization_id)
    WHERE status IN ('submitting', 'unknown');

CREATE UNIQUE INDEX uq_seat_quantity_operations_scheduled
    ON seat_quantity_operations (organization_id)
    WHERE status = 'scheduled';

CREATE INDEX idx_seat_quantity_operations_reconcile
    ON seat_quantity_operations (organization_id)
    WHERE status IN ('submitting', 'unknown', 'scheduled', 'repair_required');

-- An admin or purchase operation needs an actor when it is registered.
-- Deleting that actor's token sets actor_api_key_id to null. The row stays.
-- A check constraint would reject that update, so the requirement is insert-only.
CREATE FUNCTION seat_quantity_operations_require_actor() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.origin NOT IN ('protective', 'legacy_cleanup')
       AND NEW.actor_user_id IS NULL
       AND NEW.actor_api_key_id IS NULL THEN
        RAISE EXCEPTION 'seat quantity operation requires an actor'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER seat_quantity_operations_actor
    BEFORE INSERT ON seat_quantity_operations
    FOR EACH ROW
    EXECUTE FUNCTION seat_quantity_operations_require_actor();
