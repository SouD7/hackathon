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
