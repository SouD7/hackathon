# API Design

Base URL: `http://localhost:8080`

Protected endpoints require `Authorization: Bearer <token>`.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/healthz` | No | Health check |
| POST | `/api/auth/register` | No | User registration |
| POST | `/api/auth/login` | No | Login |
| GET | `/api/me` | Yes | Current user |
| GET | `/api/listings` | No | List items |
| POST | `/api/listings` | Yes | Create item |
| POST | `/api/listings/{id}/purchase` | Yes | Purchase item |
| POST | `/api/conversations` | Yes | Start DM |
| GET | `/api/conversations` | Yes | List DMs |
| GET | `/api/conversations/{id}/messages` | Yes | List messages |
| POST | `/api/conversations/{id}/messages` | Yes | Send message |
| POST | `/api/ai/description` | Yes | Generate item description |

## Example Payloads

### Register

```json
{
  "name": "Sodai",
  "email": "sodai@example.com",
  "password": "password123"
}
```

### Create Listing

```json
{
  "title": "経済学の教科書",
  "description": "前期だけ使いました。書き込み少なめです。",
  "price": 1200
}
```

### Generate Description

```json
{
  "title": "経済学の教科書",
  "condition": "中古、書き込み少なめ",
  "notes": "学内で受け渡し可能"
}
```
