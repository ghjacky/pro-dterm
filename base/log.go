package base

import (
	"log"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfiguration struct {
	Path       string
	Level      string
	MaxAge     uint16
	MaxSize    uint16	// megabytes
	MaxBackups uint16
}

var Log = logrus.New()

func initLog() {
	if len(Conf.LogConfiguration.Path) == 0 {
		Conf.LogConfiguration.Path = "/dev/stdout"
	}
	logger := &lumberjack.Logger{
		// 日志输出文件路径
		Filename: Conf.LogConfiguration.Path,
		// 日志文件最大 size, 单位是 MB
		MaxSize: int(Conf.LogConfiguration.MaxAge), // megabytes
		// 最大过期日志保留的个数
		MaxBackups: int(Conf.LogConfiguration.MaxBackups),
		// 保留过期文件的最大时间间隔,单位是天
		MaxAge: int(Conf.LogConfiguration.MaxAge), //days
		// 是否需要压缩滚动日志, 使用的 gzip 压缩
		Compress: true, // disabled by default
	}
	Log.SetOutput(logger) //调用 logrus 的 SetOutput()函数
	level, err := logrus.ParseLevel(Conf.LogConfiguration.Level)
	if err != nil {
		log.Printf("failed to parse log level set in config file: %s", err.Error())
		log.Printf("using default log level: %s", logrus.InfoLevel)
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)
	Log.SetFormatter(&logrus.TextFormatter{})
}
