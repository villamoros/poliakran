// Package httpapi expone los casos de uso de Identidad como API HTTP JSON.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/villamoros/poliakran/backend/internal/identity/app"
	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
	"github.com/villamoros/poliakran/backend/internal/platform/httpx"
	"github.com/villamoros/poliakran/backend/internal/platform/ratelimit"
)

type CookieConfig struct {
	Name   string
	Secure bool
}

type Handler struct {
	svc        *app.Service
	authz      authz.Authorizer
	cookie     CookieConfig
	limiter    *ratelimit.Limiter
	trustProxy bool
	log        *slog.Logger
}

func NewHandler(svc *app.Service, az authz.Authorizer, cookie CookieConfig, limiter *ratelimit.Limiter, trustProxy bool, log *slog.Logger) *Handler {
	return &Handler{svc: svc, authz: az, cookie: cookie, limiter: limiter, trustProxy: trustProxy, log: log}
}

// Register monta las rutas del módulo en el mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/auth/register", h.rateLimited(http.HandlerFunc(h.register)))
	mux.Handle("POST /api/v1/auth/login", h.rateLimited(http.HandlerFunc(h.login)))
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.Handle("GET /api/v1/auth/me", h.RequireAuth(http.HandlerFunc(h.me)))
}

type userDTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type tenantDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type meDTO struct {
	User   userDTO      `json:"user"`
	Tenant tenantDTO    `json:"tenant"`
	Roles  []authz.Role `json:"roles"`
}

func toUserDTO(u domain.User) userDTO {
	return userDTO{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, CreatedAt: u.CreatedAt}
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	res, err := h.svc.Register(r.Context(), app.RegisterInput{
		Email: req.Email, DisplayName: req.DisplayName, Password: req.Password,
		UserAgent: r.UserAgent(),
	})
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusUnprocessableEntity, "validation_failed", "Revisa los datos del formulario.", ve.Fields)
		return
	case errors.Is(err, domain.ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, "email_taken", "Ya existe una cuenta con ese correo. Inicia sesión o usa otro correo.",
			map[string]string{"email": "Este correo ya está registrado."})
		return
	case err != nil:
		h.internalError(w, r, err)
		return
	}
	h.setSessionCookie(w, res)
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": toUserDTO(res.User)})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	res, err := h.svc.Login(r.Context(), app.LoginInput{Email: req.Email, Password: req.Password, UserAgent: r.UserAgent()})
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Correo o contraseña incorrectos.", nil)
		return
	case err != nil:
		h.internalError(w, r, err)
		return
	}
	h.setSessionCookie(w, res)
	httpx.JSON(w, http.StatusOK, map[string]any{"user": toUserDTO(res.User)})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(h.cookie.Name); err == nil {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			h.internalError(w, r, err)
			return
		}
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, _ := authz.PrincipalFrom(r.Context())
	view, _ := sessionFrom(r.Context())
	if err := h.authz.Authorize(r.Context(), p, authz.ActionIdentityReadSelf, authz.Resource{OwnerID: view.User.ID}); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden", "No tienes permiso para esta operación.", nil)
		return
	}
	httpx.JSON(w, http.StatusOK, meDTO{
		User:   toUserDTO(view.User),
		Tenant: tenantDTO{ID: view.Tenant.ID, Name: view.Tenant.Name, Kind: view.Tenant.Kind},
		Roles:  view.Roles,
	})
}

type sessionKey struct{}

func sessionFrom(ctx context.Context) (app.SessionView, bool) {
	v, ok := ctx.Value(sessionKey{}).(app.SessionView)
	return v, ok
}

// RequireAuth exige una sesión válida y deja el principal en el contexto.
// Otros módulos lo usarán para proteger sus rutas.
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(h.cookie.Name)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Inicia sesión para continuar.", nil)
			return
		}
		view, p, err := h.svc.Authenticate(r.Context(), c.Value)
		if errors.Is(err, domain.ErrSessionNotFound) {
			h.clearSessionCookie(w)
			httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Tu sesión expiró. Inicia sesión de nuevo.", nil)
			return
		}
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		ctx := authz.WithPrincipal(r.Context(), p)
		ctx = context.WithValue(ctx, sessionKey{}, view)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) rateLimited(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path + "|" + httpx.ClientIP(r, h.trustProxy)
		if !h.limiter.Allow(key) {
			w.Header().Set("Retry-After", "60")
			httpx.Error(w, http.StatusTooManyRequests, "rate_limited", "Demasiados intentos. Espera un momento y vuelve a intentarlo.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, res app.AuthResult) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    res.Token,
		Path:     "/",
		Expires:  res.ExpiresAt,
		MaxAge:   int(time.Until(res.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: h.cookie.Name, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.cookie.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.log.ErrorContext(r.Context(), "internal_error", "request_id", httpx.RequestIDFrom(r.Context()), "error", err.Error())
	httpx.Error(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor.", nil)
}
