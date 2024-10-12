-- +migrate Up
CREATE TABLE IF NOT EXISTS `playlists` (
    `id` integer not null primary key,
    `name` text not null,
    `description` text null,
    `public` integer default 1,
    `created_at` integer not null,
    `updated_at` integer not null
);

CREATE TABLE IF NOT EXISTS `playlists_tracks` (
    `playlist_id` integer not null,
    `track_id` integer not null,
    `index` integer not null default 0,
    FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `playlists_images` (
    `playlist_id` integer unique not null,
    `image` blob not null,
    `updated_at` integer not null,
    FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON UPDATE CASCADE ON DELETE CASCADE
);

create unique index if not exists playlist_pairs on `playlists_tracks` ('playlist_id', `track_id`);

-- +migrate Down
drop table if exists `playlists`;
drop table if exists `playlists_tracks`;
drop table if exists `playlists_images`;
