-- +goose Up
-- +goose StatementBegin
CREATE TABLE ship_structure_imports (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  ship_id           uuid NOT NULL REFERENCES ships(id),
  idempotency_key   text,
  status            text NOT NULL DEFAULT 'validating'
                      CHECK (status IN ('uploaded','validating','ready','failed','processing','completed')),
  file_name         text NOT NULL,
  file_size_bytes   bigint NOT NULL CHECK (file_size_bytes >= 0),
  file_sha256       text NOT NULL CHECK (file_sha256 ~ '^[0-9a-f]{64}$'),
  decks_total       integer NOT NULL DEFAULT 0 CHECK (decks_total >= 0),
  decks_valid       integer NOT NULL DEFAULT 0 CHECK (decks_valid >= 0),
  decks_invalid     integer NOT NULL DEFAULT 0 CHECK (decks_invalid >= 0),
  areas_total       integer NOT NULL DEFAULT 0 CHECK (areas_total >= 0),
  areas_valid       integer NOT NULL DEFAULT 0 CHECK (areas_valid >= 0),
  areas_invalid     integer NOT NULL DEFAULT 0 CHECK (areas_invalid >= 0),
  rooms_total       integer NOT NULL DEFAULT 0 CHECK (rooms_total >= 0),
  rooms_valid       integer NOT NULL DEFAULT 0 CHECK (rooms_valid >= 0),
  rooms_invalid     integer NOT NULL DEFAULT 0 CHECK (rooms_invalid >= 0),
  validated_payload jsonb,
  validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb,
  error_message     text,
  created_by        text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  completed_at      timestamptz,
  CHECK (decks_valid + decks_invalid = decks_total),
  CHECK (areas_valid + areas_invalid = areas_total),
  CHECK (rooms_valid + rooms_invalid = rooms_total)
);

CREATE UNIQUE INDEX uq_ship_structure_imports_idem
  ON ship_structure_imports (ship_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;

CREATE INDEX ix_ship_structure_imports_ship_created
  ON ship_structure_imports (ship_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ship_structure_imports;
-- +goose StatementEnd
