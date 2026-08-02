package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/limiter/local"
	"github.com/streamliner/rate-limiter/internal/middleware"
)

func main() {
	fmt.Println("Starting Gateway on :8080...")

	// 1. Create our Rate Limiter (2 requests per 10 seconds, burst of 2)
	realClock := realClock{}

	spec := limiter.Spec{Limit: 2, Window: 10 * time.Second, Burst: 2}
	myLimiter := local.NewTokenBucket(spec, realClock)

	// 2. Set up how to identify users
	extractor := func(r *http.Request) string {
		// Try to find an API key in the headers
		key := r.Header.Get("X-API-Key")
		if key != "" {
			return key
		}
		// Fallback to IP address
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}

	// 3. Set up how to match routes to limiters
	resolver := func(r *http.Request) limiter.Limiter {
		// For now, apply our one Token Bucket to ALL routes
		return myLimiter
	}

	// 4. Create the Reverse Proxy to forward allowed traffic to the dummy backend
	backendURL, _ := url.Parse("http://localhost:9000")
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// 5. Build the Router
	r := chi.NewRouter()

	// Plug our RateLimit middleware into the router
	r.Use(middleware.RateLimit(extractor, resolver))

	// Any request that survives the middleware gets forwarded to the backend
	r.Handle("/*", proxy)

	// 6. Start the server!
	log.Fatal(http.ListenAndServe(":8080", r))
}

// A tiny real clock for the gateway
type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}
