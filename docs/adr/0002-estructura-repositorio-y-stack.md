# ADR-0002 — Monorepositorio, stack y configuración por entornos

- **Estado:** Propuesto
- **Fecha:** 2026-10-30
- **Sprint:** 1

## Contexto

El plan Scrum permite “repositorio frontend y backend o estructura acordada”.
Con un equipo pequeño, coordinar cambios de contrato entre dos repositorios añade fricción.

## Decisión

- **Un monorepositorio** con `backend/` (Go), `frontend/` (Next.js + TypeScript) y `docs/`.
- **Backend:** Go 1.26, `net/http` de la biblioteca estándar (enrutado por método y patrón), `pgx/v5` para PostgreSQL, `goose` para migraciones, `log/slog` para logs JSON.
- **Frontend:** Next.js 16 (App Router) + TypeScript, CSS Modules con los tokens del Manual de Identidad, Vitest + Testing Library.
- **Base de datos:** PostgreSQL 16. Migraciones SQL **versionadas** en `backend/migrations/`, incrustadas en el binario y reversibles (`-- +goose Down`).
- **Mismo origen:** el navegador solo habla con Next.js; `next.config.ts` reenvía `/api/*` al backend. Así la cookie de sesión es de primer nivel y no se necesita CORS.
- **Configuración por entornos:** variables de entorno (12-factor). En local, `backend/.env` y `frontend/.env.local` (ignorados por git) a partir de los `.env.example`. `APP_ENV` admite `development`, `test` y `production`; en producción se exige `SESSION_COOKIE_SECURE=true`.
- **CI:** GitHub Actions (`.github/workflows/ci.yml`) ejecuta vet, formato, pruebas con PostgreSQL real, lint, typecheck, pruebas y build del frontend.
- **Redis:** no se incorpora hasta que exista una necesidad demostrable.

## Consecuencias

- Un cambio de contrato API viaja en el mismo commit que su consumidor.
- El limitador de intentos es en memoria (una instancia). Con varias réplicas habrá que compartirlo (Redis u otra opción).
- Next.js 16 introduce cambios respecto a versiones anteriores (p. ej. `proxy.ts` sustituye a `middleware.ts`); el equipo debe consultar la documentación incluida en `node_modules/next/dist/docs/`.
