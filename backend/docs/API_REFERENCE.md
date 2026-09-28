# API Reference

This document outlines how client applications should interact with the Distributed Rate Limiter & API Gateway.

## Authentication

By default, the gateway identifies clients using an API Key passed in the HTTP headers. 

**Header:** `X-API-Key`
**Example:**
```http
GET /api/v1/search HTTP/1.1
Host: api.example.com
X-API-Key: key_pro_example
```

*Note: The identity extractor can also be configured to resolve identity via IP address or JWT Subject (`sub` claim) if configured in `gateway.yaml`.*

## Rate Limiting Headers

Every HTTP response passing through the gateway will include standardized headers to inform the client of their current quota status.

| Header | Description |
| :--- | :--- |
| `X-Ratelimit-Limit` | The total number of tokens (requests) allowed in the current time window. |
| `X-Ratelimit-Remaining` | The number of tokens left in the current window. |
| `Retry-After` | Included **only** when a request is rate-limited. Indicates the number of seconds the client must wait before retrying. |

**Successful Response Example:**
```http
HTTP/1.1 200 OK
X-Ratelimit-Limit: 1000
X-Ratelimit-Remaining: 998
X-Request-Id: 123e4567-e89b-12d3-a456-426614174000
```

## HTTP Status Codes

The gateway intercepts requests before they reach the backend. It may return the following status codes directly:

### `429 Too Many Requests`
Returned when the client has exhausted their token quota. The request is dropped and never reaches the backend. The client must respect the `Retry-After` header.

### `503 Service Unavailable`
Returned if the backend is unreachable, or if the Redis cluster fails and the route's Circuit Breaker is configured to `fallback: closed`. 

*(Note: If the Circuit Breaker is configured to `fallback: open`, Redis failures will NOT result in 503s; the traffic will be allowed through to the backend).*
