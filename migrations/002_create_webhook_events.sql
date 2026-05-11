CREATE TABLE IF NOT EXISTS webhook_events (
    event_id    text        PRIMARY KEY,
    processed_at timestamptz NOT NULL DEFAULT now()
);
