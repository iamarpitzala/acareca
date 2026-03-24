-- +goose Up
-- +goose StatementBegin

ALTER TABLE tbl_form_entry 
ADD COLUMN IF NOT EXISTS entry_date TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE tbl_form_entry_value 
ADD COLUMN IF NOT EXISTS entry_date TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE tbl_form_entry_value 
DROP CONSTRAINT IF EXISTS tbl_form_entry_value_entry_id_form_field_id_key;

ALTER TABLE tbl_form_entry_value 
ALTER COLUMN updated_at DROP DEFAULT,
ALTER COLUMN updated_at DROP NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- CAUTION: This will fail if you have created history records!
ALTER TABLE tbl_form_entry_value 
ADD CONSTRAINT tbl_form_entry_value_entry_id_form_field_id_key UNIQUE(entry_id, form_field_id);

ALTER TABLE tbl_form_entry_value 
ALTER COLUMN updated_at SET DEFAULT now(),
ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE tbl_form_entry_value DROP COLUMN IF EXISTS entry_date;
ALTER TABLE tbl_form_entry DROP COLUMN IF EXISTS entry_date;

-- +goose StatementEnd