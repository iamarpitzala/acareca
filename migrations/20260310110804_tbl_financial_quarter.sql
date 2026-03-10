-- +goose Up
CREATE TABLE IF NOT EXISTS tbl_financial_quarter (
    id                  UUID PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    financial_year_id   UUID REFERENCES tbl_financial_year(id),
    label               TEXT NOT NULL UNIQUE,
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS tbl_financial_quarter;
