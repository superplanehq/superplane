-- A seat purchase that a caller starts stores its operation key, the
-- fingerprint of the request, and the billing period it was made for.
-- Existing rows keep NULL in all four columns: no key, fingerprint, or
-- period is invented for them.
ALTER TABLE organization_seat_purchases
    ADD COLUMN operation_key text,
    ADD COLUMN request_fingerprint text,
    ADD COLUMN period_start timestamp with time zone,
    ADD COLUMN period_end timestamp with time zone;

ALTER TABLE organization_seat_purchases
    ADD CONSTRAINT organization_seat_purchases_operation_complete CHECK (
        (operation_key IS NULL AND request_fingerprint IS NULL AND period_start IS NULL AND period_end IS NULL)
        OR (operation_key IS NOT NULL AND request_fingerprint IS NOT NULL AND period_start IS NOT NULL AND period_end IS NOT NULL)
    ),
    ADD CONSTRAINT organization_seat_purchases_operation_key_length CHECK (
        operation_key IS NULL OR char_length(operation_key) BETWEEN 1 AND 128
    );

CREATE UNIQUE INDEX uq_organization_seat_purchases_operation_key
    ON organization_seat_purchases (organization_id, operation_key)
    WHERE operation_key IS NOT NULL;
