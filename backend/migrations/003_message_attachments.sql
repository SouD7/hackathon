alter table messages
add column if not exists attachment_url text not null default '';
