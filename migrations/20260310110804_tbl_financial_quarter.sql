-- +goose Up
CREATE TABLE IF NOT EXISTS tbl_financial_quarter (
    id                      UUID PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(), -- example: ab300hjai1
    financial_year_id       UUID REFERENCES tbl_financial_year(id), -- example: ab290cajdh
    label                   TEXT PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(), -- example: Q1-25-26 Jun-Aug
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    start_date              DATE NOT NULL,
    end_date                DATE NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
)
-- +goose Down
--
