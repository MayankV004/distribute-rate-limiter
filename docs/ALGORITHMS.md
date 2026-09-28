# Algorithms

This rate limiter supports four distinct algorithms, allowing you to choose the precise mathematical model that best fits your traffic shaping and protection needs. All algorithms can be run locally (in-memory) or distributed via atomic Lua scripts in Redis.

---

## 1. Token Bucket
The Token Bucket is the industry standard for API quotas (used by AWS, Stripe, etc.). It allows for sudden bursts of traffic while enforcing a strict long-term rate.

- **State:** Tracks two values per key: `tokens` (a float representing available capacity) and `last_refill_ms` (timestamp of the last request).
- **Allow/Deny Logic:** On every request, the algorithm calculates how much time has passed since `last_refill_ms` and adds new tokens to the bucket at the configured rate, up to a maximum `burst` capacity. If `tokens >= cost`, the request is allowed and tokens are deducted.
- **Burst Behavior:** Handles sudden traffic spikes flawlessly. If a user hasn't made requests recently, their bucket fills to the maximum burst capacity, allowing them to instantly consume those tokens.
- **Memory:** `O(1)` per key. It only requires storing two numbers, making it incredibly memory efficient.
- **When to use:** Ideal for public API gateways, billing quotas, and scenarios where you want to allow clients to burst occasionally but maintain a steady average rate.

---

## 2. Sliding Window Counter
This is the **Production Default** for many high-scale systems because it provides a smooth, accurate rate limit without the memory overhead of logging individual requests.

- **State:** Tracks 2 counters per key: the `current_window` count and the `previous_window` count.
- **Weighted Overlap Formula:** Instead of resetting abruptly at the end of a minute (which allows 2x the traffic at the boundary), this algorithm calculates a weighted sum based on the current timestamp. For example, if you are 30% into the current minute, your usage is calculated as: `(previous_window * 0.7) + current_window`.
- **Max Approximation Error:** Because it assumes traffic in the previous window was evenly distributed, the maximum theoretical error is `±(limit/window)`. At scale, this is mathematically negligible.
- **Memory:** `O(1)` per key. Requires storing only two integers.
- **When to use:** Best for general-purpose rate limiting where you want a smooth, rolling window limit (e.g., 1000 requests per rolling minute) with perfect memory efficiency.

---

## 3. Sliding Window Log
The most accurate algorithm possible, but mathematically and physically the most expensive to run.

- **State:** A sorted set (or array) containing the exact timestamp of every single request made by the user in the current window.
- **Allow/Deny Logic:** When a request arrives, it removes all timestamps older than the window, counts the remaining timestamps, and allows the request if the count is below the limit.
- **Memory:** `O(requests in window)` per key. This is why it is rarely used at high scale. If a limit is 10,000 requests per minute, Redis must store 10,000 individual timestamps for that single user.
- **Adversarial Case:** A malicious client could intentionally hover exactly at the limit, forcing the server to constantly prune and calculate massive arrays of timestamps, creating unnecessary CPU/Memory pressure on Redis.
- **When to use:** Only for small, highly sensitive limits where 100% mathematical precision is required (e.g., "3 failed login attempts per minute" or fraud detection limits).

---

## 4. Leaky Bucket
The Leaky Bucket is the odd one out because it doesn't just *admit* traffic; it mathematically *shapes* traffic.

- **State:** Tracks `level` (how full the bucket is) and `last_drain_ms`.
- **Constant Output Rate vs. Burst Tolerance:** Incoming requests "fill" the bucket. The bucket "leaks" (processes requests) at a strictly constant rate. If the bucket overflows, requests are dropped.
- **Distributed Limitation:** While we implemented the math in Redis to deny requests when the bucket overflows, true "traffic shaping" (holding the request in a queue until it drips out) requires local server queueing. In our distributed implementation, it acts as a rigid, burst-intolerant rate limiter.
- **When to use:** Excellent for protecting legacy backend systems that will crash if they receive bursts. It forces clients to space out their requests evenly.

---

## Tradeoff Table

| Algorithm | Memory Profile | Accuracy | Burst Tolerance | Redis State Size (per key) |
| :--- | :--- | :--- | :--- | :--- |
| **Token Bucket** | `O(1)` | High | Configurable (`burst`) | 2 values (tokens, timestamp) |
| **Sliding Window Counter** | `O(1)` | High (Approximate) | Low (Smooths traffic) | 2 values (curr, prev) |
| **Sliding Window Log** | `O(N)` | 100% Perfect | None | N values (all timestamps) |
| **Leaky Bucket** | `O(1)` | High | None (Strict output) | 2 values (level, timestamp) |
