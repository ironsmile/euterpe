-- +migrate Up
create table if not exists `user_stats` (
    `track_id` integer,
    `favourite` integer null, -- Unix timestamp at which it was starred
    `user_rating` integer null,
    `last_played` integer null, -- Unix timestamp in seconds.
    `play_count` integer not null default 0,
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON UPDATE CASCADE ON DELETE CASCADE
);

create unique index if not exists `unique_user_stats` on `user_stats` (`track_id`);

-- +migrate Down
drop index if exists `unique_user_stats`;
drop table if exists `user_stats`;
