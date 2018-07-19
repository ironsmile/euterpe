-- +migrate Up

create table `albums_artworks` (
    `id` integer not null primary key, 
    `album_id` integer unique,
    `artwork_cover` blob default null,
    `updated_at` integer
);

create index albums_artwork_album_ids on `albums_artworks` (`album_id`);

-- +migrate Down

drop table `albums_artworks`;
