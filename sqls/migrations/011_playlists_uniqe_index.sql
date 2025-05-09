-- +migrate Up
drop index if exists playlist_pairs;
create unique index if not exists playlist_pairs_with_index on `playlists_tracks` ('playlist_id', `index`);

-- +migrate Down
drop index if exists playlist_pairs_with_index;
create unique index if not exists playlist_pairs on `playlists_tracks` ('playlist_id', `track_id`);
