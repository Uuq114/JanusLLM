package janusDb

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DatabaseDsn string
)

func ConnectDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(DatabaseDsn), &gorm.Config{})
	if err != nil {
		log.Printf("ConnectDatabase: failed to connect: %v", err)
		return nil, err
	}
	return db, nil
}

func CloseDatabaseConnection(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}
