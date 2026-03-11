-- +goose Up
-- +goose StatementBegin

CREATE TYPE section_type AS ENUM('COLLECTION', 'COST', 'OTHER_COST');

CREATE TYPE payment_responsibility AS ENUM('OWNER', 'CLINIC');

CREATE TYPE tax_type AS ENUM('INCLUSIVE', 'EXCLUSIVE', 'MANUAL');

CREATE TABLE IF NOT EXISTS tbl_form_field(
    id UUID PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    form_version_id UUID NOT NULL REFERENCES tbl_custom_form_version(id) ON DELETE CASCADE,
    label VARCHAR(255) NOT NULL,
    section_type section_type NOT NULL,

    payment_responsibility payment_responsibility NOT NULL,
    tax_type tax_type NOT NULL,

    coa_id UUID NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tbl_form_field_form_version_id ON tbl_form_field(form_version_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tbl_form_field;
DROP TYPE IF EXISTS section_type;
DROP TYPE IF EXISTS payment_responsibility;
DROP TYPE IF EXISTS tax_type;
-- +goose StatementEnd
