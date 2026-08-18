package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/singleflight"
)

const (
	openAITTFTRuntimeSettingsTTL       = 60 * time.Second
	openAITTFTRuntimeSettingsRetryTTL  = 5 * time.Second
	openAITTFTRuntimeSettingsDBTimeout = 2 * time.Second
	defaultOpenAITTFTMinContextTokens  = 100_000
	defaultOpenAITTFTMinHitRatePercent = 80
	defaultOpenAITTFTElasticP90CapMs   = 30_000
)

type openAITTFTRuntimeSettings struct {
	Enabled                bool
	BaseP90Ms              int
	CacheProtectionEnabled bool
	MinContextTokens       int
	MinHitRatePercent      int
	ElasticP90CapMs        int
}

type cachedOpenAITTFTRuntimeSettings struct {
	settings  openAITTFTRuntimeSettings
	expiresAt int64
}

var openAITTFTRuntimeSettingsCache atomic.Value // *cachedOpenAITTFTRuntimeSettings
var openAITTFTRuntimeSettingsSF singleflight.Group
var openAITTFTRuntimeSettingsRefreshRunning atomic.Bool
var openAITTFTRuntimeSettingsCacheMu sync.Mutex

func defaultOpenAITTFTRuntimeSettings(cfg *config.Config) openAITTFTRuntimeSettings {
	settings := openAITTFTRuntimeSettings{
		Enabled:                true,
		BaseP90Ms:              defaultOpenAITTFTStableThresholdMs,
		CacheProtectionEnabled: true,
		MinContextTokens:       defaultOpenAITTFTMinContextTokens,
		MinHitRatePercent:      defaultOpenAITTFTMinHitRatePercent,
		ElasticP90CapMs:        defaultOpenAITTFTElasticP90CapMs,
	}
	if cfg == nil {
		return settings
	}
	settings.Enabled = cfg.Gateway.OpenAITTFTOptimizerEnabled
	if cfg.Gateway.OpenAITTFTStableThresholdMs >= 1_000 && cfg.Gateway.OpenAITTFTStableThresholdMs <= 30_000 {
		settings.BaseP90Ms = cfg.Gateway.OpenAITTFTStableThresholdMs
	}
	if settings.ElasticP90CapMs < settings.BaseP90Ms {
		settings.ElasticP90CapMs = settings.BaseP90Ms
	}
	return settings
}

func parseOpenAITTFTRuntimeSettings(values map[string]string, fallback openAITTFTRuntimeSettings) openAITTFTRuntimeSettings {
	settings := fallback
	if raw, ok := values[SettingKeyOpenAITTFTOptimizerEnabled]; ok {
		if value, valid := parseStrictBool(raw); valid {
			settings.Enabled = value
		}
	}
	if raw, ok := values[SettingKeyOpenAITTFTBaseP90Seconds]; ok {
		if value, valid := parseBoundedInt(raw, 1, 30); valid {
			settings.BaseP90Ms = value * 1_000
		}
	}
	if raw, ok := values[SettingKeyOpenAITTFTCacheProtectionEnabled]; ok {
		if value, valid := parseStrictBool(raw); valid {
			settings.CacheProtectionEnabled = value
		}
	}
	if raw, ok := values[SettingKeyOpenAITTFTCacheMinContextTokens]; ok {
		if value, valid := parseBoundedInt(raw, 1_000, 10_000_000); valid {
			settings.MinContextTokens = value
		}
	}
	if raw, ok := values[SettingKeyOpenAITTFTCacheMinHitRatePercent]; ok {
		if value, valid := parseBoundedInt(raw, 1, 100); valid {
			settings.MinHitRatePercent = value
		}
	}
	if raw, ok := values[SettingKeyOpenAITTFTCacheElasticP90CapSeconds]; ok {
		if value, valid := parseBoundedInt(raw, 1, 60); valid && value*1_000 >= settings.BaseP90Ms {
			settings.ElasticP90CapMs = value * 1_000
		}
	}
	if settings.ElasticP90CapMs < settings.BaseP90Ms {
		settings.ElasticP90CapMs = max(settings.BaseP90Ms, fallback.ElasticP90CapMs)
	}
	return settings
}

func parseStrictBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func parseBoundedInt(raw string, minValue, maxValue int) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < minValue || value > maxValue {
		return 0, false
	}
	return value, true
}

func normalizeOpenAITTFTSystemSettings(settings *SystemSettings) error {
	if settings == nil {
		return infraerrors.BadRequest("INVALID_OPENAI_TTFT_SETTINGS", "OpenAI first-token settings are required")
	}
	if settings.OpenAITTFTBaseP90Seconds == 0 && settings.OpenAITTFTCacheMinContextTokens == 0 && settings.OpenAITTFTCacheMinHitRatePercent == 0 && settings.OpenAITTFTCacheElasticP90CapSeconds == 0 {
		settings.OpenAITTFTBaseP90Seconds = defaultOpenAITTFTStableThresholdMs / 1_000
		settings.OpenAITTFTCacheMinContextTokens = defaultOpenAITTFTMinContextTokens
		settings.OpenAITTFTCacheMinHitRatePercent = defaultOpenAITTFTMinHitRatePercent
		settings.OpenAITTFTCacheElasticP90CapSeconds = defaultOpenAITTFTElasticP90CapMs / 1_000
	}
	if settings.OpenAITTFTBaseP90Seconds < 1 || settings.OpenAITTFTBaseP90Seconds > 30 {
		return infraerrors.BadRequest("INVALID_OPENAI_TTFT_BASE_P90", "OpenAI first-token base P90 must be between 1 and 30 seconds")
	}
	if settings.OpenAITTFTCacheMinContextTokens < 1_000 || settings.OpenAITTFTCacheMinContextTokens > 10_000_000 {
		return infraerrors.BadRequest("INVALID_OPENAI_TTFT_CACHE_CONTEXT", "OpenAI cache context threshold must be between 1000 and 10000000 tokens")
	}
	if settings.OpenAITTFTCacheMinHitRatePercent < 1 || settings.OpenAITTFTCacheMinHitRatePercent > 100 {
		return infraerrors.BadRequest("INVALID_OPENAI_TTFT_CACHE_HIT_RATE", "OpenAI cache hit rate must be between 1 and 100 percent")
	}
	if settings.OpenAITTFTCacheElasticP90CapSeconds < settings.OpenAITTFTBaseP90Seconds || settings.OpenAITTFTCacheElasticP90CapSeconds > 60 {
		return infraerrors.BadRequest("INVALID_OPENAI_TTFT_ELASTIC_P90_CAP", "OpenAI elastic P90 cap must be between the base P90 and 60 seconds")
	}
	return nil
}

func openAITTFTRuntimeSettingKeys() []string {
	return []string{
		SettingKeyOpenAITTFTOptimizerEnabled,
		SettingKeyOpenAITTFTBaseP90Seconds,
		SettingKeyOpenAITTFTCacheProtectionEnabled,
		SettingKeyOpenAITTFTCacheMinContextTokens,
		SettingKeyOpenAITTFTCacheMinHitRatePercent,
		SettingKeyOpenAITTFTCacheElasticP90CapSeconds,
	}
}

func (s *OpenAIGatewayService) openAITTFTRuntimeSettings(ctx context.Context) openAITTFTRuntimeSettings {
	fallback := defaultOpenAITTFTRuntimeSettings(nil)
	if s != nil {
		fallback = defaultOpenAITTFTRuntimeSettings(s.cfg)
	}
	now := time.Now()
	if cached, ok := openAITTFTRuntimeSettingsCache.Load().(*cachedOpenAITTFTRuntimeSettings); ok && cached != nil {
		if now.UnixNano() >= cached.expiresAt {
			s.refreshOpenAITTFTRuntimeSettingsAsync(ctx, cached.settings)
		}
		return cached.settings
	}
	openAITTFTRuntimeSettingsCacheMu.Lock()
	if cached, ok := openAITTFTRuntimeSettingsCache.Load().(*cachedOpenAITTFTRuntimeSettings); ok && cached != nil {
		openAITTFTRuntimeSettingsCacheMu.Unlock()
		return cached.settings
	}
	openAITTFTRuntimeSettingsCache.Store(&cachedOpenAITTFTRuntimeSettings{settings: fallback, expiresAt: now.UnixNano()})
	openAITTFTRuntimeSettingsCacheMu.Unlock()
	s.refreshOpenAITTFTRuntimeSettingsAsync(ctx, fallback)
	return fallback
}

func (s *OpenAIGatewayService) refreshOpenAITTFTRuntimeSettingsAsync(ctx context.Context, stale openAITTFTRuntimeSettings) {
	if s == nil || !openAITTFTRuntimeSettingsRefreshRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer openAITTFTRuntimeSettingsRefreshRunning.Store(false)
		_, _, _ = openAITTFTRuntimeSettingsSF.Do("openai_ttft_runtime_settings", func() (any, error) {
			repo := s.openAIAdvancedSchedulerSettingRepo()
			if repo == nil {
				storeOpenAITTFTRuntimeSettings(stale, openAITTFTRuntimeSettingsRetryTTL)
				return stale, nil
			}
			dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAITTFTRuntimeSettingsDBTimeout)
			defer cancel()
			values, err := repo.GetMultiple(dbCtx, openAITTFTRuntimeSettingKeys())
			if err != nil {
				storeOpenAITTFTRuntimeSettings(stale, openAITTFTRuntimeSettingsRetryTTL)
				slog.Warn("openai_ttft_runtime_settings_load_failed", "error", err)
				return stale, err
			}
			settings := parseOpenAITTFTRuntimeSettings(values, defaultOpenAITTFTRuntimeSettings(s.cfg))
			storeOpenAITTFTRuntimeSettings(settings, openAITTFTRuntimeSettingsTTL)
			return settings, nil
		})
	}()
}

func storeOpenAITTFTRuntimeSettings(settings openAITTFTRuntimeSettings, ttl time.Duration) {
	openAITTFTRuntimeSettingsCache.Store(&cachedOpenAITTFTRuntimeSettings{settings: settings, expiresAt: time.Now().Add(ttl).UnixNano()})
}

func resetOpenAITTFTRuntimeSettingsCacheForTest() {
	openAITTFTRuntimeSettingsCache = atomic.Value{}
	openAITTFTRuntimeSettingsSF = singleflight.Group{}
	openAITTFTRuntimeSettingsRefreshRunning.Store(false)
}
