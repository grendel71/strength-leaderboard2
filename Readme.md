# Strength Leaderboard

A self-hosted powerlifting leaderboard web app. Athletes register, log their lifts, and compete on ranked leaderboards.

## Features

- **Main leaderboard** — ranked by squat, bench, deadlift, OHP, or total; filterable by gender
- **Bonus lifts** — admin-defined custom lift types (supports weight, distance, and/or reps)
- **Athlete profiles** — avatar upload, body weight, bio
- **Auth** — session-based registration and login with bcrypt password hashing
- **HTMX** — dynamic table updates and profile dialogs without full page reloads
- **S3 avatar storage** — compatible with any S3-compatible object store (e.g. Cloudflare R2, MinIO)

## Tech Stack

- **Go** — standard `net/http` + [chi](https://github.com/go-chi/chi) router
- **PostgreSQL** — [sqlc](https://sqlc.dev)-generated queries, migrations run on startup
- **HTMX** — progressive enhancement for dynamic UI
- **Docker** — multi-stage build, image published to GHCR on push to `main`
- **Nix** — `flake.nix` for reproducible dev environment

## Running

### With Docker Compose (recommended)

```yaml
services:
  app:
    image: ghcr.io/blau/strength-leaderboard2:latest
    environment:
      DB_HOST: db
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: leaderboard
      S3_ENDPOINT: https://...
      S3_REGION: auto
      S3_BUCKET: avatars
      S3_ACCESS_KEY: ...
      S3_SECRET_KEY: ...
      S3_PUBLIC_URL: https://...
    ports:
      - "3000:3000"
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: leaderboard
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | HTTP listen port |
| `DB_HOST` | — | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | — | Postgres user |
| `DB_PASSWORD` | — | Postgres password |
| `DB_NAME` | — | Postgres database name |
| `DB_SSLMODE` | `disable` | Postgres SSL mode |
| `S3_ENDPOINT` | — | S3-compatible endpoint URL |
| `S3_REGION` | — | S3 region |
| `S3_BUCKET` | — | Bucket name for avatar uploads |
| `S3_ACCESS_KEY` | — | S3 access key |
| `S3_SECRET_KEY` | — | S3 secret key |
| `S3_PUBLIC_URL` | — | Public base URL for serving uploaded avatars |

Database migrations run automatically on startup.

## Development

With Nix:
```sh
nix develop
go run ./cmd/server/
```

Without Nix, you need Go 1.22+ and a running Postgres instance. Set the env vars above and run:
```sh
go run ./cmd/server/
```

To regenerate DB queries after editing `queries/*.sql`:
```sh
sqlc generate
```

## CI / Docker Image

On every push to `main`, GitHub Actions builds and pushes a Docker image to `ghcr.io/blau/strength-leaderboard2:latest`. PRs build but do not push.

## Admin

Users with `role = 'admin'` in the `users` table can create new bonus lift definitions from the profile edit page.
