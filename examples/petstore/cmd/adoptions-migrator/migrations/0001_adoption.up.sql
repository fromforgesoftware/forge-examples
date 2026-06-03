CREATE TABLE adoptions.adoption (
    id         UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    owner      TEXT NOT NULL,
    pet_id     UUID NOT NULL,
    status     TEXT NOT NULL DEFAULT 'PLACED',
    fee_cents  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_adoption_owner ON adoptions.adoption (owner);
CREATE INDEX idx_adoption_pet ON adoptions.adoption (pet_id);
