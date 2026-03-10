-- +goose Up
CREATE TABLE IF NOT EXISTS tbl_financial_year (
    id UUID         PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(), -- example: ab290cajdh
    label           TEXT PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(), -- example: FY 2025-26
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
)
-- +goose Down
--
