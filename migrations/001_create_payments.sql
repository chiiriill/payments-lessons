CREATE TABLE IF NOT EXISTS payments (
    id text PRIMARY KEY,
    order_id text NOT NULL,
    amount bigint NOT NULL CHECK (amount > 0),
    currency text NOT NULL,
    description text NOT NULL DEFAULT '',
    status text NOT NULL,
    payment_url text NOT NULL DEFAULT '',
    provider_payment_id text NOT NULL DEFAULT '',
    idempotency_key text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS payments_idempotency_key_uq
    ON payments(idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS outbox_events (
    id bigserial PRIMARY KEY,
    event_type text NOT NULL,
    aggregate_id text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL DEFAULT 'new',
    attempts int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
