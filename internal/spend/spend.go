package spend

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/auth"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
)

var (
	ModelPrice = map[string][]float64{} // model group -> [input token price, output token price]
)

type SpendRecord struct {
	RecordId         int       `gorm:"primaryKey;column:record_id"`
	RequestId        string    `gorm:"column:request_id"`
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

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type UpstreamResp struct {
	Id         string     `json:"id"`
	CreateTime time.Time  `json:"create"`
	Model      string     `json:"model"`
	Object     string     `json:"object"`
	Usage      TokenUsage `json:"usage"`
}

func CreateSpendRecord(c *gin.Context, ch chan<- SpendRecord) {
	var upstreamResp UpstreamResp
	if err := json.Unmarshal(c.MustGet("upstreamResp").([]byte), &upstreamResp); err != nil {
		log.Fatal("Failed to decode upstream response:", err)
		return
	}
	key := c.MustGet("key").(auth.Key)
	model := c.MustGet("modelGroup").(string)
	spend := ModelPrice[model][0]*float64(upstreamResp.Usage.PromptTokens) +
		ModelPrice[model][1]*float64(upstreamResp.Usage.CompletionTokens)
	record := SpendRecord{
		RequestId:        upstreamResp.Id,
		AuthKey:          key.KeyContent,
		UserId:           key.UserId,
		OrganizationId:   key.OrganizationId,
		ModelGroup:       model,
		Spend:            spend,
		TotalTokens:      upstreamResp.Usage.TotalTokens,
		PromptTokens:     upstreamResp.Usage.PromptTokens,
		CompletionTokens: upstreamResp.Usage.CompletionTokens,
		CreateTime:       time.Now(),
	}
	ch <- record
}

func InsertBatchSpendRecord(records []SpendRecord) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_spend_log").Create(&records).Error; err != nil {
		log.Fatal("Failed to insert batch spend records:", err)
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
	if err := db.Table("janus_spend_log").Where("create_time BETWEEN ? AND ?", startTime, endTime).Find(&records).Error; err != nil {
		log.Fatal("Failed to get spend records:", err)
		return nil, err
	}

	return records, nil
}
