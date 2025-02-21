-- 8. portfolio: For images / design references / completed work
CREATE TABLE portfolio_items (
                               id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               studio_id     UUID NOT NULL,
                               artist_id     UUID NOT NULL, -- references users(id) with role='ARTIST'
                               image_url     TEXT NOT NULL,
                               title         VARCHAR(200),
                               description   TEXT,
                               created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                               updated_at    TIMESTAMPTZ,
                               CONSTRAINT fk_studio_portfolio
                                 FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE,
                               CONSTRAINT fk_artist_portfolio
                                 FOREIGN KEY (artist_id) REFERENCES users (id)
);
