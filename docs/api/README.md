# API v1

Base: `/api/v1`. JSON en UTF-8. En local el navegador accede a través de Next.js
(`http://localhost:3000/api/...`), que reenvía al backend (`http://localhost:8080`).

## Convenciones

- Las peticiones que cambian estado exigen `Content-Type: application/json` y, si
  llevan cabecera `Origin`, que esté en `ALLOWED_ORIGINS`.
- Los cuerpos con campos desconocidos se rechazan (400).
- Toda respuesta incluye `X-Request-ID` para correlacionar con los logs.
- La sesión viaja en la cookie `pa_session` (`HttpOnly`, `SameSite=Lax`).

### Formato de error

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Revisa los datos del formulario.",
    "fields": { "email": "Escribe un correo electrónico válido." }
  }
}
```

| Código | HTTP | Significado |
|---|---|---|
| `invalid_body` | 400 | JSON inválido, vacío o con campos no admitidos |
| `unauthenticated` | 401 | Falta sesión o expiró |
| `invalid_credentials` | 401 | Correo o contraseña incorrectos |
| `forbidden` | 403 | La política de autorización deniega la operación |
| `origin_not_allowed` | 403 | Origen no permitido |
| `email_taken` | 409 | El correo ya está registrado |
| `course_not_found` | 404 | El curso no existe, no está publicado o es de otro tenant |
| `version_conflict` | 409 | El curso cambió desde que se leyó; hay que recargar |
| `invalid_transition` | 409 | Cambio de estado no permitido (p. ej. retirar un borrador) |
| `publish_requirements_unmet` | 422 | Faltan condiciones de publicación; detalle en `fields` |
| `invalid_query` | 400 | Parámetros de consulta inválidos |
| `unsupported_media_type` | 415 | Falta `Content-Type: application/json` |
| `validation_failed` | 422 | Errores por campo en `fields` |
| `rate_limited` | 429 | Demasiados intentos; ver `Retry-After` |
| `internal_error` | 500 | Error interno (detalle solo en logs) |

## Identidad

### `POST /auth/register`
```json
{ "email": "ana@example.com", "displayName": "Ana Pérez", "password": "minimo-10-caracteres" }
```
`201` con `{ "user": User }` y cookie de sesión. Errores: 400, 409, 415, 422, 429.

### `POST /auth/login`
```json
{ "email": "ana@example.com", "password": "..." }
```
`200` con `{ "user": User }` y cookie de sesión. Errores: 400, 401, 415, 429.

### `POST /auth/logout`
Revoca la sesión actual y borra la cookie. `204`. Idempotente.

### `GET /auth/me`
Requiere sesión. `200`:
```json
{
  "user": { "id": "uuid", "email": "ana@example.com", "displayName": "Ana Pérez", "createdAt": "2026-10-30T22:57:40Z" },
  "tenant": { "id": "uuid", "name": "Ana Pérez", "kind": "personal" },
  "roles": ["student"]
}
```

## Catálogo público (LMS-US-003)

Sin sesión. Solo devuelve cursos **publicados** del tenant de plataforma (ADR-0008).

### `GET /catalog/courses?category=&limit=24&offset=0`
`limit` entre 1 y 100. `category` opcional (ver valores abajo). `200`:
```json
{
  "items": [
    {
      "slug": "introduccion-a-python",
      "title": "Introducción a la programación con Python",
      "summary": "Da tus primeros pasos…",
      "category": "programacion",
      "level": "basico",
      "durationHours": 8,
      "publishedAt": "2026-10-30T23:31:59Z"
    }
  ],
  "total": 1, "limit": 24, "offset": 0
}
```

### `GET /catalog/courses/{slug}`
`200` con `{ "course": PublicCourse }`, que añade `description`, `objectives` y
`prerequisites`. `404` si no existe, es borrador o está retirado.

## Administración de cursos (LMS-US-004, LMS-US-006)

Requieren sesión (`401`) y rol `admin` u `owner` en el tenant activo (`403`).
Actúan siempre sobre el tenant activo: un curso de otro tenant responde `404`.

| Método y ruta | Acción | Respuesta |
|---|---|---|
| `GET /admin/courses` | Lista todos los cursos del tenant, en cualquier estado | `200 { items: AdminCourse[] }` |
| `POST /admin/courses` | Crea un curso **en borrador** | `201 { course }` + `Location` |
| `GET /admin/courses/{id}` | Detalle | `200 { course }` |
| `PUT /admin/courses/{id}` | Sustituye la información descriptiva | `200 { course }`, `409 version_conflict` |
| `POST /admin/courses/{id}/publish` | Publica (idempotente) | `200 { course }`, `422 publish_requirements_unmet` |
| `POST /admin/courses/{id}/withdraw` | Retira del catálogo (idempotente) | `200 { course }`, `409 invalid_transition` |

Cuerpo de `POST` (en `PUT` se añade `"version"`, la versión leída):
```json
{
  "title": "Introducción a la programación con Python",
  "summary": "Da tus primeros pasos en programación…",
  "description": "En este curso aprenderás…",
  "category": "programacion",
  "level": "basico",
  "durationHours": 8,
  "objectives": ["Escribir programas sencillos en Python"],
  "prerequisites": "Ninguno."
}
```

**Validación al guardar:** `title` (5–120) y `summary` (20–280) obligatorios;
`description` ≤ 10 000; `category` y `level` vacíos o de la lista;
`durationHours` 0–500; hasta 10 `objectives` de 3–200 caracteres;
`prerequisites` ≤ 1000.

**Condiciones para publicar:** `description` ≥ 50 caracteres, `category`,
`level`, `durationHours` > 0 y al menos un objetivo. Un curso publicado debe
seguir cumpliéndolas al editarlo. En el Sprint 3 se exigirá además contenido
(módulos y lecciones).

**Valores:** `category` ∈ `programacion`, `bases-de-datos`, `desarrollo-web`,
`herramientas-software`, `otras`. `level` ∈ `basico`, `intermedio`, `avanzado`.
`status` ∈ `draft`, `published`, `withdrawn`.

El `slug` se genera del título al crear (con sufijo `-2`, `-3`… si se repite en
el tenant) y no cambia al editar, para que las URL publicadas sean estables.

## Operación

| Ruta | Uso |
|---|---|
| `GET /healthz` | El proceso está vivo |
| `GET /readyz` | El proceso puede atender (PostgreSQL responde) |
