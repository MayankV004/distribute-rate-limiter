// Package limiter defines the admission-control contract and its registry.
// registry.go maps (algorithm, store) pairs to their constructors.
package limiter

// TODO (Phase 1 + 3): implement the algorithm registry.
//
//   func New(algorithm Algorithm, store Store, spec Spec, ...) (Limiter, error)
//     - "local"  + algorithm → returns from internal/limiter/local
//     - "redis"  + algorithm → returns from internal/limiter/distributed
//     - Unknown algorithm or store → return descriptive error
//
// This is what config-driven route setup calls — one line to swap
// local↔redis for the Phase 8 experiment.
