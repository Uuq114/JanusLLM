package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
	"github.com/Uuq114/JanusLLM/internal/models"
)

const (
	defaultEndpointWeight         = 100
	defaultEndpointTimeoutSeconds = 60
	defaultEndpointRetryTimes     = 1
)

type modelGroupRecord struct {
	GroupID            int64   `gorm:"column:group_id"`
	GroupName          string  `gorm:"column:group_name"`
	Strategy           string  `gorm:"column:strategy"`
	CostPerInputToken  float64 `gorm:"column:cost_per_input_token"`
	CostPerOutputToken float64 `gorm:"column:cost_per_output_token"`
	RequestDefaults    []byte  `gorm:"column:request_defaults"`
	Enabled            bool    `gorm:"column:enabled"`
}

type modelEndpointRecord struct {
	EndpointID        int64  `gorm:"column:endpoint_id"`
	GroupID           int64  `gorm:"column:group_id"`
	EndpointName      string `gorm:"column:endpoint_name"`
	ProviderType      string `gorm:"column:provider_type"`
	UpstreamModelName string `gorm:"column:upstream_model_name"`
	BaseURL           string `gorm:"column:base_url"`
	APIKeySecretRef   string `gorm:"column:api_key_secret_ref"`
	Weight            int    `gorm:"column:weight"`
	TimeoutSeconds    int    `gorm:"column:timeout_seconds"`
	RetryTimes        int    `gorm:"column:retry_times"`
	SkipTLSVerify     bool   `gorm:"column:skip_tls_verify"`
	Enabled           bool   `gorm:"column:enabled"`
}

type configSyncPlan struct {
	Groups           []modelGroupRecord
	Endpoints        []plannedModelEndpoint
	DisableGroups    []modelGroupRecord
	DisableEndpoints []modelEndpointRecord
}

type plannedModelEndpoint struct {
	GroupName string
	modelEndpointRecord
}

type endpointPlanKey struct {
	GroupName    string
	EndpointName string
}

func syncConfigToDB(config *JanusConfig, logger *zap.Logger) error {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		return err
	}
	defer janusDb.CloseDatabaseConnection(db)

	return syncConfigToGormDB(db, config.Models.ModelGroups, logger)
}

func syncConfigToGormDB(db *gorm.DB, groups []models.ModelGroup, logger *zap.Logger) error {
	if db == nil {
		return errors.New("database is nil")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var existingGroups []modelGroupRecord
		if err := tx.Table("janus_model_group").Find(&existingGroups).Error; err != nil {
			return err
		}

		var existingEndpoints []modelEndpointRecord
		if err := tx.Table("janus_model_endpoint").Find(&existingEndpoints).Error; err != nil {
			return err
		}

		plan, err := buildConfigSyncPlan(groups, existingGroups, existingEndpoints)
		if err != nil {
			return err
		}

		groupIDs, err := applyModelGroupPlan(tx, plan.Groups, existingGroups)
		if err != nil {
			return err
		}
		if err := applyModelEndpointPlan(tx, plan.Endpoints, existingEndpoints, groupIDs); err != nil {
			return err
		}
		if err := disableStaleModelConfig(tx, plan); err != nil {
			return err
		}

		if logger != nil {
			logger.Info("Synced model config from YAML to database",
				zap.Int("groups", len(plan.Groups)),
				zap.Int("endpoints", len(plan.Endpoints)),
				zap.Int("disabled_groups", len(plan.DisableGroups)),
				zap.Int("disabled_endpoints", len(plan.DisableEndpoints)),
			)
		}
		return nil
	})
}

func buildConfigSyncPlan(groups []models.ModelGroup, existingGroups []modelGroupRecord, existingEndpoints []modelEndpointRecord) (configSyncPlan, error) {
	plan := configSyncPlan{
		Groups:    make([]modelGroupRecord, 0, len(groups)),
		Endpoints: make([]plannedModelEndpoint, 0),
	}

	desiredGroups := make(map[string]struct{}, len(groups))
	desiredEndpoints := make(map[endpointPlanKey]struct{})
	existingGroupNamesByID := make(map[int64]string, len(existingGroups))
	for _, group := range existingGroups {
		existingGroupNamesByID[group.GroupID] = group.GroupName
	}

	for _, group := range groups {
		groupName := strings.TrimSpace(group.Name)
		if groupName == "" {
			return configSyncPlan{}, errors.New("model group name is empty")
		}
		if _, exists := desiredGroups[groupName]; exists {
			return configSyncPlan{}, fmt.Errorf("duplicate model group in config: %s", groupName)
		}
		desiredGroups[groupName] = struct{}{}

		requestDefaults, err := marshalRequestDefaults(group.RequestDefaults)
		if err != nil {
			return configSyncPlan{}, fmt.Errorf("marshal request_defaults for group %s: %w", groupName, err)
		}

		plan.Groups = append(plan.Groups, modelGroupRecord{
			GroupName:          groupName,
			Strategy:           normalizeStrategy(group.Strategy),
			CostPerInputToken:  group.CostPerInputToken,
			CostPerOutputToken: group.CostPerOutputToken,
			RequestDefaults:    requestDefaults,
			Enabled:            true,
		})

		for _, endpoint := range group.Models {
			endpointName := strings.TrimSpace(endpoint.Name)
			if endpointName == "" {
				return configSyncPlan{}, fmt.Errorf("model endpoint name is empty in group %s", groupName)
			}
			key := endpointPlanKey{GroupName: groupName, EndpointName: endpointName}
			if _, exists := desiredEndpoints[key]; exists {
				return configSyncPlan{}, fmt.Errorf("duplicate endpoint in config: group=%s endpoint=%s", groupName, endpointName)
			}
			desiredEndpoints[key] = struct{}{}

			plan.Endpoints = append(plan.Endpoints, plannedModelEndpoint{
				GroupName: groupName,
				modelEndpointRecord: modelEndpointRecord{
					EndpointName:      endpointName,
					ProviderType:      strings.TrimSpace(endpoint.Type),
					UpstreamModelName: endpointName,
					BaseURL:           strings.TrimSpace(endpoint.BaseURL),
					APIKeySecretRef:   strings.TrimSpace(endpoint.APIKeySecretRef),
					Weight:            normalizePositive(endpoint.Weight, defaultEndpointWeight),
					TimeoutSeconds:    normalizePositive(endpoint.TimeoutSeconds, defaultEndpointTimeoutSeconds),
					RetryTimes:        normalizeNonNegative(endpoint.RetryTimes, defaultEndpointRetryTimes),
					SkipTLSVerify:     endpoint.SkipTLSVerify,
					Enabled:           true,
				},
			})
		}
	}

	for _, group := range existingGroups {
		if _, ok := desiredGroups[group.GroupName]; !ok && group.Enabled {
			plan.DisableGroups = append(plan.DisableGroups, group)
		}
	}
	for _, endpoint := range existingEndpoints {
		groupName := existingGroupNamesByID[endpoint.GroupID]
		key := endpointPlanKey{GroupName: groupName, EndpointName: endpoint.EndpointName}
		if _, ok := desiredEndpoints[key]; !ok && endpoint.Enabled {
			plan.DisableEndpoints = append(plan.DisableEndpoints, endpoint)
		}
	}

	return plan, nil
}

func applyModelGroupPlan(tx *gorm.DB, desired []modelGroupRecord, existing []modelGroupRecord) (map[string]int64, error) {
	groupIDs := make(map[string]int64, len(desired))
	existingByName := make(map[string]modelGroupRecord, len(existing))
	for _, group := range existing {
		existingByName[group.GroupName] = group
	}

	for _, group := range desired {
		updates := map[string]interface{}{
			"strategy":              group.Strategy,
			"cost_per_input_token":  group.CostPerInputToken,
			"cost_per_output_token": group.CostPerOutputToken,
			"request_defaults":      gorm.Expr("?::jsonb", string(group.RequestDefaults)),
			"enabled":               true,
		}
		if existingGroup, ok := existingByName[group.GroupName]; ok {
			if err := tx.Table("janus_model_group").
				Where("group_id = ?", existingGroup.GroupID).
				Updates(updates).Error; err != nil {
				return nil, err
			}
			groupIDs[group.GroupName] = existingGroup.GroupID
			continue
		}

		var newGroupID int64
		if err := tx.Raw(`
			INSERT INTO janus_model_group (
				group_name,
				strategy,
				cost_per_input_token,
				cost_per_output_token,
				request_defaults,
				enabled
			)
			VALUES (?, ?, ?, ?, ?::jsonb, TRUE)
			RETURNING group_id
		`, group.GroupName, group.Strategy, group.CostPerInputToken, group.CostPerOutputToken, string(group.RequestDefaults)).
			Scan(&newGroupID).Error; err != nil {
			return nil, err
		}
		groupIDs[group.GroupName] = newGroupID
	}
	return groupIDs, nil
}

func applyModelEndpointPlan(tx *gorm.DB, desired []plannedModelEndpoint, existing []modelEndpointRecord, groupIDs map[string]int64) error {
	existingByKey := make(map[endpointDBKey]modelEndpointRecord, len(existing))
	for _, endpoint := range existing {
		existingByKey[endpointDBKey{GroupID: endpoint.GroupID, EndpointName: endpoint.EndpointName}] = endpoint
	}

	for _, endpoint := range desired {
		groupID, ok := groupIDs[endpoint.GroupName]
		if !ok {
			return fmt.Errorf("group id not found for endpoint sync: %s", endpoint.GroupName)
		}
		endpoint.GroupID = groupID
		updates := map[string]interface{}{
			"provider_type":       endpoint.ProviderType,
			"upstream_model_name": endpoint.UpstreamModelName,
			"base_url":            endpoint.BaseURL,
			"api_key_secret_ref":  endpoint.APIKeySecretRef,
			"weight":              endpoint.Weight,
			"timeout_seconds":     endpoint.TimeoutSeconds,
			"retry_times":         endpoint.RetryTimes,
			"skip_tls_verify":     endpoint.SkipTLSVerify,
			"enabled":             true,
		}

		if existingEndpoint, ok := existingByKey[endpointDBKey{GroupID: groupID, EndpointName: endpoint.EndpointName}]; ok {
			if err := tx.Table("janus_model_endpoint").
				Where("endpoint_id = ?", existingEndpoint.EndpointID).
				Updates(updates).Error; err != nil {
				return err
			}
			continue
		}

		newEndpoint := endpoint.modelEndpointRecord
		newEndpoint.GroupID = groupID
		if err := tx.Table("janus_model_endpoint").
			Omit("endpoint_id", "create_time", "update_time").
			Create(&newEndpoint).Error; err != nil {
			return err
		}
	}
	return nil
}

type endpointDBKey struct {
	GroupID      int64
	EndpointName string
}

func disableStaleModelConfig(tx *gorm.DB, plan configSyncPlan) error {
	for _, endpoint := range plan.DisableEndpoints {
		if err := tx.Table("janus_model_endpoint").
			Where("endpoint_id = ?", endpoint.EndpointID).
			Update("enabled", false).Error; err != nil {
			return err
		}
	}
	for _, group := range plan.DisableGroups {
		if err := tx.Table("janus_model_group").
			Where("group_id = ?", group.GroupID).
			Update("enabled", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func marshalRequestDefaults(defaults map[string]interface{}) ([]byte, error) {
	if defaults == nil {
		return []byte("{}"), nil
	}
	if len(defaults) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(defaults)
}

func normalizeStrategy(strategy string) string {
	return balancer.NormalizeStrategy(strategy)
}

func normalizePositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func normalizeNonNegative(value int, fallback int) int {
	if value >= 0 {
		return value
	}
	return fallback
}
