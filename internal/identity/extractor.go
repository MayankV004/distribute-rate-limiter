package identity

import (
	"net/http"
)

// Extractor pulls a stable client identifier from an HTTP request.
// It returns (identifier, true) if found, or ("", false) if this strategy
// could not identify the client (e.g., header was missing).
type Extractor interface {
	Extract(r *http.Request) (string, bool)
}

// Chain tries a sequence of extractors in order.
// The first extractor that returns (string, true) wins.
func Chain(extractors ...Extractor) Extractor {
	return &chainExtractor{extractors: extractors}
}

type chainExtractor struct {
	extractors []Extractor
}

func (c *chainExtractor) Extract(r *http.Request) (string, bool) {
	for _, e := range c.extractors {
		if id, ok := e.Extract(r); ok {
			return id, true
		}
	}
	return "", false // Should never happen if IP extractor is last
}

// APIKeyExtractor reads an identity from a specific HTTP header.
type APIKeyExtractor struct {
	headerName string
}

func NewAPIKeyExtractor(headerName string) *APIKeyExtractor {
	return &APIKeyExtractor{headerName: headerName}
}

func (e *APIKeyExtractor) Extract(r *http.Request) (string, bool) {
	key := r.Header.Get(e.headerName)
	if key == "" {
		return "", false
	}
	return key, true
}
