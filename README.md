# go-ratelimiter
Scalable golang rate limiter using the sliding window algorithm. Currently supports only Redis.

## Algorithm Overview
Implements a **distributed sliding window counter** with these characteristics:

1. Divides time into fixed windows (e.g. 1 minute)
2. Each window contains multiple buckets (e.g. 5-second intervals)
3. Counters are stored in Redis hashes with automatic expiry

### How It Works
```
[Window: 1 minute]
┌───────────┬───────────┬───────────┬───────────┬───────────┬───────────┐
│ Bucket 1  │ Bucket 2  │ Bucket 3  │ Bucket 4  │ Bucket 5  │ Bucket 6  │
│ (0-5s)    │ (5-10s)   │ (10-15s)  │ (15-20s)  │ (20-25s)  │ (25-30s)  │
└───────────┴───────────┴───────────┴───────────┴───────────┴───────────┘
[Current time: 28s] → Active buckets: 3-6
```

### Redis Data Structure
```
Hash Key: "unix_timestamp_of_bucket_start" 
{
  "ratelimiter_name:scope:identifier": counter_value
}
Example:
"1625097605": {
  "api_limiter:ip:127.0.0.1": 3,
  "api_limiter:user:alice": 1
}
```

## Example Usage
```go
client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

rl := ratelimiter.NewRateLimiter(&ratelimiter.Options{
    Name:           "test",
    Datastore:      ratelimiter.NewRedisDatastore(client),
    LimitPerMinute: 5,
    WindowInterval: 1 * time.Minute,
    BucketInterval: 5 * time.Second,
    WriteHeaders:   false,
    DevMode:        false,
})

throttled, err := rl.ShouldThrottle(context.Background(), nil, ratelimiter.Field{
    Scope:      "user",
    Identifier: "127.0.0.1",
})
```

## Headers
When enabled, complies with [draft-polli-ratelimit-headers](https://tools.ietf.org/id/draft-polli-ratelimit-headers-00.html):
```
RateLimit-Limit: 10         # Maximum allowed requests
RateLimit-Remaining: 3      # Requests left in window  
RateLimit-Reset: 12         # Seconds until window reset
```

## Design

### Atomic Operations
All Redis operations use a single Lua script for atomic execution:
```lua
-- Pseudocode
sum = sum_counts_in_active_buckets()
if sum >= limit then
    return "throttle"
else
    increment_current_bucket()
    set_expiry()
end
```

### Performance Characteristics
| Aspect          | Benefit                          |
|-----------------|----------------------------------|
| Bucket size     | Balances precision vs memory use |
| Lua scripting   | Atomic operations                | 
| Hash storage    | Efficient counter grouping       |
| TTL management  | Automatic data cleanup           |

## Comparison to Other Algorithms
| Algorithm       | Advantages                      | Disadvantages               |
|-----------------|---------------------------------|-----------------------------|
| Fixed Window    | Simple implementation           | Burst traffic at boundaries |
| Token Bucket    | Smooths request bursts          | Complex distributed sync    |
| Sliding Log     | Perfect accuracy                | High memory usage           |
| **This System** | Balanced accuracy/performance   | Slightly higher Redis load  |
