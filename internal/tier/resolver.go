// Package tier resolves an identity string to a named rate-limit tier.
//
// Identity extraction (internal/identity) tells us WHO is calling.
// Tier resolution tells us WHAT quota they get.
// These are deliberately separate concerns.
//
// See docs/IMPLEMENTATION_PLAN.md § 13 G1 for full rationale.
//
// TODO (Phase 4): implement the Resolver interface and both implementations.
//
//   var ErrUnknown = errors.New("tier: identity not found in any resolver")
//
//   type Resolver interface {
//       Resolve(ctx context.Context, identity string) (tier string, err error)
//   }
//
// Pipeline position:
//   identity extraction → Resolver.Resolve → key building → limiter.Allow
//   The tier name is embedded in the Redis key so free and pro keys
//   never share a quota bucket.
package tier
