create table albums (
    id integer not null primary key, 
    name text,
    artist_id integer not null
);

create table artists (
    id integer not null primary key, 
    name text
);

create table tracks (
    id integer not null primary key,
    album_id integer,
    artist_id integer,
    name text,
    fs_path text
);
