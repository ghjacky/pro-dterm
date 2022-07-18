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

var db *gorm.DB

func initMysql() {
	var err error
	db, err = gorm.Open(
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
}

func DB() *gorm.DB{
	return db
}
func closeDB() {

}

func MigrateDB(dbs ...interface{}) {
	if err := db.Set("gorm:table_options", "CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(dbs...); err != nil {
		Log.Fatal(err.Error())
	}
}
