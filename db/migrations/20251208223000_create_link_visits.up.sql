-- +goose Up
CREATE TABLE link_visits (
    id         BIGSERIAL PRIMARY KEY,
    link_id    BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    ip         VARCHAR(255) NOT NULL,
    user_agent TEXT,
    referer    TEXT,
    status     INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX link_visits_link_id_idx ON link_visits (link_id);
CREATE INDEX link_visits_created_at_idx ON link_visits (created_at);

-- +goose Down
DROP TABLE IF EXISTS link_visits;
