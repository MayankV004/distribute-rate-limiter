#!/bin/bash
go build -o scratch/gateway ./cmd/gateway

echo "=== L1 CACHE DISABLED ==="
redis-cli config resetstat > /dev/null
redis-cli flushall > /dev/null
sed -i 's/enabled: true/enabled: false/' scratch/gateway.bench.yaml
./scratch/gateway scratch/gateway.bench.yaml > /dev/null 2>&1 &
GW_PID=$!
sleep 1
go run scratch/bench.go
kill $GW_PID
wait $GW_PID 2>/dev/null || true
redis-cli info commandstats | grep cmdstat_evalsha

echo ""
echo "=== L1 CACHE ENABLED ==="
redis-cli config resetstat > /dev/null
redis-cli flushall > /dev/null
sed -i 's/enabled: false/enabled: true/' scratch/gateway.bench.yaml
./scratch/gateway scratch/gateway.bench.yaml > /dev/null 2>&1 &
GW_PID=$!
sleep 1
go run scratch/bench.go
kill $GW_PID
wait $GW_PID 2>/dev/null || true
redis-cli info commandstats | grep cmdstat_evalsha

