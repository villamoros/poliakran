# Definition of Done

Un elemento está terminado y forma parte del Incremento cuando:

1. Cumple los criterios de aceptación.
2. Su lógica está implementada.
3. La autorización está aplicada en backend.
4. Las validaciones están implementadas.
5. Existen pruebas automatizadas apropiadas.
6. Las pruebas relevantes pasan.
7. El código ha sido revisado conforme al proceso del equipo.
8. Se ha integrado correctamente con el resto del producto.
9. No existen errores bloqueantes conocidos.
10. La documentación necesaria está actualizada.
11. No se exponen secretos ni datos sensibles en logs.
12. La funcionalidad puede demostrarse en el entorno acordado.

Para una funcionalidad que modifique datos, también debe comprobarse su
consistencia y su comportamiento ante solicitudes repetidas o errores relevantes.

### Cómo se verifica en este repositorio

| Punto | Evidencia |
|---|---|
| 5–6 | `go test -p 1 ./...` con `TEST_DATABASE_URL` y `npm test` en verde; CI en `.github/workflows/ci.yml` |
| 7 | Pull request revisado antes de integrar en `main` |
| 10 | ADR, `docs/api/`, guía de desarrollo y sprint actualizados en el mismo cambio |
| 11 | Prueba `TestCredentialsNeverReachLogs` y redacción en `internal/platform/logging` |
| 12 | Entorno acordado: local con `docker compose` (ver `docs/development.md`) |

# Definition of Ready (lista de comprobación, no obligatoria)

Una historia está suficientemente preparada cuando tiene descripción clara,
usuario o necesidad identificados, criterios de aceptación, dependencias
principales conocidas, alcance entendido, riesgos relevantes identificados,
puede estimarse y puede demostrarse.
