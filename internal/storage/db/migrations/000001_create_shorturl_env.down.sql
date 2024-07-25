BEGIN TRANSACTION;

DROP INDEX IF EXISTS unique_original_url;
DROP TABLE IF EXISTS public.short_links;

COMMIT ;