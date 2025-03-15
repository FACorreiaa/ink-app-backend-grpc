-- 7. messages: The actual chat messages
CREATE TABLE messages (
                        id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        conversation_id   UUID NOT NULL,
                        sender_user_id    UUID,                    -- references a user if staff sends it
                        sender_customer_id  UUID,                    -- references a client if the client sends it
                        content           TEXT,
                        created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- store references to file uploads if needed
                        CONSTRAINT fk_conversation_message
                          FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE CASCADE,
                        CONSTRAINT fk_sender_user
                          FOREIGN KEY (sender_user_id) REFERENCES users (id),
                        CONSTRAINT fk_sender_client
                          FOREIGN KEY (sender_customer_id) REFERENCES customers (id)
);
