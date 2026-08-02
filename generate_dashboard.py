import json

panels = [
    {
        "title": "Decisions Rate (Allow/Deny)",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0},
        "targets": [
            {
                "expr": "sum(rate(ratelimit_decisions_total[1m])) by (decision)",
                "legendFormat": "{{decision}}"
            }
        ]
    },
    {
        "title": "Store Latency (p99)",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0},
        "targets": [
            {
                "expr": "histogram_quantile(0.99, sum(rate(ratelimit_store_latency_seconds_bucket[1m])) by (le))",
                "legendFormat": "p99"
            }
        ]
    },
    {
        "title": "Circuit Breaker State",
        "type": "stat",
        "gridPos": {"h": 8, "w": 8, "x": 0, "y": 8},
        "targets": [
            {
                "expr": "ratelimit_breaker_state",
                "legendFormat": "{{name}}"
            }
        ]
    },
    {
        "title": "Store Errors Rate",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 8, "x": 8, "y": 8},
        "targets": [
            {
                "expr": "sum(rate(ratelimit_store_errors_total[1m])) by (kind)",
                "legendFormat": "{{kind}}"
            }
        ]
    },
    {
        "title": "Fallback Rate",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 8, "x": 16, "y": 8},
        "targets": [
            {
                "expr": "sum(rate(ratelimit_fallback_total[1m])) by (policy)",
                "legendFormat": "{{policy}}"
            }
        ]
    }
]

dashboard = {
    "title": "Rate Limiter Dashboard",
    "schemaVersion": 38,
    "panels": panels,
    "time": {
        "from": "now-15m",
        "to": "now"
    },
    "refresh": "5s"
}

with open("deployments/grafana/dashboard.json", "w") as f:
    json.dump(dashboard, f, indent=2)
