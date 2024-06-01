-- +migrate Up
CREATE TABLE IF NOT EXISTS `radio_stations` (
    `id` integer not null primary key,
    `name` text not null,
    `stream_url` text not null,
    `home_page` text null
);

-- +migrate Down
drop table if exists `radio_stations`;
