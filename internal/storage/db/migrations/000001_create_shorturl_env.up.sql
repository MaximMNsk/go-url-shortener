BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS public.short_links
    (
        id serial PRIMARY KEY,
        original_url  TEXT  NOT NULL,
        short_url  TEXT  NOT NULL,
        uid  TEXT  NOT NULL,
        user_id  TEXT  NULL,
        is_deleted BOOLEAN DEFAULT FALSE
    );

CREATE UNIQUE INDEX IF NOT EXISTS unique_original_url
    ON public.short_links(original_url);

COMMIT ;