#!/bin/bash

# Experiment F: Maximum Throughput Test (RateGate Benchmark equivalent)
# Fires exactly 5,000 Requests Per Second for 60 seconds against the Gateway.

echo "=== Ensuring Gateway Cluster is Running ==="
docker compose -f deployments/docker-compose.yaml up -d
docker compose -f deployments/docker-compose.yaml unpause redis 2>/dev/null || true

echo ""
echo "================================================="
echo "   STARTING MAX THROUGHPUT LOAD TEST (5,000 RPS) "
echo "================================================="
echo "Testing your Ryzen 5 5600H to prove we can sustain"
echo "5,000 RPS for 60 seconds straight (300,000 requests)!"
echo ""

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack \
    -rate=5000/s \
    -duration=60s \
    -header="X-API-Key: key_pro_example" \
    | ~/go/bin/vegeta report --type=text
