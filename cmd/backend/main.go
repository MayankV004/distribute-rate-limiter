package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := flag.String("addr", ":9000", "Listen address")
	latency := flag.Duration("latency", 0, "Simulated backend processing time")
	flag.Parse()

	mux := http.NewServeMux()

	// A simple health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// The catch-all endpoint that echoes the request back
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if *latency > 0 {
			time.Sleep(*latency)
		}

		// Convert headers to a normal map for JSON
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		response := map[string]interface{}{
			"message": "Hello from the dummy backend!",
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": headers,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	fmt.Printf("Starting dummy backend on %s (Latency: %s)\n", *addr, *latency)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
