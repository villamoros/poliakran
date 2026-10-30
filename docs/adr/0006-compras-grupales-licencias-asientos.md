# ADR-0006 — Compras grupales como licencias por asientos

- **Estado:** Propuesto · fuera del alcance del Sprint 1
- **Fecha:** 2026-10-30

## Contexto

Las organizaciones comprarán acceso a cursos para grupos de personas. Pagos y
suscripciones están fuera del MVP, pero el modelo de inscripciones (Sprint 4)
debe poder crecer hacia compras grupales sin rehacerse.

## Decisión

- Una compra grupal se modela como una **licencia**: `licenses (id, tenant_id, course_id, seats_total, valid_from, valid_until)`.
- Cada persona que ocupa un lugar es una **asignación de asiento**: `seat_assignments (license_id, user_id, assigned_at, released_at)`.
- Una **inscripción** puede originarse por inscripción individual o por asignación de asiento (`enrollments.source`).
- Liberar un asiento no borra el progreso del estudiante.

## Consecuencias

- El contexto de Inscripciones (Sprint 4) incluirá `source` desde su primera migración.
- La facturación de licencias queda para cuando se incorporen pagos.
