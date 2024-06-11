-- Messages table: stores text and media messages between users and tattoo artists
CREATE TABLE "message" (
                         message_id BIGSERIAL PRIMARY KEY,
                         sender_id UUID NOT NULL REFERENCES "user" (user_id) ON DELETE CASCADE,
                         receiver_id UUID NOT NULL REFERENCES "user" (user_id) ON DELETE CASCADE,
                         content TEXT,
                         media_url TEXT,
                         media_type TEXT,
                         created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON "message" (sender_id);
CREATE INDEX ON "message" (receiver_id);
