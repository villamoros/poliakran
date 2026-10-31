# Registro de decisiones de arquitectura (ADR)

Cada decisión relevante se documenta en un archivo numerado con el formato
*contexto → decisión → consecuencias*. Un ADR aceptado no se edita: si cambia la
decisión, se crea uno nuevo que lo sustituye.

| ADR | Título | Estado |
|---|---|---|
| [0001](0001-monolito-modular-hexagonal.md) | Monolito modular en Go con arquitectura hexagonal | Propuesto |
| [0002](0002-estructura-repositorio-y-stack.md) | Monorepositorio, stack y configuración por entornos | Propuesto |
| [0003](0003-autenticacion-sesiones-propias.md) | Autenticación propia con sesiones opacas en cookie | Propuesto |
| [0004](0004-multitenencia.md) | Multitenencia con `tenant_id` y tenant personal | **Aceptado** |
| [0005](0005-autorizacion-centralizada.md) | Autorización centralizada basada en políticas | Propuesto |
| [0006](0006-compras-grupales-licencias-asientos.md) | Compras grupales como licencias por asientos | Propuesto · fuera del Sprint 1 |
| [0007](0007-frontera-ia.md) | Frontera de integración para IA y agentes | Propuesto |
| [0008](0008-catalogo-tenant-plataforma.md) | Catálogo por tenant y tenant de plataforma | Propuesto · Sprint 2 |

Estados posibles: Propuesto, Aceptado, Sustituido, Descartado. Todos los ADR del
Sprint 1 están en **Propuesto** hasta que el Product Owner los revise en la Sprint Review.
