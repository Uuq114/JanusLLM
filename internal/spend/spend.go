package spend

import (
	"log"
	"time"

	"github.com/Uuq114/JanusLLM/internal/auth"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
)

var (
	ModelPrice = map[string][]float64{} // model group -> [input token price, output token price]
)

type SpendRecord struct {
	RecordId         int       `gorm:"primaryKey;column:record_id"`
	RequestId        int       `gorm:"column:request_id"`
	AuthKey          string    `gorm:"column:auth_key"`
	UserId           int       `gorm:"column:user_id"`
	OrganizationId   int       `gorm:"column:organization_id"`
	ModelGroup       string    `gorm:"column:model_group"`
	Spend            float64   `gorm:"column:spend"`
	TotalTokens      int       `gorm:"column:total_tokens"`
	PromptTokens     int       `gorm:"column:prompt_tokens"`
	CompletionTokens int       `gorm:"column:completion_tokens"`
	CreateTime       time.Time `gorm:"column:create_time"`
}

// CRUD

func CreateSpendRecord(requestId int, authKey string, modelGroup string, totalTokens int,
	promptTokens int, completionTokens int, createTime time.Time) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var userId int
	var organizationId int
	keyRecord := auth.GetUserRecord(authKey)
	if keyRecord != nil {
		userId = keyRecord.UserId
		organizationId = keyRecord.OrganizationId
	}
	spend := ModelPrice[modelGroup][0]*float64(promptTokens) + ModelPrice[modelGroup][1]*float64(completionTokens)

	record := SpendRecord{
		RequestId:        requestId,
		AuthKey:          authKey,
		UserId:           userId,
		OrganizationId:   organizationId,
		ModelGroup:       modelGroup,
		Spend:            spend,
		TotalTokens:      totalTokens,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		CreateTime:       createTime,
	}

	if err := db.Create(&record).Error; err != nil {
		log.Fatal("Failed to create spend record:", err)
	}
}

func GetRangeSpendRecords(startTime time.Time, endTime time.Time) ([]SpendRecord, error) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil, err
	}
	defer janusDb.CloseDatabaseConnection(db)

	var records []SpendRecord
	if err := db.Where("create_time BETWEEN ? AND ?", startTime, endTime).Find(&records).Error; err != nil {
		log.Fatal("Failed to get spend records:", err)
		return nil, err
	}

	return records, nil
}
