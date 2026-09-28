//go:build integration

package integration

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// redisAddr is set by TestMain and shared by all test files in this package.
var redisAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	redisC, err := redis.RunContainer(
		ctx,
		testcontainers.WithImage("redis:7-alpine"),
		redis.WithLogLevel(redis.LogLevelWarning),
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
