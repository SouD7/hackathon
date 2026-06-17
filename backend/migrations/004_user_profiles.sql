alter table users add column if not exists profile_image_url text not null default '';
alter table users add column if not exists bio text not null default '';
