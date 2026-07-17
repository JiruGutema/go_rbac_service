# RBAC Service

Standalone Role-Based Access Control service (Go + PostgreSQL + Redis). See `docs/` for the design.

## Development

### Option 1 - everything in Docker (hot reload included)

```bash
docker compose up
```

The app runs with [Air](https://github.com/air-verse/air) inside the container; edit any `.go` file and it rebuilds automatically. Postgres is on `localhost:5432`, Redis on `localhost:6379`.

### Option 2 - Air on the host, only dependencies in Docker

```bash
docker compose up -d postgres redis
cp .env.example .env
air
```

## Build a production image

```bash
docker build -t rbac-service .
```

## Connection defaults

| Service  | URL |
|----------|-----|
| App      | http://localhost:8080 |
| Postgres | `postgres://rbac:rbac@localhost:5432/rbac?sslmode=disable` |
| Redis    | `localhost:6379` |
