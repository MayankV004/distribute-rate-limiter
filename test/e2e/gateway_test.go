//go:build e2e

package e2e

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const baseConfig = `
server:
  addr: "%s"
  metrics_addr: ":0"
  read_timeout: 5s
  shutdown_grace: 5s

redis:
  addrs: ["%s"]
  pool_size: 10
  dial_timeout: 500ms
  command_timeout: 100ms

breaker:
  error_ratio: 0.5
  min_requests: 5
  open_duration: 5s
  half_open_successes: 2

identity:
  order: ["api_key"]
  api_key_header: "X-API-Key"
  default_tier: "free"
  unknown_key_policy: "default_tier"

api_keys:
  "free_key": "free"
  "pro_key": "pro"

tiers:
  free:
    limit: 2
    window: 10s
    burst: 2
  pro:
    limit: 10
    window: 10s
    burst: 10

routes:
  - pattern: "/api/v1/*"
    algorithm: token_bucket
    store: local
    cost: 1
    fallback: closed
    upstream: "%s"
`

func writeConfig(path string, gatewayAddr, redisAddr, upstreamURL string, freeLimit int) error {
	cfg := fmt.Sprintf(baseConfig, gatewayAddr, redisAddr, upstreamURL)
	// replace the free tier limit dynamically for the reload test
	cfg = strings.Replace(cfg, "limit: 2", fmt.Sprintf("limit: %d", freeLimit), 1)
	cfg = strings.Replace(cfg, "burst: 2", fmt.Sprintf("burst: %d", freeLimit), 1)
	return os.WriteFile(path, []byte(cfg), 0644)
}

func waitForGateway(url string) error {
	client := &http.Client{Timeout: 50 * time.Millisecond}
	for i := 0; i < 50; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("gateway not ready")
}

func sendRequests(url, apiKey string, count int) int {
	allowed := 0
	client := &http.Client{Timeout: 1 * time.Second}
	for i := 0; i < count; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-API-Key", apiKey)
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == 200 {
				allowed++
			}
			resp.Body.Close()
		}
	}
	return allowed
}

func TestGatewayHotReload(t *testing.T) {
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
	binPath := filepath.Join(tempDir, "gateway_bin")
	cmdBuild := exec.Command("go", "build", "-o", binPath, "../../cmd/gateway/main.go")
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("failed to build gateway: %v\n%s", err, out)
	}

	// 3. Write initial config (free limit = 2)
	cfgPath := filepath.Join(tempDir, "gateway.yaml")
	// Using port 8081 for gateway to avoid conflicts
	gatewayAddr := ":8081"
	redisAddr := "localhost:6379" // not actually used for store: local
	gatewayURL := "http://localhost:8081/api/v1/test"
	if err := writeConfig(cfgPath, gatewayAddr, redisAddr, backendURL, 2); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// 4. Start the gateway
	cmdRun := exec.Command(binPath, cfgPath)
	cmdRun.Stdout = os.Stdout
	cmdRun.Stderr = os.Stderr
	if err := cmdRun.Start(); err != nil {
		t.Fatalf("failed to start gateway: %v", err)
	}
	defer func() {
		cmdRun.Process.Signal(syscall.SIGKILL)
		cmdRun.Wait()
	}()

	// 5. Wait for it to be ready
	if err := waitForGateway(gatewayURL); err != nil {
		t.Fatalf("gateway didn't become ready: %v", err)
	}

	// 6. Test Free Tier (limit 2)
	allowed := sendRequests(gatewayURL, "free_key", 5)
	if allowed != 2 {
		t.Fatalf("expected 2 allowed requests for free tier, got %d", allowed)
	}

	// 7. Test Pro Tier (limit 10)
	allowed = sendRequests(gatewayURL, "pro_key", 5)
	if allowed != 5 {
		t.Fatalf("expected 5 allowed requests for pro tier, got %d", allowed)
	}

	// 8. Modify config to increase free limit to 5
	if err := writeConfig(cfgPath, gatewayAddr, redisAddr, backendURL, 5); err != nil {
		t.Fatalf("failed to rewrite config: %v", err)
	}

	// 9. Send SIGHUP to the gateway process
	if err := cmdRun.Process.Signal(syscall.SIGHUP); err != nil {
		t.Fatalf("failed to send SIGHUP: %v", err)
	}

	// Wait for reload
	time.Sleep(500 * time.Millisecond)

	// 10. Test Free Tier again (limit should now be 5)
	// Because the handler was swapped, the local store limiters were recreated,
	// so the remaining tokens are reset. The limit is now 5.
	allowed = sendRequests(gatewayURL, "free_key", 6)
	if allowed != 5 {
		t.Fatalf("expected 5 allowed requests for free tier after reload, got %d", allowed)
	}
}
