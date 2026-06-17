# Mr. Market

学生サークルのハッカソン向けに作る、学生同士のフリマ MVP です。

## Stack

- Backend: Go, PostgreSQL, Gemini API
- Frontend: React, Vite, TypeScript
- Deploy: Cloud Run, Cloud SQL, Vercel

## Features

- ユーザー登録・ログイン
- 商品出品
- 商品購入フロー
- DM
- Gemini API による商品説明生成

## Local Setup

```bash
docker compose up -d
```

Backend:

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

Frontend:

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173`.

## Design Docs

- [ER Design](docs/er.md)
- [API Design](docs/api.md)
- [Deploy Guide](docs/deploy.md)
