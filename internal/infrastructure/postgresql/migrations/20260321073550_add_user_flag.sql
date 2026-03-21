-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
ADD COLUMN flag INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
DROP COLUMN flag;

-- +goose StatementEnd