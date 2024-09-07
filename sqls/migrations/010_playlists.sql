-- migrate Up

CREATE TABLE IF NOT EXISTS `playlists` (
    `id` integer not null primary key,
    `name` text not null,
    `description` text null,
    `public` integer default 1,
    `created_at` integer not null,
    `updated_at` integer not null
);

-- migrate Down
drop table if exists `playlists`;
