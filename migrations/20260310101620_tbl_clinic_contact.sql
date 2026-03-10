-- +goose Up
CREATE TABLE IF NOT EXISTS tbl_clinic_contact (
    id UUID         PRIMARY KEY NOT NULL UNIQUE DEFAULT uuid_generate_v4(), 
    clinic_id UUID  NOT NULL REFERENCES tbl_clinic(id) ON DELETE CASCADE 
    contact_type    clinic_contact_type_enum NOT NULL, --PHONE | EMAIL | WEBSITE | FAX
    value           TEXT NOT NULL, 
    label           TEXT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(), 
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
) -- +goose Down
--