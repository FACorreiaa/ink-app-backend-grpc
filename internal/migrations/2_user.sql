-- 1. studios: The “tenant” or main account for each studio/artist setup
CREATE TABLE studios (
                       id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       name          VARCHAR(150) NOT NULL,
                       subdomain     VARCHAR(100) UNIQUE,        -- e.g., "inkbyjohn" => "inkbyjohn.myplatform.com"
                       created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at    TIMESTAMPTZ
);

CREATE TABLE studio_settings (
  studio_id UUID PRIMARY KEY REFERENCES studios(id) ON DELETE CASCADE,
  logo_url TEXT,
  theme VARCHAR(50),
  business_hours JSONB,
  contact_info JSONB,
  social_media JSONB,
  preferences JSONB,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);



-- 2. users: Staff members and owners within a studio (e.g. owner = main artist, or multiple staff)
CREATE TABLE users (
                     id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                     studio_id     UUID NOT NULL,
                     email         VARCHAR(255) UNIQUE NOT NULL,
                     hashed_password TEXT NOT NULL,            -- store hashed password (if not using external OAuth)
                     role          VARCHAR(50) NOT NULL,       -- e.g. 'OWNER', 'ARTIST', 'ASSISTANT', etc.
                     display_name  VARCHAR(150),
                     username VARCHAR(150),
                     first_name VARCHAR(150),
                     last_name VARCHAR(150),
                     created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                     updated_at    TIMESTAMPTZ,
                     CONSTRAINT fk_studio_user
                       FOREIGN KEY (studio_id) REFERENCES studios (id) ON DELETE CASCADE
);

CREATE TABLE invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  studio_id UUID NOT NULL REFERENCES studios(id) ON DELETE CASCADE,
  email VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL,
  token VARCHAR(255) UNIQUE NOT NULL,
  invited_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ
);

CREATE TABLE role_permissions (
  role VARCHAR(50) NOT NULL,
  permission VARCHAR(100) NOT NULL,
  PRIMARY KEY (role, permission)
);

INSERT INTO studios (id, name, subdomain, created_at, updated_at)
VALUES (
         'a1b2c3d4-e5f6-47f8-9a1b-2c3d4e5f6071'
         , -- Random UUID
         'Ink Odyssey Studio',
         'inkodyssey', -- Maps to inkodyssey.myplatform.com
         NOW(),
         NOW()
       );

INSERT INTO users (
  studio_id,
  email,
  hashed_password,
  role,
  display_name,
  username,
  first_name,
  last_name,
  created_at,
  updated_at
) VALUES (
           'a1b2c3d4-e5f6-47f8-9a1b-2c3d4e5f6071', -- Matches studio_id above
           'jane@inkodyssey.com',
           '$2a$10$5nX5gQz8eK8z5J5q8x5z5e5Qz8eK8z5J5q8x5z5e5Qz8eK8z5J5q8', -- Hash for "password123"
           'OWNER',
           'Jane Ink',
           'janeink',
           'Jane',
           'Doe',
           NOW(),
           NOW()
         );
