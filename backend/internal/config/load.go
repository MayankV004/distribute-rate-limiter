package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load reads the YAML file at path, applies GATEWAY_* environment variable
// overrides, and returns a fully validated Config.
//
// Environment variable mapping:
//
//	GATEWAY_SERVER_ADDR        → server.addr
//	GATEWAY_REDIS_ADDRS        → redis.addrs
//	GATEWAY_REDIS_POOL_SIZE    → redis.pool_size
//	... (any key, dot-separated, uppercased with GATEWAY_ prefix)
//
// Example:
//
//	cfg, err := Load("configs/gateway.dev.yaml")
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// 1. Load YAML file.
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	// 2. Apply environment variable overrides.
	//    Env vars use GATEWAY_ prefix and double underscore for nesting:
	//      GATEWAY_REDIS__POOL_SIZE → redis.pool_size
	//    Single underscore within a key name is preserved; double underscore
	//    maps to a koanf path separator (dot).
	err := k.Load(env.Provider("GATEWAY_", ".", func(s string) string {
		// Strip GATEWAY_ prefix, lowercase, replace __ with .
		s = strings.TrimPrefix(s, "GATEWAY_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "__", ".")
		return s
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("config: env override: %w", err)
	}

	// 3. Unmarshal into struct.
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	// 4. Validate before returning. A gateway with a bad config is worse
	//    than one that refuses to start.
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: invalid: %w", err)
	}

	return &cfg, nil
}

// MustLoad calls Load and panics on any error. Use only in main() before
// the logger is set up; prefer Load() everywhere else.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
