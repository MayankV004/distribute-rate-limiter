#!/bin/bash

# Experiment E: Real-World Scenario Testing (k6)
# This script runs a k6 virtual user simulation with mixed tier traffic and "think time".
# It executes k6 via Docker, so no local k6 installation is required.

echo "=== Ensuring Gateway Cluster is Running ==="
docker compose -f deployments/docker-compose.yaml up -d
docker compose -f deployments/docker-compose.yaml unpause redis 2>/dev/null || true

echo ""
echo "================================================="
echo "   STARTING k6 REAL-WORLD SCENARIO TEST          "
echo "================================================="
echo "Simulating up to 1000 virtual users with mixed traffic (Free + Pro tiers)."
echo ""

# Run k6 via Docker, mounting the local script and using host networking
# so it can reach the gateway at localhost:80.
docker run --rm -i --network host grafana/k6 run - < scratch/k6_scenario.js

echo ""
echo "================================================="
echo "   k6 TEST COMPLETE                              "
echo "================================================="
echo "Notice how k6 reports on p(90) and p(95) latency, and how the"
echo "Free Tier users likely hit 429s while Pro users sailed through."
