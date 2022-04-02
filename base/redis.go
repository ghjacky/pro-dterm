package base

import (
	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type RedisConfiguration struct {
	Addr     string
	Password string
	DB       uint8
}

var RedisClient *goredislib.Client
var GlobalSyncInstance *redsync.Redsync

func initRedis() {
	RedisClient = goredislib.NewClient(&goredislib.Options{
		Addr:     Conf.RedisConfiguration.Addr,
		Password: Conf.RedisConfiguration.Password,
		DB:       int(Conf.RedisConfiguration.DB),
	})
	pool := goredis.NewPool(RedisClient)
	GlobalSyncInstance = redsync.New(pool)
}

func GenerateMutex(key string) *redsync.Mutex {
	return GlobalSyncInstance.NewMutex(key)
}
