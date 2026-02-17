package redis

import goredis "github.com/redis/go-redis/v9"

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type redisImpl struct {
	client *goredis.Client
}
