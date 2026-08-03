#!/bin/bash

# Experiment D: Chaos Testing (Circuit Breaker)
# This script simulates a Redis outage while under load.

echo "=== Ensuring Gateway Cluster is Running ==="
docker compose -f deployments/docker-compose.yaml up -d
docker compose -f deployments/docker-compose.yaml unpause redis 2>/dev/null || true
sleep 2

echo ""
echo "================================================="
echo "   STARTING CONTINUOUS LOAD (100 RPS for 15s)    "
echo "================================================="
echo "Watch carefully! We are going to kill Redis mid-flight."

# Start Vegeta in the background
echo "GET http://localhost:80/api/v1/public" | ~/go/bin/vegeta attack -rate=100 -duration=15s -header="X-API-Key: key_free_example" > scratch/chaos_results.bin &
VEGETA_PID=$!

sleep 3
echo "\n[CHAOS] 💥 PULLING THE PLUG ON REDIS NOW! 💥"
docker compose -f deployments/docker-compose.yaml pause redis

sleep 5
echo "\n[CHAOS] 🔌 BRINGING REDIS BACK ONLINE... 🔌"
docker compose -f deployments/docker-compose.yaml unpause redis

# Wait for Vegeta to finish
wait $VEGETA_PID

echo "\n================================================="
echo "   LOAD TEST COMPLETE. GENERATING REPORT...      "
echo "================================================="
~/go/bin/vegeta report -type=text scratch/chaos_results.bin

echo "\nCheck the Success [ratio] above. If the Circuit Breaker worked,"
echo "it should be near 100% because it failed-open to protect availability!"
