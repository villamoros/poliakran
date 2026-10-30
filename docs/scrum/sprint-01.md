# Sprint 1 — Fundamentos del producto

- **Fechas:** 2026-10-30 → 2026-11-13 (2 semanas)
- **Sprint Goal:** establecer la base técnica y de seguridad del LMS y completar el primer flujo vertical de identidad.
- **Incremento esperado:** una aplicación ejecutable en la que un usuario pueda autenticarse de forma funcional.

## Elementos seleccionados

| ID | Elemento | Estado |
|---|---|---|
| LMS-US-001 | Registro de estudiante | ✅ Hecho |
| LMS-US-002 | Inicio de sesión | ✅ Hecho |
| LMS-US-013 | Cierre de sesión (necesario para un flujo de identidad completo) | ✅ Hecho |
| LMS-EP11 | Base de calidad: pruebas, validación, logs estructurados, CI | ✅ Hecho |

## Plan del Sprint (tareas técnicas)

| Tarea | Estado |
|---|---|
| Estructura del monorepositorio (`backend/`, `frontend/`, `docs/`) | ✅ |
| Backend Go: monolito modular, composition root, apagado ordenado | ✅ |
| Frontend Next.js 16 + TypeScript con tokens del Manual de Identidad | ✅ |
| Configuración por entornos (`.env.example`, validación de config) | ✅ |
| PostgreSQL 16 con `docker compose` y migraciones versionadas (goose) | ✅ |
| ADR iniciales (0001–0007) | ✅ Propuestos |
| Selección del mecanismo de autenticación (ADR-0003) | ✅ Propuesto |
| API: registro, login, logout, `me`, `healthz`, `readyz` | ✅ |
| Autorización centralizada y principal por tenant | ✅ |
| Pantallas: inicio, registro, inicio de sesión, panel | ✅ |
| Pruebas unitarias, de integración (PostgreSQL) y de API | ✅ |
| Pruebas de componentes y cliente API en frontend | ✅ |
| Logs JSON con request id y redacción de datos sensibles | ✅ |
| Pipeline de CI | ✅ (se activará cuando exista remoto) |

## Verificación de criterios de aceptación

### LMS-US-001 — Registro
| Criterio | Cómo se cumple | Prueba |
|---|---|---|
| Permite enviar los datos de registro | `POST /api/v1/auth/register`, formulario `/registro` | `TestIdentityFlowEndToEnd`, `RegisterForm.test.tsx` |
| Los datos son validados | Dominio + validación espejo en cliente | `TestNewRegistration`, `validation.test.ts` |
| Sin identidades duplicadas | Índice único sobre correo normalizado; 409 | `TestRegisterRejectsDuplicateEmailCaseInsensitive` |
| La sesión se establece | Cookie `HttpOnly` tras registrarse | `TestIdentityFlowEndToEnd` |
| Errores comprensibles | Mensajes por campo en español | `TestRegisterValidationErrors` |
| Credenciales fuera de logs | Redacción + eventos solo con ids | `TestCredentialsNeverReachLogs`, `TestSensitiveValuesAreRedacted` |

### LMS-US-002 — Inicio de sesión
| Criterio | Cómo se cumple | Prueba |
|---|---|---|
| El usuario puede autenticarse | `POST /api/v1/auth/login` | `TestLogin` |
| El sistema reconoce su identidad | `GET /api/v1/auth/me` | `TestIdentityFlowEndToEnd` |
| Funciones protegidas exigen sesión | Middleware `RequireAuth`, 401 | `TestIdentityFlowEndToEnd` |
| Errores gestionados de forma segura | Respuesta idéntica para correo desconocido y contraseña errónea; rate limiting | `TestIdentityFlowEndToEnd`, `TestLimiterBlocksAfterLimitAndResets` |

## Definition of Done del Incremento

| # | Punto | Estado |
|---|---|---|
| 1 | Criterios de aceptación | ✅ |
| 2 | Lógica implementada | ✅ |
| 3 | Autorización en backend | ✅ |
| 4 | Validaciones | ✅ |
| 5–6 | Pruebas automatizadas en verde | ✅ |
| 7 | Revisión de código | ✅ Revisado por el Product Owner antes del cierre del sprint |
| 8 | Integración con el producto | ✅ Recorrido navegador → Next.js → Go → PostgreSQL verificado |
| 9 | Sin errores bloqueantes conocidos | ✅ |
| 10 | Documentación | ✅ README, guía local, API, ADR |
| 11 | Sin secretos en logs | ✅ |
| 12 | Demostrable en el entorno acordado | ✅ Local con `docker compose` |
| — | Consistencia ante repetición | ✅ Registro duplicado → 409; logout idempotente |

## Impedimentos y decisiones abiertas

- **LMS-DEC-001** resuelta: BD compartida con `tenant_id` (confirmada por el Product Owner el 2026-10-30).
- **LMS-DEC-002** resuelta: un estudiante puede pertenecer a varias organizaciones, con organización activa por sesión (LMS-US-016 en backlog).
- No hay remoto Git todavía: la CI no se ha ejecutado en GitHub.

## Sprint Review

**Resultado:** Sprint Goal cumplido. El incremento es ejecutable de punta a punta.

**Demostración realizada (navegador → Next.js → Go → PostgreSQL):**
1. Inicio en `/` y acceso a “Crear cuenta”.
2. Validación en cliente con errores por campo.
3. Registro correcto → redirección a `/panel` con nombre, correo, rol y espacio.
4. Acceso directo a `/panel` sin sesión → redirección a `/ingresar?next=/panel`.
5. Login con contraseña incorrecta → mensaje genérico, sin revelar si el correo existe.
6. Login correcto → panel, cookie `HttpOnly` y `SameSite=Lax`.
7. Registro con correo duplicado en mayúsculas → 409 con mensaje comprensible.

**Evidencia:** 24 pruebas de frontend, pruebas de backend con PostgreSQL real, build de producción de Next.js y arranque del backend con migraciones.

**Feedback para el backlog:**
- Decidido: multitenencia con BD compartida y `tenant_id` (LMS-DEC-001).
- Decidido: estudiantes en varias organizaciones con organización activa por sesión (LMS-DEC-002 → LMS-US-016).
- Pendiente: proveedor de correo para recuperar contraseña (LMS-DEC-003 / LMS-US-015).

**Historias no completadas en el sprint:** ninguna de las seleccionadas.

## Sprint Retrospective

**Qué salió bien**
- El flujo vertical completo (UI, API, base de datos) se probó de punta a punta desde el primer día.
- Las decisiones de arquitectura quedaron como ADR antes de escribir la mayor parte del código.
- La Definition of Done se verificó con evidencia concreta, no solo con “está escrito”.

**Qué mejorar**
- Next.js 16 cambia APIs respecto a versiones anteriores: consultar `node_modules/next/dist/docs/` antes de escribir código nuevo.
- `.github/` no se puede escribir desde el entorno remoto: el workflow de CI se dejó en `tools/ci/` y hay que moverlo a mano.
- La CI no se ha ejecutado en GitHub porque aún no hay remoto.

**Acciones para el Sprint 2**
1. Confirmar el remoto de GitHub y activar la CI (LMS-TD-002 y pipeline).
2. Añadir linter de dependencias entre contextos (LMS-TD-001) antes de crear el contexto de Catálogo.
3. Refinar LMS-US-003 y LMS-US-004 con criterios de administración de roles (LMS-US-014).
