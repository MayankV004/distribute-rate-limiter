package identity

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JWTExtractor reads the "sub" claim from a Bearer token, strictly after
// verifying its cryptographic signature and expiration.
type JWTExtractor struct {
	keyFunc  jwt.Keyfunc
	audience string
}

// NewJWTFromJWKS creates an extractor that fetches and caches public keys
// from a remote JSON Web Key Set (JWKS) endpoint. It supports RS256 and ES256.
func NewJWTFromJWKS(jwksURI, audience string, refresh time.Duration) (*JWTExtractor, error) {
	// keyfunc handles background refreshing and caching of the public keys.
	kf, err := keyfunc.NewDefault([]string{jwksURI})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS keyfunc: %w", err)
	}

	return &JWTExtractor{
		keyFunc:  kf.Keyfunc,
		audience: audience,
	}, nil
}

// NewJWTFromHMAC creates an extractor that uses a static symmetric secret (HS256).
func NewJWTFromHMAC(secret, audience string) *JWTExtractor {
	kf := func(token *jwt.Token) (interface{}, error) {
		// Ensure the token method is actually HMAC, not a bypass attempt.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}
	return &JWTExtractor{
		keyFunc:  kf,
		audience: audience,
	}
}

// Extract implements the Extractor interface.
// If the token is missing, malformed, expired, or fails signature verification,
// it returns ("", false) so the Chain can try the next strategy (like IP).
func (e *JWTExtractor) Extract(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and strictly verify the token
	tok, err := jwt.Parse(rawToken, e.keyFunc,
		jwt.WithValidMethods([]string{"RS256", "ES256", "HS256"}),
		jwt.WithAudience(e.audience),
		jwt.WithExpirationRequired(),
	)
	
	if err != nil || !tok.Valid {
		// We DO NOT block the request here. A failed JWT just means this
		// specific extractor failed to identify the user. Rate limiting
		// is not the authentication layer. We fall through to IP.
		return "", false
	}

	// Token is cryptographically valid. Extract the Subject claim.
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", false
	}

	return sub, true
}
