-- +migrate Up

-- First deal with the `artists` table. An artist is expected to be present
-- only once in the database. So this should be represented in the schema.
create unique index if not exists unique_artist on `artists` ('name');

-- Now make sure albums are unique. This means an album in a particular directory
-- since one might have the same album multiple times. Different formats, possibly.
create unique index if not exists unique_albums on `albums` (`name`, `fs_path`);

-- Next is line is the tracks. A track is uniquely identified by its file location.
-- So make sure it is unique then!
drop index if exists tracks_paths;
create unique index if not exists unique_tracks on `tracks` ('fs_path');

-- +migrate Down

drop index if exists unique_artist;
drop index if exists unique_albums;
drop index if exists unique_tracks;
create index tracks_paths on `tracks` (`fs_path`);
