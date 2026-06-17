# ER Design

```mermaid
erDiagram
  USERS ||--o{ LISTINGS : sells
  USERS ||--o{ LISTINGS : buys
  LISTINGS ||--o{ CONVERSATIONS : has
  USERS ||--o{ CONVERSATIONS : buyer
  USERS ||--o{ CONVERSATIONS : seller
  CONVERSATIONS ||--o{ MESSAGES : has
  USERS ||--o{ MESSAGES : sends

  USERS {
    bigint id PK
    text name
    text email UK
    text password_hash
    timestamptz created_at
  }

  LISTINGS {
    bigint id PK
    bigint seller_id FK
    text title
    text description
    integer price
    text status
    bigint buyer_id FK
    timestamptz created_at
    timestamptz purchased_at
  }

  CONVERSATIONS {
    bigint id PK
    bigint listing_id FK
    bigint buyer_id FK
    bigint seller_id FK
    timestamptz created_at
    timestamptz updated_at
  }

  MESSAGES {
    bigint id PK
    bigint conversation_id FK
    bigint sender_id FK
    text body
    timestamptz created_at
  }
```
