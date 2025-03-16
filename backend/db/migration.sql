-- Migration script to add new columns to the torrents table
ALTER TABLE torrents ADD COLUMN length INTEGER;
ALTER TABLE torrents ADD COLUMN files TEXT;
ALTER TABLE torrents ADD COLUMN downloaded INTEGER;
ALTER TABLE torrents ADD COLUMN progress REAL;
ALTER TABLE torrents ADD COLUMN state TEXT;
