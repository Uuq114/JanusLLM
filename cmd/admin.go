package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Uuq114/JanusLLM/internal/auth"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
	"github.com/Uuq114/JanusLLM/internal/proxy"
)

const adminMasterUser = "admin"

func registerAdminRoutes(r *gin.Engine, config AdminConfig, logger *zap.Logger) {
	admin := r.Group("/v1/admin")
	admin.Use(adminAuthMiddleware(config, logger))
	{
		admin.GET("/organizations", listOrganizations)
		admin.POST("/organizations", createOrganization)
		admin.GET("/organizations/:organization_id", getOrganization)
		admin.PATCH("/organizations/:organization_id", updateOrganization)
		admin.DELETE("/organizations/:organization_id", deleteOrganization)

		admin.GET("/teams", listTeams)
		admin.POST("/teams", createTeam)
		admin.GET("/teams/:team_id", getTeam)
		admin.PATCH("/teams/:team_id", updateTeam)
		admin.DELETE("/teams/:team_id", deleteTeam)

		admin.GET("/keys", listKeys)
		admin.POST("/keys", createKey)
		admin.GET("/keys/:key_id", getKey)
		admin.PATCH("/keys/:key_id", updateKey)
		admin.DELETE("/keys/:key_id", deleteKey)
	}
}

func adminAuthMiddleware(config AdminConfig, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok || !isMasterAdminCredential(config.MasterKey, username, password) {
			logger.Warn("admin auth failed", zap.String("username", username))
			c.Header("WWW-Authenticate", `Basic realm="JanusLLM Admin"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin credentials"})
			c.Abort()
			return
		}

		c.Set("adminUser", username)
		c.Next()
	}
}

func isMasterAdminCredential(masterKey string, username string, password string) bool {
	if masterKey == "" {
		return false
	}
	userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(adminMasterUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(masterKey)) == 1
	return userMatch && passMatch
}

type organizationDTO struct {
	OrganizationID   int64     `gorm:"column:organization_id" json:"organization_id"`
	OrganizationName string    `gorm:"column:organization_name" json:"organization_name"`
	CreateTime       time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime       time.Time `gorm:"column:update_time" json:"update_time"`
}

type organizationRequest struct {
	OrganizationName string `json:"organization_name" binding:"required"`
}

type organizationPatchRequest struct {
	OrganizationName *string `json:"organization_name"`
}

func listOrganizations(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var organizations []organizationDTO
	if err := db.Table("janus_auth_organization").Order("organization_id").Find(&organizations).Error; err != nil {
		respondDBError(c, "list organizations failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": organizations})
}

func createOrganization(c *gin.Context) {
	var req organizationRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	organization := organizationDTO{OrganizationName: strings.TrimSpace(req.OrganizationName)}
	if organization.OrganizationName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization_name is required"})
		return
	}
	if err := db.Table("janus_auth_organization").
		Omit("organization_id", "create_time", "update_time").
		Create(&organization).Error; err != nil {
		respondDBError(c, "create organization failed", err)
		return
	}
	c.JSON(http.StatusCreated, organization)
}

func getOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var organization organizationDTO
	if !firstByID(c, db.Table("janus_auth_organization").Where("organization_id = ?", id), &organization) {
		return
	}
	c.JSON(http.StatusOK, organization)
}

func updateOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}
	var req organizationPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	updates := map[string]interface{}{}
	if req.OrganizationName != nil {
		name := strings.TrimSpace(*req.OrganizationName)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "organization_name cannot be empty"})
			return
		}
		updates["organization_name"] = name
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_organization").Where("organization_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update organization failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	getOrganization(c)
}

func deleteOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_organization").Where("organization_id = ?", id).Delete(&organizationDTO{})
	if result.Error != nil {
		respondDBError(c, "delete organization failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

type teamDTO struct {
	TeamID         int64     `gorm:"column:user_id" json:"team_id"`
	TeamName       string    `gorm:"column:user_name" json:"team_name"`
	OrganizationID int64     `gorm:"column:organization_id" json:"organization_id"`
	CreateTime     time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime     time.Time `gorm:"column:update_time" json:"update_time"`
}

type teamRequest struct {
	TeamName       string `json:"team_name" binding:"required"`
	OrganizationID int64  `json:"organization_id" binding:"required"`
}

type teamPatchRequest struct {
	TeamName       *string `json:"team_name"`
	OrganizationID *int64  `json:"organization_id"`
}

func listTeams(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var teams []teamDTO
	if err := db.Table("janus_auth_user").Order("user_id").Find(&teams).Error; err != nil {
		respondDBError(c, "list teams failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": teams})
}

func createTeam(c *gin.Context) {
	var req teamRequest
	if !bindAdminJSON(c, &req) {
		return
	}
	team := teamDTO{
		TeamName:       strings.TrimSpace(req.TeamName),
		OrganizationID: req.OrganizationID,
	}
	if team.TeamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team_name is required"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_user").
		Omit("user_id", "create_time", "update_time").
		Create(&team).Error; err != nil {
		respondDBError(c, "create team failed", err)
		return
	}
	c.JSON(http.StatusCreated, team)
}

func getTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "team_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var team teamDTO
	if !firstByID(c, db.Table("janus_auth_user").Where("user_id = ?", id), &team) {
		return
	}
	c.JSON(http.StatusOK, team)
}

func updateTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "team_id")
	if !ok {
		return
	}
	var req teamPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	updates := map[string]interface{}{}
	if req.TeamName != nil {
		name := strings.TrimSpace(*req.TeamName)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "team_name cannot be empty"})
			return
		}
		updates["user_name"] = name
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_user").Where("user_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update team failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	getTeam(c)
}

func deleteTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "team_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_user").Where("user_id = ?", id).Delete(&teamDTO{})
	if result.Error != nil {
		respondDBError(c, "delete team failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

type keyDTO struct {
	KeyID             int64            `gorm:"column:key_id" json:"key_id"`
	KeyContent        string           `gorm:"column:key_content" json:"key_content"`
	KeyName           string           `gorm:"column:key_name" json:"key_name"`
	ModelList         auth.StringSlice `gorm:"column:model_list" json:"model_list"`
	TeamID            int64            `gorm:"column:user_id" json:"team_id"`
	OrganizationID    int64            `gorm:"column:organization_id" json:"organization_id"`
	Balance           float64          `gorm:"column:balance" json:"balance"`
	TotalSpend        float64          `gorm:"column:total_spend" json:"total_spend"`
	RequestPerMinute  int              `gorm:"column:request_per_minute" json:"request_per_minute"`
	SpendLimitPerWeek float64          `gorm:"column:spend_limit_per_week" json:"spend_limit_per_week"`
	CreateTime        time.Time        `gorm:"column:create_time" json:"create_time"`
	UpdateTime        time.Time        `gorm:"column:update_time" json:"update_time"`
	ExpireTime        *time.Time       `gorm:"column:expire_time" json:"expire_time"`
}

type keyRequest struct {
	KeyContent        string     `json:"key_content"`
	KeyName           string     `json:"key_name" binding:"required"`
	ModelList         []string   `json:"model_list"`
	TeamID            int64      `json:"team_id" binding:"required"`
	OrganizationID    int64      `json:"organization_id" binding:"required"`
	Balance           float64    `json:"balance"`
	RequestPerMinute  int        `json:"request_per_minute"`
	SpendLimitPerWeek float64    `json:"spend_limit_per_week"`
	ExpireTime        *time.Time `json:"expire_time"`
}

type keyPatchRequest struct {
	KeyContent        *string    `json:"key_content"`
	KeyName           *string    `json:"key_name"`
	ModelList         *[]string  `json:"model_list"`
	TeamID            *int64     `json:"team_id"`
	OrganizationID    *int64     `json:"organization_id"`
	Balance           *float64   `json:"balance"`
	RequestPerMinute  *int       `json:"request_per_minute"`
	SpendLimitPerWeek *float64   `json:"spend_limit_per_week"`
	ExpireTime        *time.Time `json:"expire_time"`
}

func listKeys(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var keys []keyDTO
	if err := db.Table("janus_auth_key").Order("key_id").Find(&keys).Error; err != nil {
		respondDBError(c, "list keys failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func createKey(c *gin.Context) {
	var req keyRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	keyContent := strings.TrimSpace(req.KeyContent)
	if keyContent == "" {
		generated, err := generateKeyContent()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "generate key failed"})
			return
		}
		keyContent = generated
	}

	modelList := auth.StringSlice(req.ModelList)
	if len(modelList) == 0 {
		modelList = auth.StringSlice{"*"}
	}

	key := keyDTO{
		KeyContent:        keyContent,
		KeyName:           strings.TrimSpace(req.KeyName),
		ModelList:         modelList,
		TeamID:            req.TeamID,
		OrganizationID:    req.OrganizationID,
		Balance:           req.Balance,
		TotalSpend:        0,
		RequestPerMinute:  req.RequestPerMinute,
		SpendLimitPerWeek: req.SpendLimitPerWeek,
		ExpireTime:        req.ExpireTime,
	}
	if key.KeyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_name is required"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_key").
		Omit("key_id", "create_time", "update_time").
		Create(&key).Error; err != nil {
		respondDBError(c, "create key failed", err)
		return
	}
	c.JSON(http.StatusCreated, key)
}

func getKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var key keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &key) {
		return
	}
	c.JSON(http.StatusOK, key)
}

func updateKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}
	var req keyPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var existing keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &existing) {
		return
	}

	updates := map[string]interface{}{}
	if req.KeyContent != nil {
		keyContent := strings.TrimSpace(*req.KeyContent)
		if keyContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key_content cannot be empty"})
			return
		}
		updates["key_content"] = keyContent
	}
	if req.KeyName != nil {
		keyName := strings.TrimSpace(*req.KeyName)
		if keyName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key_name cannot be empty"})
			return
		}
		updates["key_name"] = keyName
	}
	if req.ModelList != nil {
		modelList := auth.StringSlice(*req.ModelList)
		if len(modelList) == 0 {
			modelList = auth.StringSlice{"*"}
		}
		updates["model_list"] = modelList
	}
	if req.TeamID != nil {
		updates["user_id"] = *req.TeamID
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}
	if req.Balance != nil {
		updates["balance"] = *req.Balance
	}
	if req.RequestPerMinute != nil {
		updates["request_per_minute"] = *req.RequestPerMinute
	}
	if req.SpendLimitPerWeek != nil {
		updates["spend_limit_per_week"] = *req.SpendLimitPerWeek
	}
	if req.ExpireTime != nil {
		updates["expire_time"] = *req.ExpireTime
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	result := db.Table("janus_auth_key").Where("key_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update key failed", result.Error)
		return
	}
	invalidateKeyCache(existing.KeyContent)
	if updatedContent, ok := updates["key_content"].(string); ok {
		invalidateKeyCache(updatedContent)
	}
	getKey(c)
}

func deleteKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var existing keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &existing) {
		return
	}
	result := db.Table("janus_auth_key").Where("key_id = ?", id).Delete(&keyDTO{})
	if result.Error != nil {
		respondDBError(c, "delete key failed", result.Error)
		return
	}
	invalidateKeyCache(existing.KeyContent)
	c.Status(http.StatusNoContent)
}

func invalidateKeyCache(keyContent string) {
	deleteCachedKey(keyContent)
	proxy.RemoveRequestRing(keyContent)
}

func generateKeyContent() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "sk-" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func connectAdminDB(c *gin.Context) (*gorm.DB, bool) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return nil, false
	}
	return db, true
}

func bindAdminJSON(c *gin.Context, out interface{}) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " must be a positive integer"})
		return 0, false
	}
	return id, true
}

func firstByID(c *gin.Context, query *gorm.DB, out interface{}) bool {
	if err := query.First(out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return false
		}
		respondDBError(c, "query resource failed", err)
		return false
	}
	return true
}

func respondDBError(c *gin.Context, message string, err error) {
	status := http.StatusInternalServerError
	if isConstraintError(err) {
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": message})
}

func isConstraintError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "violates foreign key") ||
		strings.Contains(msg, "violates unique constraint") ||
		strings.Contains(msg, "constraint")
}
