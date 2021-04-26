-- +migrate Up
alter table `artists_images` add column `image_small` blob default null;
alter table `albums_artworks` add column `artwork_cover_small` blob default null;

-- +migrate Down
alter table `artists_images` drop column `image_small`;
alter table `albums_artworks` drop column `artwork_cover_small`;
