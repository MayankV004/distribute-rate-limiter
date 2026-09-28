package config_test

import (
	"testing"
	"time"

	"github.com/streamliner/rate-limiter/internal/config"
)

// TestLoadDevConfig verifies that the committed dev config parses and validates
// without error. This test breaks the build if gateway.dev.yaml drifts out of
// sync with the Config struct.
func TestLoadDevConfig(t *testing.T) {
	cfg, err := config.Load("../../configs/gateway.dev.yaml")
	if err != nil {
		t.Fatalf("Load(gateway.dev.yaml): %v", err)
	}

	if cfg.Server.Addr == "" {
		t.Error("server.addr is empty")
	}
	if len(cfg.Routes) == 0 {
		t.Error("no routes defined")
	}
}

// TestLoadReferenceConfig verifies the reference (prod) config.
// The reference config uses store: redis, which requires redis.addrs — check it.
func TestLoadReferenceConfig(t *testing.T) {
	cfg, err := config.Load("../../configs/gateway.yaml")
	if err != nil {
		t.Fatalf("Load(gateway.yaml): %v", err)
	}

	if len(cfg.Redis.Addrs) == 0 {
		t.Error("redis.addrs is empty in reference config")
	}
	if len(cfg.Tiers) == 0 {
		t.Error("no tiers defined")
	}
}

// TestValidateMissingAddr ensures startup fails fast on a missing server addr.
func TestValidateMissingAddr(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Server.Addr = ""
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for empty server.addr, got nil")
	}
}

// TestValidateUnknownAlgorithm ensures startup fails on a bad algorithm name.
func TestValidateUnknownAlgorithm(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Routes[0].Algorithm = "fixed_window"
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for unknown algorithm, got nil")
	}
}

// TestValidateJWTSubWithoutVerification enforces G2: jwt_sub in identity order
// requires a non-passthrough JWT mode.
func TestValidateJWTSubWithoutVerification(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Identity.Order = []string{"jwt_sub"}
	cfg.Identity.JWT.Mode = "passthrough"
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for jwt_sub without verification mode, got nil")
	}
}

// TestValidateNonZeroRetries enforces G6: upstream_retries must be 0.
func TestValidateNonZeroRetries(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Routes[0].UpstreamRetries = 1
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for upstream_retries != 0, got nil")
	}
}

// TestValidateInvalidCIDR ensures bad CIDR blocks are caught at startup.
func TestValidateInvalidCIDR(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Identity.TrustedProxyCIDRs = []string{"not-a-cidr"}
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for invalid CIDR, got nil")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// minimalValidConfig returns the smallest Config that passes Validate.
// Tests mutate individual fields to check specific rules.
func minimalValidConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Addr:          ":8080",
			ReadTimeout:   5 * time.Second,
			ShutdownGrace: 5 * time.Second,
		},
		Redis: config.RedisConfig{
			Addrs:          []string{"localhost:6379"},
			CommandTimeout: 50 * time.Millisecond,
		},
		Breaker: config.BreakerConfig{
			ErrorRatio:        0.5,
			MinRequests:       10,
			OpenDuration:      5 * time.Second,
			HalfOpenSuccesses: 2,
		},
		Identity: config.IdentityConfig{
			Order:            []string{"ip"},
			APIKeyHeader:     "X-API-Key",
			UnknownKeyPolicy: "deny",
			DefaultTier:      "free",
			JWT:              config.JWTConfig{Mode: "passthrough"},
		},
		Tiers: map[string]config.TierConfig{
			"free": {Limit: 100, Window: time.Minute},
		},
		Routes: []config.RouteConfig{
			{
				Pattern:   "/api/v1/*",
				Algorithm: "token_bucket",
				Store:     "local",
				Cost:      1,
				Fallback:  "open",
				Upstream:  "http://localhost:9000",
			},
		},
	}
}
