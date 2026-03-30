-- +goose Up
-- +goose StatementBegin

-- Widen form name to support longer descriptive names
ALTER TABLE tbl_form
    ALTER COLUMN name TYPE VARCHAR(255);

-- Allow section_type, coa_id, payment_responsibility to be NULL for computed fields
ALTER TABLE tbl_form_field
    ALTER COLUMN section_type DROP NOT NULL,
    ALTER COLUMN coa_id       DROP NOT NULL,
    ALTER COLUMN payment_responsibility DROP NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE tbl_form_field
    ALTER COLUMN section_type SET NOT NULL,
    ALTER COLUMN coa_id       SET NOT NULL,
    ALTER COLUMN payment_responsibility SET NOT NULL;

ALTER TABLE tbl_form
    ALTER COLUMN name TYPE VARCHAR(40);

-- +goose StatementEnd
