package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseOpenAITTFTRuntimeSettingsUsesDatabaseOverridesAndSafeFallbacks(t *testing.T) {
	fallback := openAITTFTRuntimeSettings{
		Enabled:                true,
		BaseP90Ms:              10_000,
		CacheProtectionEnabled: true,
		MinContextTokens:       100_000,
		MinHitRatePercent:      80,
		ElasticP90CapMs:        30_000,
	}

	got := parseOpenAITTFTRuntimeSettings(map[string]string{
		SettingKeyOpenAITTFTOptimizerEnabled:          "false",
		SettingKeyOpenAITTFTBaseP90Seconds:            "12",
		SettingKeyOpenAITTFTCacheProtectionEnabled:    "false",
		SettingKeyOpenAITTFTCacheMinContextTokens:     "150000",
		SettingKeyOpenAITTFTCacheMinHitRatePercent:    "85",
		SettingKeyOpenAITTFTCacheElasticP90CapSeconds: "40",
	}, fallback)

	require.False(t, got.Enabled)
	require.Equal(t, 12_000, got.BaseP90Ms)
	require.False(t, got.CacheProtectionEnabled)
	require.Equal(t, 150_000, got.MinContextTokens)
	require.Equal(t, 85, got.MinHitRatePercent)
	require.Equal(t, 40_000, got.ElasticP90CapMs)

	invalid := parseOpenAITTFTRuntimeSettings(map[string]string{
		SettingKeyOpenAITTFTBaseP90Seconds:            "0",
		SettingKeyOpenAITTFTCacheMinContextTokens:     "999",
		SettingKeyOpenAITTFTCacheMinHitRatePercent:    "101",
		SettingKeyOpenAITTFTCacheElasticP90CapSeconds: "5",
	}, fallback)
	require.Equal(t, fallback, invalid)
}

func TestOpenAITTFTRuntimeDefaultsFollowLegacyGatewayConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAITTFTOptimizerEnabled = false
	cfg.Gateway.OpenAITTFTStableThresholdMs = 12_000

	got := defaultOpenAITTFTRuntimeSettings(cfg)
	require.False(t, got.Enabled)
	require.Equal(t, 12_000, got.BaseP90Ms)
	require.True(t, got.CacheProtectionEnabled)
	require.Equal(t, 100_000, got.MinContextTokens)
	require.Equal(t, 80, got.MinHitRatePercent)
	require.Equal(t, 30_000, got.ElasticP90CapMs)
}

func TestNormalizeOpenAITTFTSystemSettingsValidatesBoundsAndCap(t *testing.T) {
	valid := &SystemSettings{
		OpenAITTFTBaseP90Seconds:            10,
		OpenAITTFTCacheMinContextTokens:     100_000,
		OpenAITTFTCacheMinHitRatePercent:    80,
		OpenAITTFTCacheElasticP90CapSeconds: 30,
	}
	require.NoError(t, normalizeOpenAITTFTSystemSettings(valid))

	for _, tc := range []struct {
		name   string
		mutate func(*SystemSettings)
	}{
		{name: "base below range", mutate: func(s *SystemSettings) { s.OpenAITTFTBaseP90Seconds = 0 }},
		{name: "context below range", mutate: func(s *SystemSettings) { s.OpenAITTFTCacheMinContextTokens = 999 }},
		{name: "hit rate above range", mutate: func(s *SystemSettings) { s.OpenAITTFTCacheMinHitRatePercent = 101 }},
		{name: "cap below base", mutate: func(s *SystemSettings) { s.OpenAITTFTCacheElasticP90CapSeconds = 9 }},
		{name: "cap above range", mutate: func(s *SystemSettings) { s.OpenAITTFTCacheElasticP90CapSeconds = 61 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := *valid
			tc.mutate(&settings)
			require.Error(t, normalizeOpenAITTFTSystemSettings(&settings))
		})
	}
}
