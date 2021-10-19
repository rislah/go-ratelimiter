package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultBucketInterval = 5 * time.Second
	defaultWindowInterval = 1 * time.Minute
)

type Field struct {
	Scope      string
	Identifier string
}

type Ratelimiter struct {
	name           string
	datastore      Datastore
	limitPerMinute int
	windowInterval time.Duration
	bucketInterval time.Duration
	writeHeaders   bool
	devMode        bool
}

type Options struct {
	Name           string
	Datastore      Datastore
	LimitPerMinute int
	WindowInterval time.Duration
	BucketInterval time.Duration
	WriteHeaders   bool
	DevMode        bool
}

func NewRateLimiter(opts *Options) *Ratelimiter {
	if opts.BucketInterval == 0 {
		opts.BucketInterval = defaultBucketInterval
	}

	if opts.WindowInterval == 0 {
		opts.WindowInterval = defaultWindowInterval
	}

	return &Ratelimiter{
		name:           opts.Name,
		datastore:      opts.Datastore,
		devMode:        opts.DevMode,
		limitPerMinute: opts.LimitPerMinute,
		windowInterval: opts.WindowInterval,
		bucketInterval: opts.BucketInterval,
		writeHeaders:   opts.WriteHeaders,
	}
}

func (r *Ratelimiter) ShouldThrottle(ctx context.Context, w http.ResponseWriter, field Field) (bool, error) {
	fk := fieldKey(r.name, field.Scope, field.Identifier)
	currentCounter, throttled, earliestExp, err := r.datastore.IncrementSlidingWindow(ctx, fk, r.limitPerMinute, r.windowInterval, r.bucketInterval)
	if err != nil {
		return false, err
	}

	if r.writeHeaders {
		remainingStr := strconv.Itoa(remaining(currentCounter, r.limitPerMinute))
		limitStr := strconv.Itoa(r.limitPerMinute)
		resetStr := strconv.Itoa(earliestExp)
		w.Header().Add("RateLimit-Limit", limitStr)
		if earliestExp != 0 {
			w.Header().Add("RateLimit-Reset", resetStr)
		}
		w.Header().Add("RateLimit-Remaining", remainingStr)
	}

	if r.devMode {
		return false, nil
	}

	return throttled, nil
}

func fieldKey(name, scope, identifier string) string {
	return fmt.Sprintf("%s:%s:%s", name, scope, identifier)
}

func remaining(current, limit int) int {
	diff := limit - current
	if diff < 0 {
		return 0
	}

	return diff
}
