alter table listings add column if not exists image_urls text not null default '[]';

update listings
set image_urls = json_build_array(image_url)::text
where image_url <> '' and (image_urls = '' or image_urls = '[]');
