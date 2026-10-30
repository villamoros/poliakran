# ADR-0001 — Monolito modular en Go con arquitectura hexagonal

- **Estado:** Propuesto
- **Fecha:** 2026-10-30
- **Sprint:** 1

## Contexto

El MVP debe validar el núcleo del LMS con un equipo pequeño, pero el producto
crecerá hacia rutas de aprendizaje, competencias, organizaciones y un sistema
multiagente. Necesitamos límites de dominio claros desde el principio sin pagar
el coste operativo de microservicios.

## Decisión

1. El backend es **un solo binario Go** (`backend/cmd/api`) con un único despliegue y una única base de datos.
2. El código se organiza por **contextos delimitados**, no por capas técnicas:

   | Contexto | Paquete | Sprint |
   |---|---|---|
   | Identidad y Organizaciones | `internal/identity` | 1 |
   | Catálogo | `internal/catalog` | 2 |
   | Enseñanza (módulos, lecciones, recursos) | `internal/teaching` | 3 |
   | Inscripciones | `internal/enrollment` | 4 |
   | Progreso | `internal/progress` | 4 |
   | Evaluación | `internal/assessment` | 5 |

3. Cada contexto sigue **arquitectura hexagonal**:
   - `domain/`: entidades, reglas y errores. Sin dependencias de infraestructura.
   - `app/`: casos de uso y **puertos** (interfaces) que necesita.
   - `adapters/`: implementaciones de los puertos (PostgreSQL, HTTP, criptografía…).
4. `internal/platform/` contiene solo piezas transversales sin reglas de negocio:
   configuración, logs, base de datos, middleware HTTP, autorización, rate limiting.
5. Un contexto **no importa** el `domain` ni los `adapters` de otro. Si necesita algo,
   lo pide a través de un puerto propio que el composition root (`cmd/api`) conecta.
6. Las tablas de cada contexto se crean en migraciones propias; un contexto no
   escribe en tablas de otro.

## Consecuencias

- Pruebas rápidas de casos de uso con dobles en memoria (ver `internal/identity/app/service_test.go`).
- Extraer un contexto a un servicio en el futuro es posible porque sus dependencias ya son explícitas.
- Hay algo más de código de “fontanería” (puertos, adaptadores) que en un CRUD directo.
- La regla 5 todavía no se verifica automáticamente; se propone añadir un linter de dependencias (backlog `LMS-TD-001`).
