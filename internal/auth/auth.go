package auth

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	janusDb "github.com/Uuq114/JanusLLM/internal/db"
	"gorm.io/gorm"
)

type Key struct {
	KeyId          int         `gorm:"primaryKey;column:key_id"`
	KeyContent     string      `gorm:"column:key_content"`
	KeyName        string      `gorm:"column:key_name"`
	ModelList      StringSlice `gorm:"column:model_list"`
	TeamModelList  StringSlice `gorm:"column:team_model_list"`
	TeamId         int         `gorm:"column:team_id"`
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
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("unsupported data type: %T", value)
	}
	if str == "" {
		*s = StringSlice{}
		return nil
	}
	*s = strings.Split(str, ",")
	return nil
}

func (s *StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return strings.Join(*s, ","), nil
}

func EffectiveModelList(teamModelList StringSlice, keyModelList StringSlice) StringSlice {
	team := normalizedPermissionList(teamModelList)
	key := normalizedPermissionList(keyModelList)

	if isWildcardModelList(team) && isWildcardModelList(key) {
		return StringSlice{"*"}
	}
	if isWildcardModelList(team) {
		return key
	}
	if isWildcardModelList(key) {
		return team
	}

	allowed := make(map[string]struct{}, len(team))
	for _, model := range team {
		allowed[model] = struct{}{}
	}

	effective := make(StringSlice, 0, len(key))
	for _, model := range key {
		if _, ok := allowed[model]; ok {
			effective = append(effective, model)
		}
	}
	return effective
}

func normalizedPermissionList(modelList StringSlice) StringSlice {
	if len(modelList) == 0 {
		return StringSlice{"*"}
	}

	seen := make(map[string]struct{}, len(modelList))
	normalized := make(StringSlice, 0, len(modelList))
	for _, model := range modelList {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if model == "*" {
			return StringSlice{"*"}
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		normalized = append(normalized, model)
	}
	if len(normalized) == 0 {
		return StringSlice{"*"}
	}
	return normalized
}

func isWildcardModelList(modelList StringSlice) bool {
	return len(modelList) == 1 && modelList[0] == "*"
}

func CreateKeyRecord(keyContent string, keyName string, modelList []string, teamName string, organizationName string,
	balance float64, requestPerMinute int, spendLimitPerWeek float64) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("CreateKeyRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	team := GetTeamRecord(teamName)
	if team == nil {
		log.Printf("CreateKeyRecord: team record not found: %s", teamName)
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
		TeamId:            team.TeamId,
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
	key, err := GetValidKeyByContent(keyContent)
	if err != nil {
		return false
	}
	return key != nil
}

func GetKeyByContent(keyContent string) (*Key, error) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("GetKeyByContent: connect database failed: %v", err)
		return nil, err
	}
	defer janusDb.CloseDatabaseConnection(db)

	var key Key
	result := keyQuery(db).
		Where("key_content = ?", keyContent).
		First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("GetKeyByContent: query failed: %v", result.Error)
		return nil, result.Error
	}
	return &key, nil
}

func GetValidKeyByContent(keyContent string) (*Key, error) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("GetValidKeyByContent: connect database failed: %v", err)
		return nil, err
	}
	defer janusDb.CloseDatabaseConnection(db)

	var key Key
	result := keyQuery(db).
		Where("key_content = ?", keyContent).
		Where("balance > 0").
		Where("expire_time > ? OR expire_time IS NULL", time.Now()).
		First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("GetValidKeyByContent: query failed: %v", result.Error)
		return nil, result.Error
	}
	return &key, nil
}

func GetAllValidKey() []Key {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("GetAllValidKey: connect database failed: %v", err)
		return nil
	}
	defer janusDb.CloseDatabaseConnection(db)

	var keys []Key
	result := keyQuery(db).
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

func keyQuery(db *gorm.DB) *gorm.DB {
	return db.Table("janus_auth_key").
		Select("janus_auth_key.*, janus_auth_team.model_list AS team_model_list").
		Joins("JOIN janus_auth_team ON janus_auth_team.team_id = janus_auth_key.team_id")
}
