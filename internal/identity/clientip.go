package identity

// TODO (Phase 2): implement client IP extraction.
//
//   type IPExtractor struct {
//       TrustedCIDRs []*net.IPNet  // only trust X-Forwarded-For from these
//   }
//
//   func NewIPExtractor(cidrList []string) (*IPExtractor, error)
//     Parses CIDR strings, returns error on any invalid block.
//
//   func (e *IPExtractor) Extract(r *http.Request) (string, bool)
//     - If RemoteAddr is in TrustedCIDRs, walk X-Forwarded-For right-to-left
//       and return the first non-trusted IP.
//     - Otherwise return RemoteAddr (strip port).
//     - Always returns true (IP is always extractable).
