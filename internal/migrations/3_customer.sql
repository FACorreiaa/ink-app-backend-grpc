-- 4. appointments: Tracks booking info for each client
CREATE TABLE customers (
                       id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       studio_id     UUID NOT NULL,
  -- optionally track the specific artist if you want each client tied to a single artist
  -- user_id       UUID,
  -- references users(id) with role = 'ARTIST'
                       full_name     VARCHAR(150) NOT NULL,
                       email         VARCHAR(255),
                       phone         VARCHAR(50),
                       notes         TEXT,                       -- e.g. style preferences, special instructions
                       nif       VARCHAR(25),
                       address       VARCHAR(255),
                       city          VARCHAR(100),
                       postal_code   VARCHAR(25),
                       country       VARCHAR(100),
                       id_card_number VARCHAR(25),
                       first_name    VARCHAR(100),
                       last_name     VARCHAR(100),
                       birthday    DATE,
                       is_archived BOOLEAN,
                       created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at    TIMESTAMPTZ,
                       CONSTRAINT fk_studio_client
                         FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE
);

CREATE TABLE customer_artists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    user_id UUID NOT NULL,  -- assuming artists are in the users table
    studio_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_customer_artists_customer
        FOREIGN KEY (customer_id) REFERENCES customers (id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_artists_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_artists_studio
        FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE,
    CONSTRAINT unique_customer_artist UNIQUE (customer_id, user_id)
);