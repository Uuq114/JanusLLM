package auth

import (
	"database/sql/driver"
	"fmt"
	"log"
	"strings"
	"time"

	janusDb "github.com/Uuq114/JanusLLM/internal/db"
)

type Key struct {
	KeyId          int         `gorm:"primaryKey;column:key_id"`
	KeyContent     string      `gorm:"column:key_content"`
	KeyName        string      `gorm:"column:key_name"`
	ModelList      StringSlice `gorm:"column:model_list"`
	UserId         int         `gorm:"column:user_id"`
	OrganizationId int         `gorm:"column:organization_id"`

	Balance           float64 `gorm:"column:balance"`
	TotalSpend        float64 `gorm:"column:total_spend"`
	RequestPerMinute  int     `gorm:"column:request_per_minute"`
	SpendLimitPerWeek float64 `gorm:"column:spend_limit_per_week"`

	CreateTime time.Time `gorm:"column:create_time"`
	ExpireTime time.Time `gorm:"column:expire_time"`
}

type StringSlice []string

func ToString(s StringSlice) string {
	return strings.Join(s, ",")
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]uint8)
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

func CreateKeyRecord(keyContent string, keyName string, modelList []string, userName string, organizationName string,
	balance float64, requestPerMinute int, spendLimitPerWeek float64) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("CreateKeyRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	user := GetUserRecord(userName)
	if user == nil {
		log.Printf("CreateKeyRecord: user record not found: %s", userName)
		return
	}
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Printf("CreateKeyRecord: organization record not found: %s", organizationName)
		return
	}

	result := db.Table("janus_auth_key").Omit("create_time").Create(&Key{
		KeyContent:        keyContent,
		KeyName:           keyName,
		ModelList:         modelList,
		UserId:            user.UserId,
		OrganizationId:    organization.OrganizationId,
		Balance:           balance,
		TotalSpend:        0,
		RequestPerMinute:  requestPerMinute,
		SpendLimitPerWeek: spendLimitPerWeek,
		ExpireTime:        time.Now().Add(30 * 24 * time.Hour),
	})
	if result.Error != nil {
		log.Printf("CreateKeyRecord: create failed: %v", result.Error)
	}
}

func CheckKeyRecord(keyContent string) bool {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("CheckKeyRecord: connect database failed: %v", err)
		return false
	}
	defer janusDb.CloseDatabaseConnection(db)

	var key Key
	result := db.Table("janus_auth_key").Where("key_content = ?", keyContent).First(&key)
	if result.Error != nil {
		log.Printf("CheckKeyRecord: query failed: %v", result.Error)
		return false
	}
	if key.ExpireTime.Before(time.Now()) {
		log.Printf("CheckKeyRecord: key expired")
		return false
	}
	return true
}

func GetAllValidKey() []Key {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("GetAllValidKey: connect database failed: %v", err)
		return nil
	}
	defer janusDb.CloseDatabaseConnection(db)

	var keys []Key
	result := db.Table("janus_auth_key").
		Where("balance > 0").
		Where("expire_time > ? OR expire_time IS NULL", time.Now()).
		Find(&keys)
	if result.Error != nil {
		log.Printf("GetAllValidKey: query failed: %v", result.Error)
		return nil
	}
	return keys
}

func DeleteKeyRecord(keyContent string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("DeleteKeyRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_key").Where("key_content = ?", keyContent).Delete(&Key{})
	if result.Error != nil {
		log.Printf("DeleteKeyRecord: delete failed: %v", result.Error)
	}
}
