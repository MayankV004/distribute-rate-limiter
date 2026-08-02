package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"github.com/go-chi/chi/v5"
	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/identity"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/limiter/registry"
	"github.com/streamliner/rate-limiter/internal/middleware"
	"github.com/streamliner/rate-limiter/internal/proxy"
	"github.com/streamliner/rate-limiter/internal/server"
	"github.com/streamliner/rate-limiter/internal/tier"
)

// A tiny real clock for the gateway
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func main() {
	cfgFile := "configs/gateway.dev.yaml"
	if len(os.Args) > 1 {
		cfgFile = os.Args[1]
	}

	fmt.Printf("Starting Gateway using config: %s\n", cfgFile)

	// An atomic pointer holds our active chi.Router.
	// This allows us to swap the entire routing table and limiter instances
	// without dropping a single in-flight request (SIGHUP hot reload).
	var activeHandler atomic.Pointer[http.Handler]

	// buildHandler parses the config and returns a fully initialized router.
	buildHandler := func() http.Handler {
		cfg := config.MustLoad(cfgFile)

		// 1. Build Identity Extractor Chain
		var extractors []identity.Extractor
		for _, name := range cfg.Identity.Order {
			switch name {
			case "api_key":
				extractors = append(extractors, identity.NewAPIKeyExtractor(cfg.Identity.APIKeyHeader))
			case "jwt_sub":
				// If JWT is configured, add it here.
				if cfg.Identity.JWT.Mode == "hmac_secret" {
					secret := os.Getenv(cfg.Identity.JWT.HMACSecretEnv)
					extractors = append(extractors, identity.NewJWTFromHMAC(secret, cfg.Identity.JWT.Audience))
				}
			case "ip":
				ipExt, err := identity.NewIPExtractor(cfg.Identity.TrustedProxyCIDRs)
				if err != nil {
					log.Fatalf("failed to create IP extractor: %v", err)
				}
				extractors = append(extractors, ipExt)
			}
		}
		extractorChain := identity.Chain(extractors...)

		// 1.5 Setup Redis if configured
		var rdb *redis.Client
		if len(cfg.Redis.Addrs) > 0 {
			rdb = redis.NewClient(&redis.Options{
				Addr:         cfg.Redis.Addrs[0],
				PoolSize:     cfg.Redis.PoolSize,
				DialTimeout:  cfg.Redis.DialTimeout,
				ReadTimeout:  cfg.Redis.CommandTimeout,
				WriteTimeout: cfg.Redis.CommandTimeout,
			})
		}

		// 2. Build Tier Resolver
		resolver := tier.NewStaticResolver(cfg.APIKeys, cfg.Identity.DefaultTier, cfg.Identity.UnknownKeyPolicy)

		// 3. (Proxy will be built per route below)

		// 4. Build Router
		r := chi.NewRouter()

		// Add global Production API Gateway middleware
		r.Use(middleware.Recover)
		r.Use(middleware.RequestID)
		r.Use(middleware.Logger)

		// For each route in config, instantiate limiters and attach middleware
		for _, routeCfg := range cfg.Routes {
			limitersByTier := make(map[string]limiter.Limiter)

			// Pre-instantiate a limiter for each tier for this specific route.
			for tierName, tierCfg := range cfg.Tiers {
				spec := limiter.Spec{
					Limit:  tierCfg.Limit,
					Window: tierCfg.Window,
					Burst:  tierCfg.Burst,
				}
				
				// Instantiate via registry.
				l, err := registry.New(routeCfg.Pattern, routeCfg.Algorithm, routeCfg.Store, spec, rdb, realClock{}, cfg.Breaker, cfg.Redis.CommandTimeout, routeCfg.L1Cache)
				if err != nil {
					log.Fatalf("failed to create limiter for route %s tier %s: %v", routeCfg.Pattern, tierName, err)
				}
				limitersByTier[tierName] = l
			}

			// Create the route-specific middleware
			mw := middleware.RateLimitForRoute(routeCfg, limitersByTier, extractorChain, resolver)
			
			// Build proxy for this route's upstream
			upstreamProxy, err := proxy.New(routeCfg.Upstream, 30*time.Second) // default 30s timeout
			if err != nil {
				log.Fatalf("invalid upstream URL %s for route %s: %v", routeCfg.Upstream, routeCfg.Pattern, err)
			}
			
			// Register with chi router (handle multiple methods if defined)
			if len(routeCfg.Methods) > 0 {
				for _, method := range routeCfg.Methods {
					r.With(mw).Method(method, routeCfg.Pattern, upstreamProxy)
				}
			} else {
				r.With(mw).Handle(routeCfg.Pattern, upstreamProxy)
			}
		}

		return r
	}

	// Initial load
	initial := buildHandler()
	activeHandler.Store(&initial)

	// Background goroutine to listen for SIGHUP for zero-downtime reloads
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			log.Println("SIGHUP received! Reloading config and swapping router...")
			// NOTE: In production, we'd use config.Load() and handle errors gracefully
			// instead of MustLoad(), to avoid crashing the server on a bad config reload.
			newHandler := buildHandler()
			activeHandler.Store(&newHandler)
			log.Println("Config reloaded successfully.")
		}
	}()

	// The main server uses a proxy handler that reads the atomic pointer
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler := *activeHandler.Load()
		handler.ServeHTTP(w, r)
	})

	initialCfg := config.MustLoad(cfgFile)
	
	// Start the robust graceful shutdown server
	ctx := context.Background()
	if err := server.Run(ctx, initialCfg.Server, serverHandler, promhttp.Handler()); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
