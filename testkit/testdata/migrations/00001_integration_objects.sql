-- +goose Up
CREATE TABLE IF NOT EXISTS integration_objects (
    id BIGSERIAL PRIMARY KEY,
    object_key TEXT NOT NULL UNIQUE,
    byte_size INT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS integration_objects;
