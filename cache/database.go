package cache

import (
	"context"
	"math/rand"
	"time"

	// goredis "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type CacheConfig struct {
	Address     string
	Password    string
	MaxActive   int           //最大连接数，即最多的tcp连接数，一般建议往大的配置，但不要超过操作系统文件句柄个数（centos下可以ulimit -n查看）
	MaxIdle     int           //最大空闲连接数，即会有这么多个连接提前等待着，但过了超时时间也会关闭
	IdleTimeout time.Duration //空闲连接超时时间，但应该设置比redis服务器超时时间短。否则服务端超时了，客户端保持着连接也没用
	Wait        bool          //如果超过最大连接，是报错，还是等待
	DbNumber    int           //选择数据库
}

var (
	_rdb *goredislib.Client

	_pool redis.Pool
	_rds  *redsync.Redsync
)

func Init() {
	viper.SetConfigType("toml")
	viper.SetConfigFile("./conf/config-db.toml")
	viper.ReadInConfig()

	opt := &goredislib.Options{
		Addr:            viper.GetString("redis.address"),
		Password:        viper.GetString("redis.password"),
		PoolSize:        viper.GetInt("redis.pool_size"),
		MinIdleConns:    viper.GetInt("redis.min_idle"),
		ConnMaxIdleTime: viper.GetDuration("redis.idle_timeout"),
		DB:              viper.GetInt("redis.db_number"),
	}
	if opt.PoolSize == 0 {
		opt.PoolSize = 1000
	}
	if opt.MinIdleConns == 0 {
		opt.MinIdleConns = 500
	}
	if opt.ConnMaxIdleTime == 0 {
		opt.ConnMaxIdleTime = 120 * time.Second
	}
	log.Infof("[Redis] Addr=%s, PoolSize=%d, MinIdle=%d, ConnMaxIdleTime=%s, DB=%d", opt.Addr, opt.PoolSize, opt.MinIdleConns, opt.ConnMaxIdleTime.String(), opt.DB)

	_rdb = goredislib.NewClient(opt)
	if _, err := _rdb.Ping(context.Background()).Result(); err != nil {
		log.Errorf("test go-redis connection error : %v", err)
		_rdb = nil
		return
	}

	_pool = goredis.NewPool(_rdb)
	_rds = redsync.New(_pool)
}

func Close() {
	if _rdb != nil {
		_rdb.Close()
	}
}

func NewMutex(name string) *redsync.Mutex {
	return _rds.NewMutex(name,
		redsync.WithExpiry(5*time.Second),
		redsync.WithRetryDelayFunc(func(tries int) time.Duration {
			return time.Duration(rand.Intn(20)+5) * time.Millisecond
		}),
		redsync.WithTries(200))
}

func Redis() *goredislib.Client {
	return _rdb
}
