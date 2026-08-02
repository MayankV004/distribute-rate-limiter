package identity

// TODO (Phase 2): implement JWT identity extractor with signature verification.
//
// See docs/IMPLEMENTATION_PLAN.md § 13 G2 for the full security rationale.
//
//   type JWTExtractor struct {
//       keyFunc  jwt.Keyfunc
//       audience string
//   }
//
// Three constructor modes (selected by config):
//
//   func NewJWTFromJWKS(jwksURI, audience string, refresh time.Duration) (*JWTExtractor, error)
//     - Fetches JWKS from jwksURI at startup and on kid-mismatch
//     - Verifies RS256 or ES256 signatures
//
//   func NewJWTFromHMAC(secret, audience string) *JWTExtractor
//     - Static HS256 secret — suitable for internal services
//
//   (no passthrough constructor — passthrough = don't register this extractor)
//
//   func (e *JWTExtractor) Extract(r *http.Request) (string, bool)
//     - Strip "Bearer " prefix from Authorization header
//     - jwt.Parse with: ValidMethods, Audience, ExpirationRequired
//     - On any error (malformed, expired, wrong sig): return "", false
//       → falls through to next identity strategy (NOT a request block)
//     - On success: return sub claim, true
//
// IMPORTANT: a failed JWT is NOT a 401. Rate-limiting is not the auth layer.
// Failed verification means "I can't extract a JWT identity" — the chain
// tries the next extractor (IP, API key) instead.
