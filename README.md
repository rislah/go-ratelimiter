# go-ratelimiter
Scalable golang ratelimiter using the sliding window algorithm. Currently supports only Redis.

## Example usage
```
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
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(throttled)
```
### Headers
Optionally sets HTTP Headers to response.
https://tools.ietf.org/id/draft-polli-ratelimit-headers-00.html

## Design

All Redis operations are atomic. Fit for distributed environments.

Window interval is divided into smaller buckets. Each bucket is saved as hash key in Redis. Inside of each bucket, we have field and value.
Field is the unique identifier in the format of "ratelimiter_name:scope:value" to avoid collisions, eg: "test_limiter:ip:127.0.0.1". Value is incremented on each request in the bucket.

![alt text](https://i.imgur.com/RytpykX.png)

When we want to figure out if the user is above limit: 

![alt text](https://i.imgur.com/M6bIQbB.png)
