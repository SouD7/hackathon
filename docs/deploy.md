# Deploy Guide

## Backend: Cloud Run + Cloud SQL

1. Create Cloud SQL for PostgreSQL.
2. Apply `backend/migrations/001_init.sql` to the database.
3. Build and deploy the backend container.

```bash
gcloud builds submit backend --tag asia-northeast1-docker.pkg.dev/PROJECT_ID/campus-market/api
gcloud run deploy campus-market-api \
  --image asia-northeast1-docker.pkg.dev/PROJECT_ID/campus-market/api \
  --region asia-northeast1 \
  --allow-unauthenticated \
  --add-cloudsql-instances PROJECT_ID:asia-northeast1:INSTANCE_NAME \
  --set-env-vars 'DATABASE_URL=postgres://USER:PASSWORD@/DB_NAME?host=/cloudsql/PROJECT_ID:asia-northeast1:INSTANCE_NAME,JWT_SECRET=CHANGE_ME,CORS_ORIGIN=https://YOUR_VERCEL_APP.vercel.app' \
  --set-secrets 'GEMINI_API_KEY=gemini-api-key:latest'
```

For local development, use TCP:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/campus_market?sslmode=disable
```

## Frontend: Vercel

Project root: `frontend`

Environment variable:

```text
VITE_API_BASE_URL=https://YOUR_CLOUD_RUN_URL
```

Build command:

```bash
npm run build
```

Output directory:

```text
dist
```
