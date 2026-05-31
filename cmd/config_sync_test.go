package main

import (
	"encoding/json"
	"testing"

	"github.com/Uuq114/JanusLLM/internal/models"
)

func TestBuildConfigSyncPlanUsesYAMLAsDesiredState(t *testing.T) {
	groups := []models.ModelGroup{
		{
			Name:               "chat",
			Strategy:           "weighted",
			CostPerInputToken:  0.001,
			CostPerOutputToken: 0.002,
			RequestDefaults: map[string]interface{}{
				"temperature": 0.7,
				"stream":      false,
			},
			Models: []models.ModelConfig{
				{
					Name:           "fast",
					Type:           "openai",
					BaseURL:        "https://fast.example/v1",
					APIKey:         "plaintext-key-must-not-sync",
					Weight:         10,
					TimeoutSeconds: 30,
					RetryTimes:     2,
					SkipTLSVerify:  true,
				},
				{
					Name:            "safe",
					Type:            "anthropic",
					BaseURL:         "https://safe.example",
					APIKeySecretRef: "secret/provider/safe",
				},
			},
		},
	}
	existingGroups := []modelGroupRecord{
		{GroupID: 1, GroupName: "chat", Enabled: true},
		{GroupID: 2, GroupName: "removed", Enabled: true},
		{GroupID: 3, GroupName: "already-disabled", Enabled: false},
	}
	existingEndpoints := []modelEndpointRecord{
		{EndpointID: 10, GroupID: 1, EndpointName: "fast", Enabled: true},
		{EndpointID: 11, GroupID: 1, EndpointName: "stale", Enabled: true},
		{EndpointID: 12, GroupID: 2, EndpointName: "removed-upstream", Enabled: true},
		{EndpointID: 13, GroupID: 1, EndpointName: "already-disabled", Enabled: false},
	}

	plan, err := buildConfigSyncPlan(groups, existingGroups, existingEndpoints)
	if err != nil {
		t.Fatalf("buildConfigSyncPlan returned error: %v", err)
	}

	if len(plan.Groups) != 1 {
		t.Fatalf("expected 1 desired group, got %d", len(plan.Groups))
	}
	group := plan.Groups[0]
	if group.GroupName != "chat" || group.Strategy != "weighted" || !group.Enabled {
		t.Fatalf("unexpected desired group: %+v", group)
	}
	var requestDefaults map[string]interface{}
	if err := json.Unmarshal(group.RequestDefaults, &requestDefaults); err != nil {
		t.Fatalf("request_defaults is not JSON: %v", err)
	}
	if requestDefaults["temperature"] != 0.7 || requestDefaults["stream"] != false {
		t.Fatalf("unexpected request_defaults: %v", requestDefaults)
	}

	if len(plan.Endpoints) != 2 {
		t.Fatalf("expected 2 desired endpoints, got %d", len(plan.Endpoints))
	}
	fast := plan.Endpoints[0]
	if fast.GroupName != "chat" || fast.EndpointName != "fast" || fast.UpstreamModelName != "fast" {
		t.Fatalf("unexpected fast endpoint identity: %+v", fast)
	}
	if fast.APIKeySecretRef != "" {
		t.Fatalf("plaintext api_key should not be synced, got secret ref %q", fast.APIKeySecretRef)
	}
	if fast.Weight != 10 || fast.TimeoutSeconds != 30 || fast.RetryTimes != 2 || !fast.SkipTLSVerify {
		t.Fatalf("unexpected fast endpoint settings: %+v", fast)
	}
	safe := plan.Endpoints[1]
	if safe.APIKeySecretRef != "secret/provider/safe" {
		t.Fatalf("expected api_key_secret_ref to sync, got %q", safe.APIKeySecretRef)
	}
	if safe.Weight != defaultEndpointWeight || safe.TimeoutSeconds != defaultEndpointTimeoutSeconds {
		t.Fatalf("expected DB-compatible defaults on omitted values, got weight=%d timeout=%d", safe.Weight, safe.TimeoutSeconds)
	}

	if len(plan.DisableGroups) != 1 || plan.DisableGroups[0].GroupName != "removed" {
		t.Fatalf("expected only removed group to be disabled, got %+v", plan.DisableGroups)
	}
	if len(plan.DisableEndpoints) != 2 {
		t.Fatalf("expected 2 stale endpoints to be disabled, got %+v", plan.DisableEndpoints)
	}
	disabled := map[int64]bool{}
	for _, endpoint := range plan.DisableEndpoints {
		disabled[endpoint.EndpointID] = true
	}
	if !disabled[11] || !disabled[12] || disabled[13] {
		t.Fatalf("unexpected disabled endpoints: %+v", plan.DisableEndpoints)
	}
}

func TestBuildConfigSyncPlanRejectsDuplicateGroups(t *testing.T) {
	_, err := buildConfigSyncPlan([]models.ModelGroup{
		{Name: "chat"},
		{Name: "chat"},
	}, nil, nil)
	if err == nil {
		t.Fatal("expected duplicate group error")
	}
}

func TestBuildConfigSyncPlanRejectsDuplicateEndpoints(t *testing.T) {
	_, err := buildConfigSyncPlan([]models.ModelGroup{
		{
			Name: "chat",
			Models: []models.ModelConfig{
				{Name: "upstream"},
				{Name: "upstream"},
			},
		},
	}, nil, nil)
	if err == nil {
		t.Fatal("expected duplicate endpoint error")
	}
}

func TestBuildConfigSyncPlanHandlesEmptyRequestDefaults(t *testing.T) {
	plan, err := buildConfigSyncPlan([]models.ModelGroup{
		{Name: "chat"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("buildConfigSyncPlan returned error: %v", err)
	}
	if got := string(plan.Groups[0].RequestDefaults); got != "{}" {
		t.Fatalf("expected empty request_defaults JSON object, got %s", got)
	}
}

func TestBuildConfigSyncPlanNormalizesStrategyAliases(t *testing.T) {
	plan, err := buildConfigSyncPlan([]models.ModelGroup{
		{Name: "chat", Strategy: "round_robin"},
		{Name: "sticky", Strategy: "sticky-hash"},
		{Name: "fast", Strategy: "latency-based"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("buildConfigSyncPlan returned error: %v", err)
	}

	got := map[string]string{}
	for _, group := range plan.Groups {
		got[group.GroupName] = group.Strategy
	}
	if got["chat"] != "round-robin" || got["sticky"] != "client-sticky" || got["fast"] != "latency" {
		t.Fatalf("unexpected normalized strategies: %+v", got)
	}
}
