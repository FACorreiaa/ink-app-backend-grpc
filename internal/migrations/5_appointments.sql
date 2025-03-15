-- 4. appointments: Tracks booking info for each client
CREATE TABLE appointments (
                            id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                            studio_id     UUID NOT NULL,
                            customers_id     UUID NOT NULL,
                            artist_id     UUID,                       -- references users(id) with role='ARTIST'
                            start_time    TIMESTAMPTZ NOT NULL,
                            end_time      TIMESTAMPTZ NOT NULL,
                            status        VARCHAR(50) NOT NULL,       -- e.g. 'SCHEDULED', 'COMPLETED', 'CANCELED'
                            notes         TEXT,                       -- e.g. deposit, location, design references
                            created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                            updated_at    TIMESTAMPTZ,
                            CONSTRAINT fk_studio_appointment
                              FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE,
                            CONSTRAINT fk_client_appointment
                              FOREIGN KEY (customers_id) REFERENCES customers (id) ON DELETE CASCADE,
                            CONSTRAINT fk_artist_appointment
                              FOREIGN KEY (artist_id) REFERENCES users (id)
);
