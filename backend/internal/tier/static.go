package tier

import "context"

// StaticResolver resolves identities using an in-memory map.
// It is populated once at startup from the Config.APIKeys block.
type StaticResolver struct {
	keys          map[string]string
	defaultTier   string
	unknownPolicy string // "deny" | "default_tier"
}

// NewStaticResolver creates a resolver initialized from the given map of identity -> tier.
func NewStaticResolver(keys map[string]string, defaultTier, unknownPolicy string) *StaticResolver {
	if keys == nil {
		keys = make(map[string]string)
	}
	return &StaticResolver{
		keys:          keys,
		defaultTier:   defaultTier,
		unknownPolicy: unknownPolicy,
	}
}

// Resolve looks up the identity in the static map.
// If not found, it applies the unknownPolicy:
//   - "deny" -> returns ("", ErrUnknown)
//   - "default_tier" -> returns (defaultTier, nil)
func (r *StaticResolver) Resolve(ctx context.Context, identity string) (string, error) {
	if tier, ok := r.keys[identity]; ok {
		return tier, nil
	}

	if r.unknownPolicy == "default_tier" {
		return r.defaultTier, nil
	}

	// Safe default: if unrecognised and policy isn't default_tier, give them no quota.
	return "", ErrUnknown
}
