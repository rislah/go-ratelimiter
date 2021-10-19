package ratelimiter

import (
	"context"
	"strconv"
	"time"
)

type Datastore interface {
	IncrementSlidingWindow(ctx context.Context, field string, limitPerMinute int, windowInterval, bucketInterval time.Duration) (int, bool, int, error)
}

func bucketsByTime(bucketInterval time.Duration, expireTime, now time.Time) []string {
	buckets := []string{}
	for bucket := expireTime; bucket.After(now); bucket = bucket.Add(-1 * bucketInterval) {
		bucketUnix := int(bucket.Unix())
		buckets = append(buckets, strconv.Itoa(bucketUnix))
	}

	return buckets
}
func ttlSecondsFromExpirationTime(expireTime, now time.Time) int {
	return int(expireTime.Sub(now) / time.Second)
}
