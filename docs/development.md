# Guía de desarrollo local

## Requisitos

- **Go 1.26** (con Go ≥ 1.21 instalado, la herramienta descarga sola la versión indicada en `go.mod`).
- **Node.js 22** (mínimo 20.9) y npm.
- **Docker Desktop** para PostgreSQL. Alternativa: un PostgreSQL 16 local con el usuario `poliakran` y las bases `poliakran` y `poliakran_test`.

## Primer arranque

Los comandos funcionan en PowerShell y en bash.

```bash
# 1. Base de datos (crea también poliakran_test)
docker compose up -d db

# 2. Backend: http://localhost:8080
cd backend
cp .env.example .env          # PowerShell: Copy-Item .env.example .env
go run ./cmd/api              # aplica migraciones al arrancar (MIGRATE_ON_START=true)

# 3. Frontend: http://localhost:3000 (otra terminal)
cd frontend
cp .env.example .env.local    # PowerShell: Copy-Item .env.example .env.local
npm install
npm run dev
```

Abre http://localhost:3000, crea una cuenta y llegarás a `/panel`.

## Migraciones

```bash
cd backend
go run ./cmd/api migrate status
go run ./cmd/api migrate up
go run ./cmd/api migrate down        # revierte la última
go run ./cmd/api migrate create nombre sql   # nueva migración en el directorio actual
```

Las migraciones viven en `backend/migrations/`, se nombran `NNNNN_descripcion.sql`
y deben incluir su sección `-- +goose Down`.

## Pruebas

```bash
# Backend: las pruebas de integración se omiten si no hay TEST_DATABASE_URL
cd backend
TEST_DATABASE_URL=postgres://poliakran:poliakran@localhost:5432/poliakran_test?sslmode=disable go test -p 1 ./...
# PowerShell:
#   $env:TEST_DATABASE_URL="postgres://poliakran:poliakran@localhost:5432/poliakran_test?sslmode=disable"; go test -p 1 ./...

# Frontend
cd frontend
npm run lint
npm run typecheck
npm test
```

`-p 1` evita que dos paquetes migren la base de pruebas a la vez.

## Logs

El backend escribe una línea JSON por evento en la salida estándar. Cada
petición HTTP genera un `http_request` con `request_id`, `method`, `path`,
`status` y `duration_ms`. Los eventos de identidad (`user_registered`,
`login_succeeded`, `login_failed`) solo incluyen identificadores, nunca correos,
contraseñas ni tokens. Claves como `password`, `token` o `cookie` se redactan
automáticamente.

## Estructura

```
backend/
  cmd/api/                 composition root, servidor HTTP y comando migrate
  migrations/              SQL versionado (goose)
  internal/platform/       config, logging, database, httpx, authz, ratelimit
  internal/identity/       contexto de Identidad y Organizaciones
    domain/                reglas y modelo
    app/                   casos de uso y puertos
    adapters/              postgres, httpapi, security
frontend/
  src/app/                 rutas (App Router)
  src/components/          formularios y panel
  src/lib/                 cliente API y validación
  src/proxy.ts             redirección optimista de rutas protegidas
docs/
  adr/  api/  scrum/
```
