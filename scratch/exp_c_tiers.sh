#!/bin/bash

# Experiment C: Tier Comparison
# This script compares the Free Tier (1,000 req/min) vs Pro Tier (25,000 req/min)
# by firing 500 requests per second at both for 5 seconds.

echo "=== Ensuring Gateway Cluster is Running ==="
docker compose -f deployments/docker-compose.yaml up -d
sleep 2 # Give it a moment to ensure it's ready

echo ""
echo "================================================="
echo "   TEST 1: FREE TIER (Quota: 1,000 req / minute) "
echo "================================================="
echo "Sending 500 requests per second for 5 seconds (2,500 total requests)..."

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=500 -duration=5s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=text

echo ""
echo "================================================="
echo "   TEST 2: PRO TIER (Quota: 25,000 req / minute) "
echo "================================================="
echo "Sending 500 requests per second for 5 seconds (2,500 total requests)..."

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=500 -duration=5s -header="X-API-Key: key_pro_example" | ~/go/bin/vegeta report -type=text

echo ""
echo "Done! Compare the 'Success [ratio]' above to see how the tiers enforced limits differently."
