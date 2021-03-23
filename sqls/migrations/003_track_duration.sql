-- +migrate Up
alter table tracks add column duration integer;

-- +migrate Down
alter table drop column duration;
