#!/bin/bash

# Experiment B: Latency
# Vegeta ramping RPS to measure latency overhead for Local and Redis stores.

echo "=== EXPERIMENT B: Latency (Local Store) ==="
sed -i 's/store: redis/store: local/g' configs/gateway.yaml
docker compose -f deployments/docker-compose.yaml up -d --force-recreate gateway1 gateway2 gateway3
sleep 10

for RATE in 100 500 1000; do
    echo "--- Rate: $RATE RPS ---"
    echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=$RATE -duration=3s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=json > scratch/local_latency_$RATE.json
    cat scratch/local_latency_$RATE.json | jq '{rate, latencies}'
done

echo ""
echo "=== EXPERIMENT B: Latency (Redis Store) ==="
sed -i 's/store: local/store: redis/g' configs/gateway.yaml
docker compose -f deployments/docker-compose.yaml up -d --force-recreate gateway1 gateway2 gateway3
sleep 10
docker compose -f deployments/docker-compose.yaml exec redis redis-cli flushall

for RATE in 100 500 1000; do
    echo "--- Rate: $RATE RPS ---"
    echo "GET http://localhost:80/api/v1/search" | ~/go/bin/vegeta attack -rate=$RATE -duration=3s -header="X-API-Key: key_free_example" | ~/go/bin/vegeta report -type=json > scratch/redis_latency_$RATE.json
    cat scratch/redis_latency_$RATE.json | jq '{rate, latencies}'
done

