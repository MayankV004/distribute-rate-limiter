package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"net"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go backend.Serve(l)
	defer backend.Close()

	// First run without L1 Cache, then with L1 Cache.
	// Since gateway isn't running yet, we just start it up?
	// The user expects us to measure over-admission and Redis-call reduction.
	
	// We'll spawn the gateway, run requests, then kill it.
	fmt.Println("Running benchmark...")
	var successCount atomic.Int32
	var failCount atomic.Int32

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				req, _ := http.NewRequest("GET", "http://localhost:8080/api/v1/orders", nil)
				req.Header.Set("X-API-Key", "alice-key")
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					if resp.StatusCode == 200 {
						successCount.Add(1)
					} else {
						failCount.Add(1)
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Elapsed: %v\n", time.Since(start))
	fmt.Printf("Admitted: %d, Denied: %d\n", successCount.Load(), failCount.Load())
}
