# ADR-0007 — Frontera de integración para IA y agentes

- **Estado:** Propuesto
- **Fecha:** 2026-10-30

## Contexto

El mentor de IA y la orquestación multiagente están fuera del MVP, pero el plan
exige que la arquitectura los permita sin acoplar el núcleo educativo a proveedores.

## Decisión

- Ningún paquete de `internal/*/domain` ni `internal/*/app` importa SDK de modelos.
- Las capacidades futuras de IA vivirán en un contexto propio (`internal/aigateway`) con adaptadores por proveedor.
- Los agentes consumirán **las mismas operaciones autorizadas** que la API: se autentican como principals con roles y acciones explícitas y pasan por `authz.Authorizer`.
- Toda acción de un agente se registrará en un log de auditoría con el principal y la herramienta (skill) usada.

## Consecuencias

- El LMS funciona sin proveedor de IA conectado (criterio 12 del Product Goal).
- Añadir un agente no requiere cambiar reglas de negocio, solo conceder acciones.
