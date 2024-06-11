-- Portfolio table: stores portfolio items for tattoo artists
CREATE TABLE "portfolio" (
                           portfolio_id BIGSERIAL PRIMARY KEY,
                           artist_id UUID NOT NULL REFERENCES "user" (user_id) ON DELETE CASCADE,
                           title TEXT NOT NULL,
                           description TEXT,
                           media_url TEXT,
                           media_type TEXT,
                           created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON "portfolio" (artist_id);

-- Create function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION set_updated_at()
  RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the trigger to the user and appointment tables
CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON "appointment"
  FOR EACH ROW
EXECUTE PROCEDURE set_updated_at();
