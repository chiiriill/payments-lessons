CREATE TABLE IF NOT EXISTS outbox_events (
    id           bigserial   PRIMARY KEY,
    event_type   text        NOT NULL,
    aggregate_id text        NOT NULL,
    payload      jsonb       NOT NULL DEFAULT '{}',
    status       text        NOT NULL DEFAULT 'new',
    attempts     int         NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
