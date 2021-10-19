package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const atomicInc = `
local field = ARGV[1]
local limitPerMinute = tonumber(ARGV[2])
local newBucketKey = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local counter = 0
local time = 0

for _, bucket in ipairs(KEYS) do
    local resp = tonumber(redis.call('hget', bucket, field))
    if resp == nil then resp = 0 else time = tonumber(bucket) end
    counter = counter + resp
end

--if ARGV[4] then
--    redis.log(redis.LOG_NOTICE, "buckets=", unpack(KEYS))
--    redis.log(redis.LOG_NOTICE, "field[ARGV1]=", field)
--    redis.log(redis.LOG_NOTICE, "limitPerMinute[ARGV2]=", limitPerMinute)
--    redis.log(redis.LOG_NOTICE, "newBucketKey[ARGV3]=", newBucketKey)
--    redis.log(redis.LOG_NOTICE, "ttl[ARGV3]=", ttl)
--    redis.log(redis.LOG_NOTICE, "counter=", counter)
--    redis.log(redis.LOG_NOTICE, "time=", time)
--end


if counter >= limitPerMinute then
    return { counter, 1, time}
end

redis.call('hincrby', newBucketKey, field, 1)
redis.call('expire', newBucketKey, ttl)

return { counter, 0, time}
	    `

type redisDataStoreImpl struct {
	client *redis.Client
	sha    string

	scriptLoadOnce sync.Once
}

func NewRedisDatastore(client *redis.Client) Datastore {
	d := &redisDataStoreImpl{
		client: client,
	}

	d.scriptLoadOnce.Do(func() {
		sha, err := d.client.ScriptLoad(context.Background(), atomicInc).Result()
		if err != nil {
			log.Fatal(err)
		}

		d.sha = sha
	})

	return d
}

func (d *redisDataStoreImpl) IncrementSlidingWindow(ctx context.Context, field string, limitPerMinute int, windowInterval, bucketInterval time.Duration) (int, bool, int, error) {
	now := time.Now()
	expireTime := now.Add(windowInterval).Truncate(bucketInterval)
	ttl := ttlSecondsFromExpirationTime(expireTime, now)
	buckets := bucketsByTime(bucketInterval, expireTime, now)

	res, err := d.client.EvalSha(ctx, d.sha, buckets,
		field,
		limitPerMinute,
		expireTime.Unix(),
		ttl,
		false,
	).Slice()

	if err != nil {
		return 0, false, 0, err
	}

	counter, ok := res[0].(int64)
	if !ok {
		return 0, false, 0, errors.New(fmt.Sprintf("failed to cast counter: %v", reflect.TypeOf(counter)))
	}

	throttle, ok := res[1].(int64)
	if !ok {
		return 0, false, 0, errors.New(fmt.Sprintf("failed to cast throttle: %v", reflect.TypeOf(throttle)))
	}

	earliestExp, ok := res[2].(int64)
	if !ok {
		return 0, false, 0, errors.New(fmt.Sprintf("failed to cast earliestExp: %v", reflect.TypeOf(earliestExp)))
	}

	earliestExpTime := time.Unix(earliestExp, 0)
	earliestExpSecs := int(earliestExpTime.Sub(now) / time.Second)
	if earliestExpSecs < 0 {
		earliestExpSecs = ttl
	}

	return int(counter), throttle == 1, earliestExpSecs, nil
}
