-- +migrate Up
alter table tracks add column year integer null; -- four digit number
alter table tracks add column bitrate integer null; -- bits per second
alter table tracks add column size integer null; -- file size in bytes
alter table tracks add column created_at integer null; -- unix timestamp

-- +migrate Down
alter table tracks drop column created_at;
alter table tracks drop column size;
alter table tracks drop column bitrate;
alter table tracks drop column year;
