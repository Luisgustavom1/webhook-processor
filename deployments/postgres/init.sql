CREATE SCHEMA IF NOT EXISTS webhooks;

SET search_path TO webhooks;

CREATE TABLE webhooks (
    id                SERIAL PRIMARY KEY,
    subscribed_events TEXT[] NOT NULL,
    callback_url      TEXT NOT NULL,
    secret            TEXT NOT NULL,
    status            TEXT NOT NULL,
    failure_count     INTEGER NOT NULL DEFAULT 0,
    last_failure_at   TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_events (
    id               VARCHAR(26) PRIMARY KEY,
    webhook_id       INTEGER NOT NULL REFERENCES webhooks(id),
    event_type       TEXT NOT NULL,
    payload          JSONB NOT NULL,
    last_error       JSONB,
    response_body    JSONB,
    response_code    INTEGER,
    retries_count    INTEGER NOT NULL DEFAULT 0,
    status           TEXT NOT NULL,
    failed_at        TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
