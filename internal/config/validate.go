package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

// validAlgorithms lists the algorithm identifiers accepted in RouteConfig.
var validAlgorithms = []string{
	"token_bucket",
	"sliding_window_log",
	"sliding_window_counter",
	"leaky_bucket",
}

// validStores lists the store identifiers accepted in RouteConfig.
var validStores = []string{"local", "redis"}

// validFallbacks lists the fallback policies accepted in RouteConfig.
var validFallbacks = []string{"open", "closed"}

// validUnknownKeyPolicies lists the values accepted for identity.unknown_key_policy.
var validUnknownKeyPolicies = []string{"deny", "default_tier"}

// validJWTModes lists the JWT mode values accepted in JWTConfig.
var validJWTModes = []string{"passthrough", "jwks_uri", "hmac_secret"}

// Validate checks cfg for configuration errors and returns the first one found.
// The gateway refuses to start if Validate returns a non-nil error.
// See IMPLEMENTATION_PLAN.md § 13 G5 for the Burst validation rationale.
func Validate(cfg *Config) error {
	if err := validateServer(&cfg.Server); err != nil {
		return err
	}
	if err := validateRedis(&cfg.Redis); err != nil {
		return err
	}
	if err := validateBreaker(&cfg.Breaker); err != nil {
		return err
	}
	if err := validateIdentity(&cfg.Identity); err != nil {
		return err
	}
	if err := validateTiers(cfg.Tiers); err != nil {
		return err
	}
	if err := validateRoutes(cfg.Routes, &cfg.Redis); err != nil {
		return err
	}
	return nil
}

// ─── Server ──────────────────────────────────────────────────────────────────

func validateServer(s *ServerConfig) error {
	if strings.TrimSpace(s.Addr) == "" {
		return errors.New("server.addr must not be empty")
	}
	if s.ReadTimeout <= 0 {
		return errors.New("server.read_timeout must be > 0")
	}
	if s.ShutdownGrace <= 0 {
		return errors.New("server.shutdown_grace must be > 0")
	}
	return nil
}

// ─── Redis ───────────────────────────────────────────────────────────────────

func validateRedis(r *RedisConfig) error {
	// Addrs is validated per-route (only required when store == "redis").
	if r.PoolSize < 0 {
		return errors.New("redis.pool_size must be >= 0")
	}
	if r.DialTimeout < 0 {
		return errors.New("redis.dial_timeout must be >= 0")
	}
	if r.CommandTimeout < 0 {
		return errors.New("redis.command_timeout must be >= 0")
	}
	if r.ClockDriftToleranceMs < 0 {
		return errors.New("redis.clock_drift_tolerance_ms must be >= 0 (0 disables the guard)")
	}
	return nil
}

// ─── Breaker ─────────────────────────────────────────────────────────────────

func validateBreaker(b *BreakerConfig) error {
	if b.ErrorRatio <= 0 || b.ErrorRatio > 1 {
		return fmt.Errorf("breaker.error_ratio must be in (0, 1], got %v", b.ErrorRatio)
	}
	if b.MinRequests <= 0 {
		return errors.New("breaker.min_requests must be > 0")
	}
	if b.OpenDuration <= 0 {
		return errors.New("breaker.open_duration must be > 0")
	}
	if b.HalfOpenSuccesses <= 0 {
		return errors.New("breaker.half_open_successes must be > 0")
	}
	return nil
}

// ─── Identity ────────────────────────────────────────────────────────────────

func validateIdentity(id *IdentityConfig) error {
	validStrategies := []string{"api_key", "jwt_sub", "ip"}
	for _, s := range id.Order {
		if !slices.Contains(validStrategies, s) {
			return fmt.Errorf("identity.order: unknown strategy %q (valid: %v)", s, validStrategies)
		}
	}

	if id.APIKeyHeader == "" {
		return errors.New("identity.api_key_header must not be empty")
	}

	for _, cidr := range id.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("identity.trusted_proxy_cidrs: invalid CIDR %q: %w", cidr, err)
		}
	}

	if id.UnknownKeyPolicy != "" && !slices.Contains(validUnknownKeyPolicies, id.UnknownKeyPolicy) {
		return fmt.Errorf("identity.unknown_key_policy: invalid value %q (valid: %v)",
			id.UnknownKeyPolicy, validUnknownKeyPolicies)
	}

	if err := validateJWT(&id.JWT, slices.Contains(id.Order, "jwt_sub")); err != nil {
		return err
	}

	return nil
}

func validateJWT(j *JWTConfig, jwtSubInOrder bool) error {
	if !slices.Contains(validJWTModes, j.Mode) && j.Mode != "" {
		return fmt.Errorf("identity.jwt.mode: invalid value %q (valid: %v)", j.Mode, validJWTModes)
	}

	// If jwt_sub is in identity.order but mode is passthrough or empty, that is
	// a misconfiguration — jwt_sub would be silently ignored.
	if jwtSubInOrder && (j.Mode == "" || j.Mode == "passthrough") {
		return errors.New(
			"identity.jwt.mode must be \"jwks_uri\" or \"hmac_secret\" when " +
				"\"jwt_sub\" is listed in identity.order — " +
				"remove jwt_sub from order or set a verification mode (G2)",
		)
	}

	if j.Mode == "jwks_uri" {
		if j.JWKSUri == "" {
			return errors.New("identity.jwt.jwks_uri must not be empty when mode is \"jwks_uri\"")
		}
		if _, err := url.ParseRequestURI(j.JWKSUri); err != nil {
			return fmt.Errorf("identity.jwt.jwks_uri: invalid URL %q: %w", j.JWKSUri, err)
		}
	}

	if j.Mode == "hmac_secret" && j.HMACSecretEnv == "" {
		return errors.New("identity.jwt.hmac_secret_env must not be empty when mode is \"hmac_secret\"")
	}

	return nil
}

// ─── Tiers ───────────────────────────────────────────────────────────────────

func validateTiers(tiers map[string]TierConfig) error {
	for name, t := range tiers {
		if t.Limit <= 0 {
			return fmt.Errorf("tiers.%s.limit must be > 0", name)
		}
		if t.Window <= 0 {
			return fmt.Errorf("tiers.%s.window must be > 0", name)
		}
		if t.Burst < 0 {
			return fmt.Errorf("tiers.%s.burst must be >= 0", name)
		}
	}
	return nil
}

// ─── Routes ──────────────────────────────────────────────────────────────────

func validateRoutes(routes []RouteConfig, redis *RedisConfig) error {
	if len(routes) == 0 {
		return errors.New("routes: at least one route must be defined")
	}

	for i, r := range routes {
		prefix := fmt.Sprintf("routes[%d] (%q)", i, r.Pattern)

		if strings.TrimSpace(r.Pattern) == "" {
			return fmt.Errorf("%s: pattern must not be empty", prefix)
		}

		if !slices.Contains(validAlgorithms, r.Algorithm) {
			return fmt.Errorf("%s: unknown algorithm %q (valid: %v)", prefix, r.Algorithm, validAlgorithms)
		}

		if !slices.Contains(validStores, r.Store) {
			return fmt.Errorf("%s: unknown store %q (valid: %v)", prefix, r.Store, validStores)
		}

		if !slices.Contains(validFallbacks, r.Fallback) {
			return fmt.Errorf("%s: unknown fallback %q (valid: %v)", prefix, r.Fallback, validFallbacks)
		}

		if r.Cost <= 0 {
			return fmt.Errorf("%s: cost must be >= 1, got %d", prefix, r.Cost)
		}

		if _, err := url.ParseRequestURI(r.Upstream); err != nil {
			return fmt.Errorf("%s: invalid upstream URL %q: %w", prefix, r.Upstream, err)
		}

		// G6: retries are not implemented; non-zero is rejected.
		if r.UpstreamRetries != 0 {
			return fmt.Errorf("%s: upstream_retries must be 0 (retries not implemented)", prefix)
		}

		// G5: Burst is only meaningful for token_bucket and leaky_bucket.
		if r.L1Cache.RemainingThreshold < 0 || r.L1Cache.RemainingThreshold > 1 {
			return fmt.Errorf("%s: l1_cache.remaining_threshold must be in [0, 1]", prefix)
		}

		// If store is redis, Redis.Addrs must be configured.
		if r.Store == "redis" && len(redis.Addrs) == 0 {
			return fmt.Errorf("%s: store is \"redis\" but redis.addrs is empty", prefix)
		}
	}
	return nil
}
