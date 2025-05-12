package auth

import (
	"database/sql/driver"
	"fmt"
	"log"
	"strings"
	"time"
)

type Key struct {
	KeyId          int         `gorm:"primaryKey;column:key_id"`
	KeyContent     string      `gorm:"column:key_content"`
	KeyName        string      `gorm:"column:key_name"`
	ModelList      StringSlice `gorm:"column:model_list"`
	UserId         int         `gorm:"column:user_id"`
	OrganizationId int         `gorm:"column:organization_id"`
	CreateTime     time.Time   `gorm:"column:create_time"`
	ExpireTime     time.Time   `gorm:"column:expire_time"`
}

type StringSlice []string

// handle model_list format change between text/[]string

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]uint8) // sql scan returns []uint8
	if !ok {
		return fmt.Errorf("unsupported data type: %T", value)
	}
	str := string(bytes)
	*s = strings.Split(str, ",")
	return nil
}

func (s *StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return strings.Join(*s, ","), nil
}

// CRUD

func CreateKeyRecord(keyContent string, keyName string, modelList []string, userName string, organizationName string) {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer CloseDatabaseConnection(db)
	user := GetUserRecord(userName)
	if user == nil {
		log.Fatal("User record not found, name", userName)
		return
	}
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Fatal("Organization record not found, name", organizationName)
		return
	}
	result := db.Table("janus_auth_key").Create(&Key{
		KeyContent:     keyContent,
		KeyName:        keyName,
		ModelList:      modelList,
		UserId:         user.UserId,
		OrganizationId: organization.OrganizationId,
		CreateTime:     time.Now(),
		ExpireTime:     time.Now().Add(30 * 24 * time.Hour), // 30 days
	})
	if result.Error != nil {
		log.Fatal("Failed to create key record, err:", result.Error)
		return
	}
}

func CheckKeyRecord(keyContent string) bool {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return false
	}
	defer CloseDatabaseConnection(db)
	var key Key
	result := db.Table("janus_auth_key").Where("key_content = ?", keyContent).First(&key)
	if result.Error != nil {
		log.Fatal("Failed to get key record, err:", result.Error)
		return false
	}
	if key.ExpireTime.Before(time.Now()) {
		log.Println("Key expired")
		return false
	}
	return true
}

func GetAllValidKey() []Key {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil
	}
	defer CloseDatabaseConnection(db)
	var keys []Key
	result := db.Table("janus_auth_key").Where("expire_time > ? OR expire_time IS NULL", time.Now()).Find(&keys)
	if result.Error != nil {
		log.Fatal("Failed to get all valid key record, err:", result.Error)
		return nil
	}
	return keys
}

// key is not updatable, UpdateKeyRecord() is deleted

func DeleteKeyRecord(keyContent string) {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer CloseDatabaseConnection(db)
	result := db.Table("janus_auth_key").Where("key_content = ?", keyContent).Delete(&Key{})
	if result.Error != nil {
		log.Fatal("Failed to delete key record, err:", result.Error)
		return
	}
}
