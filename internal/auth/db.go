package auth

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	MysqlDsn string
)

func ConnectDatabase() (*gorm.DB, error) {
	log.Println("Connecting to database with DSN:", MysqlDsn)
	db, err := gorm.Open(mysql.Open(MysqlDsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil, err
	}
	return db, nil
}

func CloseDatabaseConnection(db *gorm.DB) {
	sqlDB, _ := db.DB()
	sqlDB.Close()
}
