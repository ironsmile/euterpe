-- +migrate Up
alter table tracks add column duration integer;

-- +migrate Down
alter table tracks drop column duration;
