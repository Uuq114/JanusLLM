package spend

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
	TeamId           int       `gorm:"column:team_id"`
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
	if ch == nil {
		return
	}
	resp, exists := c.Get("upstreamResp")
	if !exists {
		return
	}

	respBytes, ok := resp.([]byte)
	if !ok {
		log.Printf("CreateSpendRecord: upstreamResp type mismatch: %T", resp)
		return
	}

	var upstreamResp UpstreamResp
	if err := json.Unmarshal(respBytes, &upstreamResp); err != nil {
		log.Printf("CreateSpendRecord: failed to decode upstream response: %v", err)
		return
	}

	key := c.MustGet("key").(auth.Key)
	model := c.MustGet("modelGroup").(string)
	price, ok := ModelPrice[model]
	if !ok || len(price) < 2 {
		log.Printf("CreateSpendRecord: missing model price config for model group: %s", model)
		return
	}

	spend := price[0]*float64(upstreamResp.Usage.PromptTokens) + price[1]*float64(upstreamResp.Usage.CompletionTokens)
	record := SpendRecord{
		RequestId:        upstreamResp.Id,
		AuthKey:          key.KeyContent,
		TeamId:           key.TeamId,
		OrganizationId:   key.OrganizationId,
		ModelGroup:       model,
		Spend:            spend,
		TotalTokens:      upstreamResp.Usage.TotalTokens,
		PromptTokens:     upstreamResp.Usage.PromptTokens,
		CompletionTokens: upstreamResp.Usage.CompletionTokens,
	}
	c.Set("spend", spend)
	ch <- record
}

func InsertBatchSpendRecord(records []SpendRecord) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("InsertBatchSpendRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_spend_log").Omit("create_time").Create(&records).Error; err != nil {
		log.Printf("InsertBatchSpendRecord: insert failed: %v", err)
	}
}

func GetRangeSpendRecords(startTime time.Time, endTime time.Time) ([]SpendRecord, error) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("GetRangeSpendRecords: connect database failed: %v", err)
		return nil, err
	}
	defer janusDb.CloseDatabaseConnection(db)

	var records []SpendRecord
	if err := db.Table("janus_spend_log").Where("create_time BETWEEN ? AND ?", startTime, endTime).Find(&records).Error; err != nil {
		log.Printf("GetRangeSpendRecords: query failed: %v", err)
		return nil, err
	}

	return records, nil
}

func CreateKeySpendRecord(c *gin.Context, ch chan<- float64) {
	if ch == nil {
		return
	}
	spend := c.MustGet("spend").(float64)
	ch <- spend
}

func UpdateKeySpendRecord(totalSpend float64, key string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("UpdateKeySpendRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_key").
		Where("key_content = ?", key).
		Updates(map[string]interface{}{
			"balance":     gorm.Expr("balance - ?", totalSpend),
			"total_spend": gorm.Expr("total_spend + ?", totalSpend),
		}).Error; err != nil {
		log.Printf("UpdateKeySpendRecord: update failed: %v", err)
	}
}
