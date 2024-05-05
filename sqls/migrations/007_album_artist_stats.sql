-- +migrate Up
create table if not exists `albums_stats` (
    `album_id` integer,
    `favourite` integer null, -- Unix timestamp at which it was starred
    `user_rating` integer null, -- Value in the 1-5 range
    FOREIGN KEY(album_id) REFERENCES albums(id) ON UPDATE CASCADE ON DELETE CASCADE
);

create unique index if not exists `unique_album_stats` on `albums_stats` (`album_id`);

create table if not exists `artists_stats` (
    `artist_id` integer,
    `favourite` integer null, -- Unix timestamp at which it was starred
    `user_rating` integer null, -- Value in the 1-5 range
    FOREIGN KEY(artist_id) REFERENCES artists(id) ON UPDATE CASCADE ON DELETE CASCADE
);

create unique index if not exists `unique_artists_stats` on `artists_stats` (`artist_id`);

-- +migrate Down
drop index if exists `unique_album_stats`;
drop table if exists `albums_stats`;

drop index if exists `unique_artists_stats`;
drop table if exists `artists_stats`;
