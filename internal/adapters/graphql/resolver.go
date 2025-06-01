package graphql

import (
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Resolver is the resolver for GraphQL queries and mutations
type Resolver struct {
	familyService ports.FamilyService
	authService   ports.AuthorizationService
	logger        *zap.Logger
	tracer        trace.Tracer
}

// NewResolver creates a new resolver
func NewResolver(familyService ports.FamilyService, authService ports.AuthorizationService, logger *zap.Logger) *Resolver {
	return &Resolver{
		familyService: familyService,
		authService:   authService,
		logger:        logger,
		tracer:        otel.Tracer("graphql.resolver"),
	}
}
