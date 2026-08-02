//go:build integration

package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// redisAddr is set by TestMain and shared by all test files in this package.
var redisAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	redisC, err := redis.RunContainer(ctx,
		redis.WithLogLevel(redis.LogLevelWarning),
		// The default 60s wait is too short on resource-constrained machines.
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ContainerRequest: tc.ContainerRequest{
				WaitingFor: wait.ForLog("Ready to accept connections").
					WithStartupTimeout(3 * time.Minute),
			},
		}),
	)
	if err != nil {
		log.Fatalf("integration: failed to start Redis container: %v", err)
	}

	addr, err := redisC.Endpoint(ctx, "")
	if err != nil {
		log.Fatalf("integration: failed to get Redis endpoint: %v", err)
	}
	redisAddr = addr

	code := m.Run()

	if err := redisC.Terminate(ctx); err != nil {
		log.Printf("integration: failed to terminate Redis container: %v", err)
	}

	os.Exit(code)
}
