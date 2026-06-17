alter table listings
add column if not exists image_url text not null default '';
