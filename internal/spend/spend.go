package spend

import (
	"encoding/json"
	"fmt"
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

const (
	ContextUpstreamResp  = "upstreamResp"
	ContextProvider      = "provider"
	ContextUpstream      = "upstream"
	ContextUpstreamModel = "upstreamModel"
	ContextLatencyMS     = "latency_ms"
	ContextCacheHit      = "cache_hit"
	ContextSpend         = "spend"
)

type SpendRecord struct {
	RecordId         int       `gorm:"primaryKey;column:record_id"`
	RequestId        string    `gorm:"column:request_id"`
	KeyId            int       `gorm:"column:key_id"`
	KeyContent       string    `gorm:"column:key_content"`
	TeamId           int       `gorm:"column:team_id"`
	OrganizationId   int       `gorm:"column:organization_id"`
	Tenant           string    `gorm:"column:tenant"`
	ModelGroup       string    `gorm:"column:model_group"`
	Provider         string    `gorm:"column:provider"`
	LatencyMS        int64     `gorm:"column:latency_ms"`
	CacheHit         bool      `gorm:"column:cache_hit"`
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
	resp, exists := c.Get(ContextUpstreamResp)
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
	if upstreamResp.Usage.TotalTokens <= 0 && upstreamResp.Usage.PromptTokens <= 0 && upstreamResp.Usage.CompletionTokens <= 0 {
		log.Printf("CreateSpendRecord: missing token usage for request_id=%s; spend record skipped", upstreamResp.Id)
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
		KeyId:            key.KeyId,
		KeyContent:       auth.RedactKeyContent(key.KeyContent),
		TeamId:           key.TeamId,
		OrganizationId:   key.OrganizationId,
		Tenant:           tenantFromKey(key),
		ModelGroup:       model,
		Provider:         stringContext(c, ContextProvider),
		LatencyMS:        int64Context(c, ContextLatencyMS),
		CacheHit:         boolContext(c, ContextCacheHit),
		Spend:            spend,
		TotalTokens:      upstreamResp.Usage.TotalTokens,
		PromptTokens:     upstreamResp.Usage.PromptTokens,
		CompletionTokens: upstreamResp.Usage.CompletionTokens,
	}
	c.Set(ContextSpend, spend)
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
	spendValue, exists := c.Get(ContextSpend)
	if !exists {
		return
	}
	spendAmount, ok := spendValue.(float64)
	if !ok {
		log.Printf("CreateKeySpendRecord: spend type mismatch: %T", spendValue)
		return
	}
	ch <- spendAmount
}

func tenantFromKey(key auth.Key) string {
	return fmt.Sprintf("org:%d/team:%d", key.OrganizationId, key.TeamId)
}

func stringContext(c *gin.Context, key string) string {
	value, exists := c.Get(key)
	if !exists {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		log.Printf("CreateSpendRecord: %s type mismatch: %T", key, value)
		return ""
	}
	return text
}

func int64Context(c *gin.Context, key string) int64 {
	value, exists := c.Get(key)
	if !exists {
		return 0
	}
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		log.Printf("CreateSpendRecord: %s type mismatch: %T", key, value)
		return 0
	}
}

func boolContext(c *gin.Context, key string) bool {
	value, exists := c.Get(key)
	if !exists {
		return false
	}
	typed, ok := value.(bool)
	if !ok {
		log.Printf("CreateSpendRecord: %s type mismatch: %T", key, value)
		return false
	}
	return typed
}

func UpdateKeySpendRecord(totalSpend float64, keyID int) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Printf("UpdateKeySpendRecord: connect database failed: %v", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_key").
		Where("key_id = ?", keyID).
		Updates(map[string]interface{}{
			"balance":     gorm.Expr("balance - ?", totalSpend),
			"total_spend": gorm.Expr("total_spend + ?", totalSpend),
		}).Error; err != nil {
		log.Printf("UpdateKeySpendRecord: update failed: %v", err)
	}
}
