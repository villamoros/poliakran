# Politécnico Akran — Plataforma LMS

Plataforma de gestión del aprendizaje (LMS, *Learning Management System*) orientada inicialmente a la publicación, administración y consumo de **cursos cortos de competencias digitales y tecnológicas**.

Los estudiantes podrán descubrir cursos, inscribirse, acceder a contenidos organizados en módulos y lecciones, completar actividades, presentar evaluaciones y consultar su progreso. Los administradores podrán gestionar el catálogo educativo, estructurar los contenidos y supervisar la operación básica de la plataforma.

La prioridad del MVP es **validar el núcleo del LMS** antes de ampliar el producto con automatización e inteligencia artificial.

## Objetivo del MVP

Construir una primera versión funcional, segura y mantenible de un LMS que permita publicar y consumir cursos cortos, administrar contenidos educativos y registrar el progreso de los estudiantes.

### Alcance

1. Registro, autenticación y gestión de sesiones.
2. Catálogo navegable de cursos.
3. Creación y administración de cursos, módulos y lecciones.
4. Inscripción de estudiantes.
5. Registro del progreso de aprendizaje.
6. Evaluaciones básicas y almacenamiento de resultados.
7. Panel del estudiante.
8. Panel administrativo para gestionar contenidos.
9. API backend con reglas de negocio y autorización centralizadas.
10. Arquitectura modular con límites de dominio explícitos.
11. Puntos de integración para futuros proveedores de IA y servicios de orquestación.
12. Observabilidad, seguridad y pruebas automatizadas básicas.

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Frontend | Next.js + TypeScript |
| Backend | Go (monolito modular) |
| Base de datos | PostgreSQL |
| Caché / colas | Redis, solo cuando haya una necesidad concreta |
| Almacenamiento de archivos | Compatible con S3 |
| Autenticación | Solución propia delimitada: argon2id + sesiones opacas en cookie ([ADR-0003](docs/adr/0003-autenticacion-sesiones-propias.md)) |

El frontend no decide por sí solo si una operación está autorizada: las reglas de negocio y la autorización viven en el backend.

## Arquitectura

Monolito modular en Go con arquitectura hexagonal y contextos delimitados
([ADR-0001](docs/adr/0001-monolito-modular-hexagonal.md)):

| Contexto | Responsabilidad | Sprint |
|---|---|---|
| Identidad y Organizaciones | Usuarios, tenants, membresías, sesiones | 1 |
| Catálogo | Cursos, categorías y publicación | 2 |
| Enseñanza | Módulos, lecciones y recursos | 3 |
| Inscripciones | Inscripciones y licencias por asientos | 4 |
| Progreso | Seguimiento del aprendizaje | 4 |
| Evaluación | Evaluaciones, intentos y resultados | 5 |

Piezas transversales: autorización centralizada por políticas
([ADR-0005](docs/adr/0005-autorizacion-centralizada.md)) y multitenencia con
`tenant_id` ([ADR-0004](docs/adr/0004-multitenencia.md), pendiente de confirmación).

## Preparación para IA

El mentor de IA **no forma parte del MVP**. Desde el diseño inicial se mantiene una frontera de integración para incorporar en el futuro un sistema de orquestación de múltiples agentes, cada uno con responsabilidades definidas y acceso controlado a herramientas y *skills*. El núcleo educativo no importará SDK de proveedores de IA ni dependerá de sus estructuras.

## Fases

1. **LMS funcional**: el alcance del MVP descrito arriba.
2. **Validación y estabilización**: pruebas de aceptación, revisión de seguridad y piloto con usuarios reales.
3. **Evolución educativa**: rutas de aprendizaje, prerrequisitos, proyectos integradores, competencias.
4. **Mentoría de IA**: AI Gateway, adaptadores de proveedores y mentor de IA.
5. **Sistema multiagente**.

## Estado

**Sprint 1 — Fundamentos del producto.** Incremento ejecutable con registro,
inicio y cierre de sesión de punta a punta (Next.js → Go → PostgreSQL).
Ver [docs/scrum/sprint-01.md](docs/scrum/sprint-01.md).

## Puesta en marcha rápida

```bash
docker compose up -d db
cd backend && cp .env.example .env && go run ./cmd/api      # :8080
cd frontend && cp .env.example .env.local && npm install && npm run dev   # :3000
```

Detalles, pruebas y migraciones en [docs/development.md](docs/development.md).

## Documentación

| Documento | Contenido |
|---|---|
| [docs/scrum/](docs/scrum/README.md) | Product Goal, Product Backlog, Definition of Done y sprints |
| [docs/adr/](docs/adr/README.md) | Decisiones de arquitectura |
| [docs/api/](docs/api/README.md) | Contrato de la API |
| [docs/development.md](docs/development.md) | Guía de desarrollo local |
