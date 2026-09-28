#!/bin/bash

# Experiment C: Tier Comparison
# This script clearly tests the Free Tier vs Pro Tier limits under the 1:1 cost structure.

echo "=== Ensuring Gateway Cluster is Running ==="
docker compose -f deployments/docker-compose.yaml up -d
docker compose -f deployments/docker-compose.yaml unpause redis 2>/dev/null || true
sleep 2 # Give it a moment to ensure it's ready

echo ""
echo "================================================="
echo "   TEST 1: FREE TIER                             "
echo "================================================="
echo "LIMIT: 1,000 requests per minute"
echo "COST:  1 token per request"
echo "---"
echo "Action: Sending 2,000 requests over 2 seconds (1,000 RPS)."
echo "Expectation: Exactly 1,000 requests should succeed (plus ~33 tokens refilled during the 2s window). The rest will fail with 429."
echo ""

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=1000 -duration=2s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=text

echo ""
echo "================================================="
echo "   TEST 2: PRO TIER (MAX THROUGHPUT)             "
echo "================================================="
echo "LIMIT: 1,000,000 requests per minute (Max)"
echo "COST:  1 token per request"
echo "---"
echo "Action: Sending 10,000 requests over 2 seconds (5,000 RPS)."
echo "Expectation: ALL 10,000 requests should succeed. Pro tier absorbs the massive spike without dropping a single request."
echo ""

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=5000 -duration=2s -header="X-API-Key: key_pro_example" | ~/go/bin/vegeta report -type=text

echo ""
echo "Done! The math is now perfectly 1:1!"
