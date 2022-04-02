package base

import (
	"github.com/spf13/viper"
	"log"
)

type Configuration struct {
	Path string
	MainConfiguration
	LogConfiguration
	MysqlConfiguration
	RedisConfiguration
}

type MainConfiguration struct {
	Listen string
}

var Conf = new(Configuration)

func (c *Configuration) Parse() {
	if len(c.Path) <= 0 {
		log.Fatalln("no config file specified")
	}
	viper.SetConfigFile(c.Path)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
	log.Printf("using config file: %s", c.Path)
	Conf.MainConfiguration.Listen = viper.GetString("main.listen")

	Conf.LogConfiguration.Path = viper.GetString("log.path")
	Conf.LogConfiguration.Level = viper.GetString("log.level")
	Conf.LogConfiguration.MaxAge = uint16(viper.GetInt("log.max_age"))
	Conf.LogConfiguration.MaxBackups = uint16(viper.GetInt("log.max_backups"))
	Conf.LogConfiguration.MaxSize = uint16(viper.GetInt("log.max_size"))

	Conf.MysqlConfiguration.Host = viper.GetString("mysql.host")
	Conf.MysqlConfiguration.Port = uint16(viper.GetInt("mysql.port"))
	Conf.MysqlConfiguration.Database = viper.GetString("mysql.database")
	Conf.MysqlConfiguration.User = viper.GetString("mysql.user")
	Conf.MysqlConfiguration.Password = viper.GetString("mysql.password")

	Conf.RedisConfiguration.Addr = viper.GetString("redis.addr")
	Conf.RedisConfiguration.DB = uint8(viper.GetInt("redis.db"))
	Conf.RedisConfiguration.Password = viper.GetString("redis.password")
}
