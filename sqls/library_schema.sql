create table `albums` (
    `id` integer not null primary key, 
    `name` text,
    `fs_path` text
);

create table `artists` (
    `id` integer not null primary key, 
    `name` text
);

create table `tracks` (
    `id` integer not null primary key,
    `album_id` integer,
    `artist_id` integer,
    `name` text,
    `number` integer,
    `fs_path` text
);

create index tracks_ids on `tracks` (`id`);
create index tracks_paths on `tracks` (`fs_path`);
create index albums_ids on `albums` (`id`);
create index artists_ids on `artists` (`id`);
