package identity

import (
	"fmt"
)

// BuildKey constructs the canonical Redis key for a rate limiter bucket.
//
// Format: ratelimit:{identity}:<method>:<pattern>:<algorithm>:<tier>
// Example: ratelimit:{key_abc}:GET:/api/v1/search:swc:pro
//
// The {identity} is a Redis Cluster Hash Tag. It ensures that all keys
// for the same identity are mapped to the same Redis Cluster slot.
func BuildKey(identity, method, pattern, algorithm, tier string) string {
	// If method is empty (meaning "match all methods"), we use a wildcard character.
	if method == "" {
		method = "*"
	}
	return fmt.Sprintf("ratelimit:{%s}:%s:%s:%s:%s", identity, method, pattern, algorithm, tier)
}
