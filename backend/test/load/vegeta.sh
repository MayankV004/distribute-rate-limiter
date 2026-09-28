#!/usr/bin/env bash
# Phase 8 load test — uses vegeta to generate ramping RPS and produce a latency report.
# Results are written to docs/BENCHMARKS.md.
# TODO (Phase 8): implement after the full stack is running.

set -euo pipefail

TARGET_HOST="${TARGET_HOST:-http://localhost}"
API_KEY="${API_KEY:-load_test_key}"
DURATION="${DURATION:-60s}"
RESULTS_FILE="../../docs/BENCHMARKS.md"

echo "## Load Test Results" >> "$RESULTS_FILE"
echo "Date: $(date -u)" >> "$RESULTS_FILE"
echo "Target: $TARGET_HOST" >> "$RESULTS_FILE"
echo "Duration: $DURATION" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Ramp from 100 to 5000 RPS
for rps in 100 500 1000 2000 5000; do
  echo "=== ${rps} RPS ===" | tee -a "$RESULTS_FILE"
  vegeta attack \
    -targets targets.txt \
    -rate="${rps}/s" \
    -duration="$DURATION" \
    -header="X-API-Key: $API_KEY" \
    | vegeta report --type=text \
    | tee -a "$RESULTS_FILE"
  echo "" >> "$RESULTS_FILE"
done

echo "Load test complete. Results in $RESULTS_FILE"
