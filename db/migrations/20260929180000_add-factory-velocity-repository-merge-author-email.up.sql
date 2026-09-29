BEGIN;

-- The public GitHub email of a merge's author, when their profile shows one.
-- The velocity report matches it against a member's SuperPlane email, so a
-- member who never linked a GitHub account still gets one row instead of a
-- second one keyed by their login.
ALTER TABLE factory_velocity_repository_merges
  ADD COLUMN author_email TEXT NOT NULL DEFAULT '';

COMMIT;
