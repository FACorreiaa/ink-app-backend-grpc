-- 5. conversations: High-level “threads” for messaging
-- Could be 1:1 (artist <-> client), or a group thread with multiple staff
CREATE TABLE conversations (
                             id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                             studio_id     UUID NOT NULL,
                             client_id     UUID NOT NULL,
  -- You can add a unique constraint if you want exactly one conversation per client-artist pair
  -- or do multi-artist group conversations by linking more than one user
                             subject       VARCHAR(200),
                             created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                             updated_at    TIMESTAMPTZ,
                             CONSTRAINT fk_studio_convo
                               FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE,
                             CONSTRAINT fk_client_convo
                               FOREIGN KEY (client_id) REFERENCES clients (id) ON DELETE CASCADE
);

-- 6. conversation_participants: which users (artists/staff) are part of a conversation
CREATE TABLE conversation_participants (
                                         conversation_id  UUID NOT NULL,
                                         user_id          UUID NOT NULL,
                                         PRIMARY KEY (conversation_id, user_id),
                                         CONSTRAINT fk_conversation
                                           FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE CASCADE,
                                         CONSTRAINT fk_user
                                           FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
