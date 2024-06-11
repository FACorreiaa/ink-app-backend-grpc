-- Appointments table: stores information about scheduled meetings and video calls
CREATE TABLE "appointment" (
                             appointment_id BIGSERIAL PRIMARY KEY,
                             user_id UUID NOT NULL REFERENCES "user" (user_id) ON DELETE CASCADE,
                             artist_id UUID NOT NULL REFERENCES "user" (user_id) ON DELETE CASCADE,
                             scheduled_time TIMESTAMPTZ NOT NULL,
                             meeting_url TEXT,
                             status TEXT NOT NULL DEFAULT 'scheduled',
                             created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                             updated_at TIMESTAMPTZ
);

CREATE INDEX ON "appointment" (user_id);
CREATE INDEX ON "appointment" (artist_id);
