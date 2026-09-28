//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func writeBreakerConfig(path, gatewayAddr, redisAddr, backendURL string) error {
	cfg := fmt.Sprintf(`
server:
  addr: "%s"
  read_timeout: 5s
  shutdown_grace: 5s

redis:
  addrs: ["%s"]
  pool_size: 10
  dial_timeout: 50ms
  command_timeout: 50ms

breaker:
  error_ratio: 0.5
  min_requests: 1
  open_duration: 1s
  half_open_successes: 1

identity:
  order: ["api_key"]
  api_key_header: "X-API-Key"
  default_tier: "free"

api_keys:
  "key-fail-open": "free"
  "key-fail-closed": "free"

tiers:
  free:
    limit: 100
    window: 10s
    burst: 10

routes:
  - pattern: "/api/fail-open"
    methods: ["GET"]
    algorithm: "token_bucket"
    store: "redis"
    cost: 1
    fallback: "open"
    upstream: "%s"

  - pattern: "/api/fail-closed"
    methods: ["GET"]
    algorithm: "token_bucket"
    store: "redis"
    cost: 1
    fallback: "closed"
    upstream: "%s"
`, gatewayAddr, redisAddr, backendURL, backendURL)

	return os.WriteFile(path, []byte(cfg), 0644)
}

func TestBreakerFallback(t *testing.T) {
	// 1. Start a dummy backend on the hardcoded port 9000
	l, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		t.Fatalf("failed to listen on :9000: %v", err)
	}
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go backend.Serve(l)
	defer backend.Close()
	backendURL := "http://127.0.0.1:9000"

	// 2. Build the gateway binary
	tempDir := t.TempDir()
	gatewayBin := filepath.Join(tempDir, "gateway")
	buildCmd := exec.Command("go", "build", "-o", gatewayBin, "../../cmd/gateway")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build gateway: %v\nOutput:\n%s", err, out)
	}

	// 3. Write config with a bad Redis address (simulating down Redis)
	cfgPath := filepath.Join(tempDir, "gateway.yaml")
	gatewayAddr := ":8082"
	redisAddr := "127.0.0.1:9999" // Nothing listening here
	gatewayURL := "http://localhost:8082"
	if err := writeBreakerConfig(cfgPath, gatewayAddr, redisAddr, backendURL); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// 4. Start gateway
	cmd := exec.Command(gatewayBin, cfgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start gateway: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// 5. Wait for gateway to be ready
	time.Sleep(500 * time.Millisecond)

	// Helper to make requests
	makeReq := func(path, apiKey string) (int, time.Duration) {
		req, _ := http.NewRequest("GET", gatewayURL+path, nil)
		req.Header.Set("X-API-Key", apiKey)
		
		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		duration := time.Since(start)
		
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, duration
	}

	// 6. Test fail-open (breaker is closed, call to Redis will fail after 50ms)
	status, dur := makeReq("/api/fail-open", "key-fail-open")
	if status != http.StatusOK {
		t.Errorf("expected 200 OK for fail-open, got %d", status)
	}
	// The first failure will take at least DialTimeout (50ms)
	if dur < 40*time.Millisecond {
		t.Errorf("expected request to take >= 40ms due to Redis timeout, took %v", dur)
	}

	// 7. Test fail-closed (breaker is still closed, 2nd request)
	status, dur = makeReq("/api/fail-closed", "key-fail-closed")
	if status != http.StatusServiceUnavailable { // 503
		t.Errorf("expected 503 Service Unavailable for fail-closed, got %d", status)
	}

	// 8. Breaker should now be OPEN (MinRequests=2, ErrorRatio=0.5).
	// Subsequent requests should fail FAST (return immediately without waiting 50ms).
	
	status, dur = makeReq("/api/fail-open", "key-fail-open")
	if status != http.StatusOK {
		t.Errorf("expected 200 OK for fail-open when breaker open, got %d", status)
	}
	if dur > 20*time.Millisecond {
		t.Errorf("expected request to fail FAST (<20ms) because breaker is open, took %v", dur)
	}

	status, dur = makeReq("/api/fail-closed", "key-fail-closed")
	if status != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable for fail-closed when breaker open, got %d", status)
	}
	if dur > 20*time.Millisecond {
		t.Errorf("expected request to fail FAST (<20ms) because breaker is open, took %v", dur)
	}
}
