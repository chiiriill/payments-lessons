ALTER TABLE payments
    ADD CONSTRAINT payments_status_check
    CHECK (status IN ('pending', 'paid', 'failed', 'refunded'));

CREATE UNIQUE INDEX IF NOT EXISTS payments_provider_payment_id_uq
    ON payments(provider_payment_id)
    WHERE provider_payment_id != '';
