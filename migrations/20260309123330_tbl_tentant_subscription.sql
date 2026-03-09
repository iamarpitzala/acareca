-- +goose Up
-- +goose StatementBegin

CREATE TYPE tentant_subscription_status AS ENUM ('active', 'past_due', 'cancelled', 'paused', 'expired');
CREATE TABLE IF NOT EXISTS tbl_tentant_subscription (
    id            SERIAL PRIMARY KEY,
    tentant_id    INTEGER NOT NULL REFERENCES tbl_tentant(id),
    subscription_id INTEGER NOT NULL,
    start_date    TIMESTAMPTZ NOT NULL,
    end_date      TIMESTAMPTZ NOT NULL,
    status        tentant_subscription_status NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tbl_tentant_subscription;
DROP TYPE IF EXISTS tentant_subscription_status;
-- +goose StatementEnd
