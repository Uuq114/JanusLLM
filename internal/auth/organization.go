package auth

import (
	"log"

	janusDb "github.com/Uuq114/JanusLLM/internal/db"
)

type Organization struct {
	OrganizationId   int    `gorm:"primaryKey;column:organization_id"`
	OrganizationName string `gorm:"column:organization_name"`
}

func CreateOrganizationRecord(name string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	result := db.Table("janus_auth_organization").Create(&Organization{OrganizationName: name})
	if result.Error != nil {
		log.Fatal("Failed to create organization record, err:", result.Error)
		return
	}
}

func GetOrganizationRecord(name string) *Organization {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil
	}
	defer janusDb.CloseDatabaseConnection(db)
	var organization Organization
	result := db.Table("janus_auth_organization").Where("organization_name = ?", name).First(&organization)
	if result.Error != nil {
		log.Fatal("Failed to get organization record, err:", result.Error)
		return nil
	}
	return &organization
}

func UpdateOrganizationRecord(name string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	var organization Organization
	result := db.Table("janus_auth_organization").Where("organization_name = ?", name).First(&organization)
	if result.Error != nil {
		log.Fatal("Failed to get organization record, err:", result.Error)
		return
	}
	result = db.Table("janus_auth_organization").Save(&organization)
	if result.Error != nil {
		log.Fatal("Failed to update organization record, err:", result.Error)
		return
	}
}
func DeleteOrganizationRecord(name string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	var organization Organization
	result := db.Table("janus_auth_organization").Where("organization_name = ?", name).First(&organization)
	if result.Error != nil {
		log.Fatal("Failed to get organization record, err:", result.Error)
		return
	}
	result = db.Table("janus_auth_organization").Delete(&organization)
	if result.Error != nil {
		log.Fatal("Failed to delete organization record, err:", result.Error)
		return
	}
}
