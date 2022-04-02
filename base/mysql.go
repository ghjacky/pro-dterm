package base

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlConfiguration struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
}

var Db *gorm.DB

func initMysql() {
	db, err := gorm.Open(
		mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			Conf.MysqlConfiguration.User,
			Conf.MysqlConfiguration.Password,
			Conf.MysqlConfiguration.Host,
			Conf.MysqlConfiguration.Port,
			Conf.MysqlConfiguration.Database)),
		&gorm.Config{},
	)
	if err != nil {
		Log.Fatalf("mysql connect failed: %s", err.Error())
	}
	Db = db.Debug()
}

func closeDB() {
}

func MigrateDB(db ...interface{}) {
	if err := Db.Set("gorm:table_options", "CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(db...); err != nil {
		Log.Fatal(err.Error())
	}
}
