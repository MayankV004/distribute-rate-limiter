// Package identity extracts a stable client identifier from an HTTP request.
//
// Extraction is tried in the order configured (api_key → jwt_sub → ip).
// The first successful extraction wins.
//
// TODO (Phase 2): implement Extractor interface:
//
//   type Extractor interface {
//       Extract(r *http.Request) (id string, ok bool)
//   }
//
//   func Chain(extractors ...Extractor) Extractor
//     Tries each in order, returns first success.
//     Falls back to client IP if none match.
package identity
