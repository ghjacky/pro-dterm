package base

func Init() {
	Conf.Parse()
	initLog()
	initMysql()
	// initRedis()
}
