package auth

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type Organization struct {
	OrganizationId   int    `gorm:"primaryKey;column:organization_id"`
	OrganizationName string `gorm:"column:organization_name"`
}

func CreateOrganizationRecord(name string) {
	if err := CreateOrganizationRecordWithError(name); err != nil {
		log.Printf("CreateOrganizationRecord: %v", err)
	}
}

func CreateOrganizationRecordWithError(name string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	result := db.Table("janus_auth_organization").Create(&Organization{OrganizationName: name})
	if result.Error != nil {
		return fmt.Errorf("create organization record: %w", result.Error)
	}
	return nil
}

func GetOrganizationRecord(name string) *Organization {
	organization, err := GetOrganizationRecordWithError(name)
	if err != nil {
		log.Printf("GetOrganizationRecord: %v", err)
		return nil
	}
	return organization
}

func GetOrganizationRecordWithError(name string) (*Organization, error) {
	db, err := connectAuthDatabase()
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	return getOrganizationRecord(db, name)
}

func getOrganizationRecord(db *gorm.DB, name string) (*Organization, error) {
	var organization Organization
	result := db.Table("janus_auth_organization").Where("organization_name = ?", name).First(&organization)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get organization record: %w", result.Error)
	}
	return &organization, nil
}

func UpdateOrganizationRecord(name string) {
	if err := UpdateOrganizationRecordWithError(name); err != nil {
		log.Printf("UpdateOrganizationRecord: %v", err)
	}
}

func UpdateOrganizationRecordWithError(name string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	organization, err := getOrganizationRecord(db, name)
	if err != nil {
		return err
	}
	if organization == nil {
		return fmt.Errorf("organization record not found: %s", name)
	}
	result := db.Table("janus_auth_organization").Save(organization)
	if result.Error != nil {
		return fmt.Errorf("update organization record: %w", result.Error)
	}
	return nil
}
func DeleteOrganizationRecord(name string) {
	if err := DeleteOrganizationRecordWithError(name); err != nil {
		log.Printf("DeleteOrganizationRecord: %v", err)
	}
}

func DeleteOrganizationRecordWithError(name string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	organization, err := getOrganizationRecord(db, name)
	if err != nil {
		return err
	}
	if organization == nil {
		return fmt.Errorf("organization record not found: %s", name)
	}
	result := db.Table("janus_auth_organization").Delete(organization)
	if result.Error != nil {
		return fmt.Errorf("delete organization record: %w", result.Error)
	}
	return nil
}
