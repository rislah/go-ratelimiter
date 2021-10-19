package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/rislah/ratelimiter"
	"github.com/stretchr/testify/assert"
)

type datastoreTestCase struct {
	client    *redis.Client
	datastore ratelimiter.Datastore

	field          string
	limitPerMinute int
	windowInterval time.Duration
	bucketInterval time.Duration
}

func TestDatastore(t *testing.T) {
	tests := []struct {
		scenario       string
		field          string
		limitPerMinute int
		windowInterval time.Duration
		bucketInterval time.Duration
		test           func(ctx context.Context, testCase datastoreTestCase)
	}{
		{
			scenario:       "should increment",
			field:          "test:ip",
			limitPerMinute: 2,
			windowInterval: 5 * time.Minute,
			bucketInterval: 1 * time.Minute,
			test: func(ctx context.Context, testCase datastoreTestCase) {
				counter, throttle, _, err := testCase.datastore.IncrementSlidingWindow(ctx, testCase.field, testCase.limitPerMinute, testCase.windowInterval, testCase.bucketInterval)
				assert.NoError(t, err)
				assert.False(t, throttle)
				assert.Equal(t, 0, counter)
				keys, err := testCase.client.Keys(ctx, "*").Result()
				assert.NoError(t, err)
				assert.NotEmpty(t, keys)
			},
		},
		{
			scenario:       "should throttle if over limit within window limit",
			field:          "test:ip",
			limitPerMinute: 1,
			windowInterval: 1 * time.Minute,
			bucketInterval: 5 * time.Second,
			test: func(ctx context.Context, testCase datastoreTestCase) {
				fn := func(i int) bool {
					counter, throttle, _, err := testCase.datastore.IncrementSlidingWindow(ctx, testCase.field, testCase.limitPerMinute, testCase.windowInterval, testCase.bucketInterval)
					assert.NoError(t, err)
					assert.Equal(t, i, counter)
					return throttle
				}

				throttle := fn(0)
				assert.False(t, throttle)
				throttle = fn(1)
				assert.True(t, throttle)
			},
		},
		{
			scenario:       "should not throttle if within window limit",
			field:          "test:ip",
			limitPerMinute: 1,
			windowInterval: 1 * time.Second,
			bucketInterval: 1 * time.Second,
			test: func(ctx context.Context, testCase datastoreTestCase) {
				fn := func(i int) bool {
					counter, throttle, _, err := testCase.datastore.IncrementSlidingWindow(ctx, testCase.field, testCase.limitPerMinute, testCase.windowInterval, testCase.bucketInterval)
					assert.NoError(t, err)
					assert.Equal(t, i, counter)
					return throttle
				}

				throttle := fn(0)
				assert.False(t, throttle)

				<-time.After(1 * time.Second)

				throttle = fn(0)
				assert.False(t, throttle)
				keys, err := testCase.client.Keys(ctx, "*").Result()
				assert.NoError(t, err)
				assert.Empty(t, keys)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			srv, err := miniredis.Run()
			assert.NoError(t, err)

			client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer client.FlushAll(ctx)
			defer cancel()
			defer srv.Close()

			ds := ratelimiter.NewRedisDatastore(client)
			test.test(ctx, datastoreTestCase{
				client:         client,
				datastore:      ds,
				field:          test.field,
				limitPerMinute: test.limitPerMinute,
				windowInterval: test.windowInterval,
				bucketInterval: test.bucketInterval,
			})
		})
	}
}
