package graph

import (
	"log/slog"

	"graphql-bff/internal/cache"
	"graphql-bff/internal/client"
	"graphql-bff/internal/telemetry"
)

// Resolver is the root dependency container for all GraphQL resolvers.
// Add any downstream clients or shared resources here.
type Resolver struct {
	Clients *client.Registry
	Cache   *cache.QueryCache
	Logger  *slog.Logger
	Metrics *telemetry.Metrics
}
