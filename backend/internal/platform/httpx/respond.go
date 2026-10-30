// Package httpx reúne utilidades HTTP compartidas por todos los módulos:
// respuestas JSON, formato de errores y middleware transversal.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20

// ErrorBody es el contrato de error de la API (ver docs/api/README.md).
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	JSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message, Fields: fields}})
}

// DecodeJSON lee un cuerpo JSON con límite de tamaño y sin campos desconocidos.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return fmt.Errorf("el cuerpo supera %d bytes", maxBodyBytes)
		case errors.Is(err, io.EOF):
			return errors.New("el cuerpo está vacío")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			return errors.New("el cuerpo contiene campos no admitidos")
		default:
			return errors.New("el cuerpo no es JSON válido")
		}
	}
	if dec.More() {
		return errors.New("el cuerpo debe contener un único objeto JSON")
	}
	return nil
}
