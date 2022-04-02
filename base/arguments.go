package base

import "flag"

func ParseFlag() {
	configFile := flag.String("config", "./configs/config.toml", "main server configuration file path")
	flag.Parse()
	Conf.Path = *configFile
}
