-- +goose Up
-- +goose StatementBegin

-- Create the ENUM type for notification status
CREATE TYPE enum_notification_status AS ENUM (
    'PENDING',    -- Notification has been created but not yet delivered
    'DELIVERED',  -- Notification has been delivered to the user (e.g., push/email sent)
    'READ',       -- Notification has been marked as read by the user
    'DISMISSED',  -- Notification has been dismissed/archived by the user
    'FAILED'      -- Delivery of notification failed
);

-- Main notification table to store user notifications
CREATE TABLE IF NOT EXISTS tbl_notification (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),           -- Unique notification ID
    recipient_id    UUID        NOT NULL,                                 -- User who receives the notification
    sender_id       UUID        NULL,                                     -- User who triggered the notification (nullable)
    event_type      VARCHAR(64) NOT NULL,                                 -- Type of event (e.g., invite.accepted)
    entity_type     VARCHAR(64) NOT NULL,                                 -- Associated entity type (e.g., "clinic", "invite")
    entity_id       UUID        NOT NULL,                                 -- Associated entity ID
    status          enum_notification_status NOT NULL DEFAULT 'PENDING',  -- Notification state
    payload         JSONB       NOT NULL DEFAULT '{}',                    -- Additional event data for message rendering
    retry_count     INT         NOT NULL DEFAULT 0,                       -- Number of delivery attempts (retries)
    readed_at       TIMESTAMPTZ,                                          -- Timestamp when marked as read (nullable)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()                    -- Creation timestamp
);

-- Index for fast lookup by recipient and status
CREATE INDEX idx_notifications_recipient_status
    ON tbl_notification (recipient_id, status);

-- Index for fast retrieval of notifications by user and recent creation time
CREATE INDEX idx_notifications_recipient_created
    ON tbl_notification (recipient_id, created_at DESC);

-- Outbox table to store pending notification events before they're processed
CREATE TABLE IF NOT EXISTS tbl_notification_outbox (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),           -- Unique outbox record ID
    event_type      VARCHAR(64)  NOT NULL,                                -- Type of notification event
    actor_id        UUID         NOT NULL,                                -- User/entity that triggered the event
    entity_type     VARCHAR(64)  NOT NULL,                                -- Entity type associated with the event
    entity_id       UUID         NOT NULL,                                -- Entity ID associated with the event
    payload         JSONB        NOT NULL DEFAULT '{}',                   -- Event details as JSON
    processed_at    TIMESTAMPTZ,                                          -- Timestamp when outbox event was processed
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()                   -- Creation timestamp
);

-- Index to efficiently select oldest unprocessed outbox events
CREATE INDEX idx_outbox_unprocessed
    ON tbl_notification_outbox (created_at ASC)
    WHERE processed_at IS NULL;

-- User notification channel preferences per event type (e.g., in_app, email, push)
CREATE TABLE IF NOT EXISTS tbl_notification_preferences (
    entity_id     UUID        NOT NULL,                                   -- User/entity ID
    event_type    VARCHAR(64) NOT NULL,                                   -- Type of notification event
    channels      JSONB       NOT NULL DEFAULT '["in_app"]',              -- Array of preferred channels as JSON
    PRIMARY KEY (entity_id, event_type)
);



-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS tbl_notification_preferences;
DROP TABLE IF EXISTS tbl_notification_outbox;
DROP TABLE IF EXISTS tbl_notification;
DROP TYPE IF EXISTS enum_notification_status;

-- +goose StatementEnd