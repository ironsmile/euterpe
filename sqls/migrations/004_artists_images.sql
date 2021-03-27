-- +migrate Up

create table `artists_images` (
    `id` integer not null primary key, 
    `artist_id` integer unique,
    `image` blob default null,
    `updated_at` integer
);

create index artists_images_artist_ids on `artists_images` (`artist_id`);

-- +migrate Down

drop table `artists_images`;
