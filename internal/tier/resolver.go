package tier

import (
	"context"
	"errors"
)

// ErrUnknown is returned when the resolver cannot find a tier for the given identity.
var ErrUnknown = errors.New("unknown identity")

// Resolver maps a client identity (e.g., an API key) to a tier name (e.g., "free", "pro").
type Resolver interface {
	// Resolve returns the tier name for a given identity.
	// If the identity is not recognized, it returns ("", ErrUnknown).
	// A non-nil, non-ErrUnknown error indicates an infrastructure failure (e.g., DB down).
	Resolve(ctx context.Context, identity string) (string, error)
}
