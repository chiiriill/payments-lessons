CREATE TABLE IF NOT EXISTS payments (
    id                 TEXT        PRIMARY KEY,
    order_id           TEXT        NOT NULL,
    amount             BIGINT      NOT NULL CHECK (amount > 0),
    currency           TEXT        NOT NULL,
    description        TEXT        NOT NULL DEFAULT '',
    status             TEXT        NOT NULL CHECK (status IN ('pending', 'paid', 'failed', 'refunded')),
    payment_url        TEXT        NOT NULL DEFAULT '',
    provider_payment_id TEXT       NOT NULL DEFAULT '',
    idempotency_key    TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS payments_idempotency_key_uq
    ON payments(idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS payments_provider_payment_id_uq
    ON payments(provider_payment_id)
    WHERE provider_payment_id != '';

CREATE TABLE IF NOT EXISTS webhook_events (
    event_id     TEXT        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id           BIGSERIAL   PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    aggregate_id TEXT        NOT NULL,
    payload      JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);
