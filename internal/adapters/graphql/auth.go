package graphql

import (
	"net/http"

	"github.com/abitofhelp/family-service2/internal/infrastructure/auth"
	"github.com/abitofhelp/family-service2/internal/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// AuthMiddleware is a middleware for handling authentication and authorization
// It's a thin wrapper around the infrastructure auth middleware
type AuthMiddleware struct {
	infraMiddleware *auth.AuthMiddleware
	logger          *zap.Logger
	tracer          trace.Tracer
}

// NewAuthMiddleware creates a new auth middleware using the infrastructure auth middleware
func NewAuthMiddleware(authService ports.AuthorizationService, jwtService *auth.JWTService, logger *zap.Logger) *AuthMiddleware {
	// Create the infrastructure auth middleware
	infraMiddleware := auth.NewAuthMiddleware(jwtService, logger)

	return &AuthMiddleware{
		infraMiddleware: infraMiddleware,
		logger:          logger,
		tracer:          otel.Tracer("graphql.auth"),
	}
}

// NewAuthMiddlewareWithOIDC creates a new auth middleware with OIDC support
func NewAuthMiddlewareWithOIDC(authService ports.AuthorizationService, jwtService *auth.JWTService, oidcService *auth.OIDCService, logger *zap.Logger) *AuthMiddleware {
	// Create the infrastructure auth middleware with OIDC
	infraMiddleware := auth.NewAuthMiddlewareWithOIDC(jwtService, oidcService, logger)

	return &AuthMiddleware{
		infraMiddleware: infraMiddleware,
		logger:          logger,
		tracer:          otel.Tracer("graphql.auth"),
	}
}

// Middleware is the HTTP middleware function
// It delegates to the infrastructure auth middleware
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := m.tracer.Start(r.Context(), "GraphQLAuthMiddleware")
		defer span.End()

		// Update the request with the new context that includes our span
		r = r.WithContext(ctx)

		// Delegate to the infrastructure middleware
		m.infraMiddleware.Middleware(next).ServeHTTP(w, r)
	})
}
