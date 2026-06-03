CREATE TABLE catalog.pet (
    id         UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    name       TEXT NOT NULL,
    species    TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'AVAILABLE'
);

CREATE INDEX idx_pet_status ON catalog.pet (status);
