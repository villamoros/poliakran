// Package migrations contiene las migraciones SQL versionadas del esquema.
// Se ejecutan con goose y se incrustan en el binario.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
