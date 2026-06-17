create table if not exists users (
  id bigserial primary key,
  name text not null,
  email text not null unique,
  profile_image_url text not null default '',
  bio text not null default '',
  password_hash text not null,
  created_at timestamptz not null default now()
);

create table if not exists listings (
  id bigserial primary key,
  seller_id bigint not null references users(id) on delete cascade,
  title text not null,
  description text not null default '',
  price integer not null check (price > 0),
  image_url text not null default '',
  image_urls text not null default '[]',
  status text not null default 'available' check (status in ('available', 'sold', 'cancelled')),
  buyer_id bigint references users(id) on delete set null,
  created_at timestamptz not null default now(),
  purchased_at timestamptz
);

create index if not exists listings_status_created_at_idx on listings(status, created_at desc);
create index if not exists listings_seller_id_idx on listings(seller_id);

create table if not exists conversations (
  id bigserial primary key,
  listing_id bigint not null references listings(id) on delete cascade,
  buyer_id bigint not null references users(id) on delete cascade,
  seller_id bigint not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (listing_id, buyer_id),
  check (buyer_id <> seller_id)
);

create index if not exists conversations_buyer_id_idx on conversations(buyer_id);
create index if not exists conversations_seller_id_idx on conversations(seller_id);

create table if not exists messages (
  id bigserial primary key,
  conversation_id bigint not null references conversations(id) on delete cascade,
  sender_id bigint not null references users(id) on delete cascade,
  body text not null,
  attachment_url text not null default '',
  created_at timestamptz not null default now()
);

create index if not exists messages_conversation_id_created_at_idx on messages(conversation_id, created_at);

create table if not exists purchase_notifications (
  id bigserial primary key,
  listing_id bigint not null references listings(id) on delete cascade,
  seller_id bigint not null references users(id) on delete cascade,
  buyer_id bigint not null references users(id) on delete cascade,
  conversation_id bigint not null references conversations(id) on delete cascade,
  read_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists purchase_notifications_seller_unread_idx on purchase_notifications(seller_id, read_at, created_at);
