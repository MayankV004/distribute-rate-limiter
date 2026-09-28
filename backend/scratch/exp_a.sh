#!/bin/bash

# Experiment A: Correctness (Local vs Redis)
# Using Vegeta to fire 1000 requests over 2s (500 RPS).

echo "=== EXPERIMENT A: Local Store ==="
sed -i 's/store: redis/store: local/g' configs/gateway.yaml
docker compose -f deployments/docker-compose.yaml up -d --force-recreate gateway1 gateway2 gateway3
sleep 10

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=500 -duration=2s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=json > scratch/local_report.json
cat scratch/local_report.json | jq '{status_codes}'

echo ""
echo "=== EXPERIMENT A: Redis Store ==="
sed -i 's/store: local/store: redis/g' configs/gateway.yaml
docker compose -f deployments/docker-compose.yaml up -d --force-recreate gateway1 gateway2 gateway3
sleep 10

# Wait for redis to clear any old state? Actually it's better to just flush it.
docker compose -f deployments/docker-compose.yaml exec redis redis-cli flushall

echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=500 -duration=2s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=json > scratch/redis_report.json
cat scratch/redis_report.json | jq '{status_codes}'

