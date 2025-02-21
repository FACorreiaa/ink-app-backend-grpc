-- 4. appointments: Tracks booking info for each client
CREATE TABLE clients (
                       id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       studio_id     UUID NOT NULL,
  -- optionally track the specific artist if you want each client tied to a single artist
  -- user_id       UUID,
  -- references users(id) with role = 'ARTIST'
                       full_name     VARCHAR(150) NOT NULL,
                       email         VARCHAR(255),
                       phone         VARCHAR(50),
                       notes         TEXT,                       -- e.g. style preferences, special instructions
                       created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at    TIMESTAMPTZ,
                       CONSTRAINT fk_studio_client
                         FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE
);
