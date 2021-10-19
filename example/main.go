package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rislah/ratelimiter"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

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
}
