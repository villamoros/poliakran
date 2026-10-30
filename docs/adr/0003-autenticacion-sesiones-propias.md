# ADR-0003 — Autenticación propia con sesiones opacas en cookie

- **Estado:** Propuesto
- **Fecha:** 2026-10-30
- **Sprint:** 1
- **Historias:** LMS-US-001, LMS-US-002

## Contexto

El plan deja abierta la elección entre un **proveedor gestionado** (Auth0, Clerk,
Cognito, Keycloak…) y una **solución propia delimitada**. Requisitos del Sprint 1:
registro, inicio y cierre de sesión, credenciales fuera de los logs, autorización
en backend y multitenencia futura.

## Opciones

| Opción | A favor | En contra |
|---|---|---|
| Proveedor gestionado | Recuperación de acceso, MFA y SSO incluidos | Coste por usuario activo, dependencia externa, sincronizar usuarios y tenants, más difícil de ejecutar en local |
| JWT sin estado | Sin consulta a BD por petición | Revocación difícil, tokens largos en el navegador |
| **Sesiones opacas en servidor** | Revocación inmediata, simple, todo en local | Una consulta por petición autenticada |

## Decisión

Solución propia **delimitada al contexto de Identidad**:

- **Contraseñas** con **argon2id** (m=19 MiB, t=2, p=1, sal de 16 bytes; parámetros OWASP), en formato PHC para poder subir parámetros sin migrar.
- **Política de contraseña:** 10–128 caracteres, no solo espacios, distinta del correo. Sin reglas de composición (NIST 800-63B).
- **Sesión:** token aleatorio de 256 bits. En la BD solo se guarda su **SHA-256**. Caducidad configurable (`SESSION_TTL`, 7 días por defecto). El cierre de sesión revoca en servidor y es idempotente.
- **Cookie:** `pa_session`, `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` obligatorio en producción. El JavaScript del navegador nunca ve el token.
- **CSRF:** `SameSite=Lax` + las operaciones que cambian estado exigen `Content-Type: application/json` + validación de `Origin` contra `ALLOWED_ORIGINS`.
- **Errores de login indistinguibles:** misma respuesta y mismo coste de verificación (hash ficticio) cuando el correo no existe.
- **Rate limiting** por IP en registro y login (`AUTH_RATE_LIMIT`, 10/min por defecto).
- **Correo duplicado:** se informa en el registro (409) porque el criterio de aceptación pide errores comprensibles. Esto permite enumerar cuentas registradas; se acepta en el MVP con el rate limiting como mitigación y se revisará cuando exista verificación de correo.
- **Puertos:** `PasswordHasher`, `Tokens` y `Store` son interfaces; cambiar a un proveedor gestionado afectaría solo a los adaptadores de Identidad.

## Consecuencias

- Todo el flujo funciona en local sin servicios externos.
- Quedan para próximos sprints: recuperación de contraseña (requiere envío de correo), verificación de correo, MFA, limpieza periódica de sesiones caducadas y bloqueo progresivo por cuenta.
