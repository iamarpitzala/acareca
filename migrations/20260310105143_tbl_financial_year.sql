-- +goose Up
CREATE TABLE IF NOT EXISTS tbl_financial_year (
    id              UUID PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    label           TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS tbl_financial_year;
