-- 9. social_integrations: Optional table to store tokens for Instagram, WhatsApp, etc.
CREATE TABLE social_integrations (
                                   id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                   studio_id     UUID NOT NULL,
                                   provider      VARCHAR(50) NOT NULL,         -- 'INSTAGRAM', 'WHATSAPP', 'PINTEREST', etc.
                                   access_token  TEXT NOT NULL,
                                   refresh_token TEXT,
                                   expires_at    TIMESTAMPTZ,
                                   created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   updated_at    TIMESTAMPTZ,
                                   CONSTRAINT fk_studio_social
                                     FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE
);
