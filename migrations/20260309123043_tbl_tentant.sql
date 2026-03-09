-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS tbl_tentant (
    id            SERIAL PRIMARY KEY,
    user_id       VARCHAR(40) NOT NULL,
    abn           VARCHAR(20),
    verifed       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tbl_tentant_setting   (
    id            SERIAL PRIMARY KEY,
    tentant_id    INTEGER NOT NULL REFERENCES tbl_tentant(id),
    timezone      VARCHAR(255) NOT NULL DEFAULT 'Australia/Sydney',
    logo          VARCHAR(255),
    color         VARCHAR(7) NOT NULL DEFAULT '#000000',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tbl_tentant;
DROP TABLE IF EXISTS tbl_tentant_setting;
-- +goose StatementEnd
