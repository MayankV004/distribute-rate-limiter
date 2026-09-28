import http from 'k6/http';
import { check, sleep } from 'k6';

// k6 Options: Ramp up to 1000 concurrent virtual users over 30 seconds
export const options = {
    stages: [
        { duration: '5s', target: 200 }, // Ramp up to 200 users
        { duration: '15s', target: 1000 }, // Ramp up to 1000 users and hold
        { duration: '5s', target: 0 },  // Ramp down to 0
    ],
    thresholds: {
        http_req_duration: ['p(95)<50'], // 95% of requests must complete below 50ms
    },
};

const BASE_URL = 'http://localhost:80';

export default function () {
    // 1. Simulate Mixed Traffic (80% Free Tier, 20% Pro Tier)
    const isProUser = Math.random() < 0.2;
    const apiKey = isProUser ? 'key_pro_example' : 'key_free_example';

    const params = {
        headers: {
            'X-API-Key': apiKey,
            'Content-Type': 'application/json',
        },
    };

    // 2. Simulate User Behavior: Hit a lightweight endpoint first
    let res = http.get(`${BASE_URL}/api/v1/public`, params);
    
    // Check if the request was successful or if the free tier got rate limited (429)
    check(res, {
        'public: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    });

    // Simulate "think time" - user reading the page before searching
    sleep(Math.random() * 0.2 + 0.05); // Sleep between 50ms and 250ms

    // 3. Simulate User Behavior: Hit a heavier endpoint
    res = http.get(`${BASE_URL}/api/v1/search`, params);
    
    check(res, {
        'search: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
    
    // Sleep before next iteration
    sleep(Math.random() * 0.1 + 0.1); 
}
