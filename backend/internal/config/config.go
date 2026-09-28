// Package config defines all configuration types for the gateway and provides
// helpers to load them from a YAML file (configs/gateway.yaml or gateway.dev.yaml).
//
// Loading flow:
//
//	YAML file  →  koanf  →  Config struct  →  Validate()  →  ready
//	                 ↑
//	          env overrides (GATEWAY_* prefix)
//
// Every field uses a `koanf:"..."` struct tag matching the YAML key exactly.
// time.Duration fields are parsed from strings like "5s", "200ms", "1m".
package config

import "time"

// ─── Top-level ───────────────────────────────────────────────────────────────

// Config is the complete, validated gateway configuration.
// Load() returns a *Config ready to use; it never returns an invalid config.
type Config struct {
	Server  ServerConfig            `koanf:"server"`
	Redis   RedisConfig             `koanf:"redis"`
	Breaker BreakerConfig           `koanf:"breaker"`
	Identity IdentityConfig         `koanf:"identity"`
	APIKeys  map[string]string      `koanf:"api_keys"` // identity → tier name  (G1)
	Tiers   map[string]TierConfig   `koanf:"tiers"`
	Routes  []RouteConfig           `koanf:"routes"`
}

// ─── Server ──────────────────────────────────────────────────────────────────

// ServerConfig controls the two HTTP listeners.
type ServerConfig struct {
	// Addr is the main proxy listener (e.g. ":8080").
	Addr string `koanf:"addr"`

	// MetricsAddr is the Prometheus /metrics listener (e.g. ":9090").
	// This port must NOT be published to the internet — expose: only in compose.
	MetricsAddr string `koanf:"metrics_addr"`

	// ReadTimeout is applied to every incoming request.
	ReadTimeout time.Duration `koanf:"read_timeout"`

	// ShutdownGrace is how long to wait for in-flight requests to drain
	// when a SIGTERM / SIGINT is received.
	ShutdownGrace time.Duration `koanf:"shutdown_grace"`

	// MetricsToken, if non-empty, requires "Authorization: Bearer <token>"
	// on every request to /metrics. Empty = rely on network-level controls (G8).
	MetricsToken string `koanf:"metrics_token"`
}

// ─── Redis ───────────────────────────────────────────────────────────────────

// RedisConfig controls the Redis client pool.
// Addrs accepts a single address (single-node) or multiple (Cluster).
// go-redis/v9 uses the same API for both; the application code does not change.
type RedisConfig struct {
	// Addrs is one or more "host:port" strings.
	// Single node:  ["redis:6379"]
	// Cluster:      ["redis1:6379", "redis2:6379", ...]
	Addrs []string `koanf:"addrs"`

	// PoolSize is the maximum number of socket connections per gateway replica.
	PoolSize int `koanf:"pool_size"`

	// DialTimeout is how long to wait when opening a new connection.
	DialTimeout time.Duration `koanf:"dial_timeout"`

	// CommandTimeout is the per-call deadline for every Redis command including
	// EVALSHA. Must be short enough to trip the circuit breaker before goroutines
	// accumulate. Recommended: 50ms.
	CommandTimeout time.Duration `koanf:"command_timeout"`

	// ClockDriftToleranceMs is the maximum acceptable difference (in ms) between
	// the Go process clock and the Redis server clock. When the Lua drift guard
	// detects a delta larger than this, the call returns ErrClockSkew and the
	// gateway applies its fail-open/closed policy. Set to 0 to disable (G3).
	ClockDriftToleranceMs int64 `koanf:"clock_drift_tolerance_ms"`
}

// ─── Circuit Breaker ─────────────────────────────────────────────────────────

// BreakerConfig controls the circuit breaker that wraps every Redis call.
//
// State machine: Closed → Open → Half-Open → Closed | Open
// See ARCHITECTURE.md § 5 for the full FSM diagram.
//
// NOTE: defaults below are starting points, not experimentally calibrated.
// Phase 5 toxiproxy experiment will produce validated values (G7).
type BreakerConfig struct {
	// ErrorRatio is the fraction of calls that must fail before the breaker
	// opens. Only evaluated after MinRequests calls in the window.
	ErrorRatio float64 `koanf:"error_ratio"`

	// MinRequests is the minimum call count in the measurement window before
	// ErrorRatio is considered. Prevents 2/3 failures from tripping the breaker.
	MinRequests int `koanf:"min_requests"`

	// OpenDuration is how long the breaker stays Open before probing.
	// Must be greater than the Redis container restart time in the compose stack.
	OpenDuration time.Duration `koanf:"open_duration"`

	// HalfOpenSuccesses is the number of consecutive successful probes required
	// to transition from Half-Open back to Closed.
	HalfOpenSuccesses int `koanf:"half_open_successes"`
}

// ─── Identity ────────────────────────────────────────────────────────────────

// IdentityConfig controls how the gateway extracts a stable client identifier
// from each request and resolves it to a rate-limit tier.
type IdentityConfig struct {
	// Order lists the extraction strategies to try in sequence.
	// First match wins. Valid values: "api_key", "jwt_sub", "ip".
	Order []string `koanf:"order"`

	// APIKeyHeader is the request header name carrying the API key.
	APIKeyHeader string `koanf:"api_key_header"`

	// TrustedProxyCIDRs lists CIDR blocks whose X-Forwarded-For header is
	// trusted for IP extraction. Requests from other sources use RemoteAddr.
	TrustedProxyCIDRs []string `koanf:"trusted_proxy_cidrs"`

	// DefaultTier is the tier name assigned when an API key is not found
	// in Config.APIKeys and UnknownKeyPolicy is "default_tier" (G1).
	DefaultTier string `koanf:"default_tier"`

	// UnknownKeyPolicy controls what happens when an API key is not in
	// Config.APIKeys: "deny" (return ErrUnknown) or "default_tier" (G1).
	UnknownKeyPolicy string `koanf:"unknown_key_policy"`

	// JWT configures JWT Bearer token identity extraction (G2).
	JWT JWTConfig `koanf:"jwt"`
}

// JWTConfig controls JWT signature verification for the jwt_sub identity strategy.
// See IMPLEMENTATION_PLAN.md § 13 G2 for the full security rationale.
type JWTConfig struct {
	// Mode selects the verification mechanism:
	//   "passthrough"  — jwt_sub extractor is disabled (fall through to next strategy)
	//   "jwks_uri"     — fetch JWKS and verify RS256/ES256 signature + exp + aud
	//   "hmac_secret"  — verify HS256 using secret from HMACSecretEnv
	Mode string `koanf:"mode"`

	// JWKSUri is the URL of the JSON Web Key Set endpoint.
	// Required when Mode == "jwks_uri".
	JWKSUri string `koanf:"jwks_uri"`

	// Audience is validated against the JWT "aud" claim.
	Audience string `koanf:"audience"`

	// JWKSRefreshInterval controls how often the JWKS cache is refreshed.
	JWKSRefreshInterval time.Duration `koanf:"jwks_refresh_interval"`

	// HMACSecretEnv is the name of the environment variable containing
	// the HS256 secret. Required when Mode == "hmac_secret".
	HMACSecretEnv string `koanf:"hmac_secret_env"`
}

// ─── Tiers ───────────────────────────────────────────────────────────────────

// TierConfig is the rate-limit quota for one named tier (e.g. "free", "pro").
// Tiers are applied after identity extraction and tier resolution.
type TierConfig struct {
	// Limit is the number of units permitted per Window.
	Limit int64 `koanf:"limit"`

	// Window is the time period over which Limit applies (e.g. "1m", "60s").
	Window time.Duration `koanf:"window"`

	// Burst is the token/leaky-bucket capacity.
	// Ignored (must be zero) for sliding_window_log and sliding_window_counter.
	// Config validation enforces this (G5).
	Burst int64 `koanf:"burst"`
}

// ─── Routes ──────────────────────────────────────────────────────────────────

// RouteConfig is the rate-limit policy for one URL pattern.
// Routes are matched in declaration order; first match wins.
type RouteConfig struct {
	// Pattern is the URL path pattern (chi-style, e.g. "/api/v1/{id}").
	// Uses the pattern, not the raw path, so /orders/1 and /orders/2 share
	// one Redis bucket.
	Pattern string `koanf:"pattern"`

	// Methods restricts matching to specific HTTP verbs.
	// Empty slice matches all methods.
	Methods []string `koanf:"methods"`

	// Algorithm selects the rate-limiting algorithm for this route.
	// Valid: "token_bucket", "sliding_window_log", "sliding_window_counter",
	//        "leaky_bucket".
	Algorithm string `koanf:"algorithm"`

	// Store selects where counter state lives.
	// "local" = in-process memory (single-node correct only).
	// "redis" = shared Redis (globally correct across all replicas).
	// Switching between local and redis for the Phase 8 experiment is a
	// config change only — no recompile.
	Store string `koanf:"store"`

	// Cost is how many units each request consumes from the quota.
	// Expensive endpoints can charge more without changing the Limiter interface.
	Cost int64 `koanf:"cost"`

	// Fallback is the policy when the backing store is unavailable.
	// "open"   = allow the request (availability wins).
	// "closed" = deny the request (protection wins).
	// Different routes can have different policies (G4 rationale).
	Fallback string `koanf:"fallback"`

	// Upstream is the URL of the backend service to proxy allowed requests to.
	Upstream string `koanf:"upstream"`

	// UpstreamRetries must be 0. Non-zero is rejected by validation.
	// Retry logic is not implemented; see IMPLEMENTATION_PLAN.md § 13 G6.
	UpstreamRetries int `koanf:"upstream_retries"`

	// Tier overrides the tier resolved from the API key for this specific route.
	// Leave empty to use the tier from identity resolution.
	Tier string `koanf:"tier"`

	// L1Cache configures the optional per-replica allow-cache for this route.
	// Disabled by default so the global limit (S3) holds exactly (G4).
	L1Cache L1CacheConfig `koanf:"l1_cache"`
}

// L1CacheConfig controls the optional short-TTL per-replica allow-cache.
// See ARCHITECTURE.md § 6 and IMPLEMENTATION_PLAN.md § 13 G4.
type L1CacheConfig struct {
	// Enabled must be false (default) for Phase 8 Experiment A to be valid.
	// Set to true only after Phase 7 measures and publishes the over-admission cost.
	Enabled bool `koanf:"enabled"`

	// TTL is how long a cached allow decision is trusted before hitting Redis again.
	TTL time.Duration `koanf:"ttl"`

	// RemainingThreshold is the minimum fraction of quota remaining before caching
	// is skipped. At Remaining <= limit * threshold, accuracy matters most.
	RemainingThreshold float64 `koanf:"remaining_threshold"`
}
