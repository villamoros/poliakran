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

## Operación

| Ruta | Uso |
|---|---|
| `GET /healthz` | El proceso está vivo |
| `GET /readyz` | El proceso puede atender (PostgreSQL responde) |
