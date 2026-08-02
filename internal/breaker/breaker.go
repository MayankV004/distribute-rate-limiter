// Package breaker implements a 3-state circuit breaker around Redis calls.
//
// TODO (Phase 5): implement the circuit breaker.
//
// States:
//   Closed   → normal operation; errors increment a counter
//   Open     → short-circuit; calls return limiter.ErrUnavailable immediately
//   HalfOpen → limited probes decide whether to re-close or re-open
//
// Transition rules (from plan § 7):
//   Closed  → Open:     error ratio > ErrorRatio AND total >= MinRequests
//   Open    → HalfOpen: after OpenDuration elapses
//   HalfOpen→ Closed:  HalfOpenSuccesses consecutive successes
//   HalfOpen→ Open:    any single failure
//
//   type Breaker struct { ... }
//
//   func New(cfg BreakerConfig) *Breaker
//
//   func (b *Breaker) Do(ctx context.Context, f func(context.Context) error) error
//     - If Open: return ErrUnavailable immediately (no call to f)
//     - If Closed or HalfOpen: call f, record result, transition if needed
//
// Safe for concurrent use.
package breaker
