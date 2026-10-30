# ADR-0004 — Multitenencia con `tenant_id` y tenant personal

- **Estado:** Aceptado (puntos A y B confirmados por el Product Owner el 2026-10-30)
- **Fecha:** 2026-10-30
- **Sprint:** 1

## Contexto

La plataforma atenderá a estudiantes independientes y, más adelante, a
organizaciones (empresas, instituciones) que compran cursos para grupos. Hay que
aislar los datos de cada organización desde el primer esquema.

## Decisión

1. **Base de datos compartida con `tenant_id`** en todas las tablas de dominio.
   Las consultas filtran siempre por el tenant del principal y la política de
   autorización rechaza recursos de otro tenant (ADR-0005).
2. **Usuarios globales:** `users` no lleva `tenant_id`. La pertenencia se modela con
   `memberships (tenant_id, user_id, role)`.
3. **Tenant personal:** cada estudiante independiente recibe al registrarse un tenant
   `kind = 'personal'` y la membresía `student`. Así todo el código trata igual a
   estudiantes y organizaciones.
4. **Tenant activo por sesión:** `sessions.tenant_id` guarda en qué tenant actúa la
   sesión. Al iniciar sesión se usa el tenant personal (o la membresía más antigua).

## Decisiones del Product Owner

**A. Aislamiento físico.** ✅ Resuelta el 2026-10-30: el Product Owner eligió **base de datos compartida con `tenant_id`**.

| | Compartida + `tenant_id` | Esquema/BD por organización |
|---|---|---|
| Coste y operación | Bajo: una BD, una migración | Alto: migrar N esquemas, más conexiones |
| Aislamiento | Lógico (código + políticas; se puede reforzar con RLS de PostgreSQL) | Físico |
| Informes globales | Sencillos | Complejos |
| Encaje con el MVP | Alto | Bajo |

Recomendación: compartida + `tenant_id`, añadiendo Row-Level Security de PostgreSQL
si un cliente exige aislamiento reforzado. Migrar a esquema por organización para
un cliente concreto sigue siendo posible porque todo dato lleva `tenant_id`.

**B. ¿Puede un estudiante independiente pertenecer a varias organizaciones?**
✅ Resuelta el 2026-10-30: **sí, con una organización activa por sesión**. El esquema
ya lo permite (PK `(tenant_id, user_id)`). Pendiente de implementar (LMS-US-016): afecta a la elección de tenant al iniciar sesión, al
cambio de tenant en la interfaz y a qué progreso ve cada organización.

## Consecuencias

- Toda migración futura de dominio debe incluir `tenant_id NOT NULL` con índice.
- Si la respuesta a A cambia, el impacto en el Sprint 1 es bajo: solo hay tablas de Identidad.
