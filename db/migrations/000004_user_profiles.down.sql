-- SQLite doesn't natively support DROP COLUMN easily before 3.35, but golang-migrate handles it via temp tables if strictly needed. 
-- For a basic down migration in modern SQLite:
ALTER TABLE users DROP COLUMN full_name;
ALTER TABLE users DROP COLUMN title;
ALTER TABLE users DROP COLUMN avatar_data;
