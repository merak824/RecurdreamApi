package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RiskAction 风控规则命中后的处置动作。
// 规则通过返回不同的 Action 决定"只留痕"还是"直接处置"，
// 新增规则时无需改动处置链路。
type RiskAction string

const (
	// RiskActionFlag 仅记录审计事件，不影响账号（用于观察期 / 低置信度信号）。
	RiskActionFlag RiskAction = "flag"
	// RiskActionBan 封禁账号（status=disabled），立即生效且不可自助恢复。
	RiskActionBan RiskAction = "ban"
	// RiskActionBlock 拒绝本次请求（注册类规则在创建账号前拦截，返回 429）。
	RiskActionBlock RiskAction = "block"
)

// isValidRiskAction 判断动作名是否合法。
func isValidRiskAction(action RiskAction) bool {
	switch action {
	case RiskActionFlag, RiskActionBan, RiskActionBlock:
		return true
	default:
		return false
	}
}

// AccountRiskRegisterProbeConfig 注册即探测规则配置。
// 新账号在窗口期内出现"非浏览器客户端直接调用 API"的特征即判定为注册机。
type AccountRiskRegisterProbeConfig struct {
	Enabled bool `json:"enabled"`
	// WindowMinutes 注册后多少分钟内仍按"新账号"对待。
	WindowMinutes int `json:"window_minutes"`
	// Action 命中后的处置动作（ban / flag）。
	Action RiskAction `json:"action"`
}

// AccountRiskMultiAccountConfig 同 IP 批量注册规则配置。
type AccountRiskMultiAccountConfig struct {
	Enabled bool `json:"enabled"`
	// MaxAccountsPerIP 窗口期内同一 IP 允许成功注册的账号数，超出即拦截。
	MaxAccountsPerIP int `json:"max_accounts_per_ip"`
	// WindowHours 统计窗口（小时）。
	WindowHours int `json:"window_hours"`
	// Action 命中后的处置动作（block / ban / flag）。
	Action RiskAction `json:"action"`
}

// AccountRiskSessionAbuseConfig 会话指纹异常规则配置。
// 同一账号反复出现 IP+UA 指纹漂移，通常意味着 token 被复制到其他环境使用。
type AccountRiskSessionAbuseConfig struct {
	Enabled bool `json:"enabled"`
	// MaxMismatches 窗口期内允许的指纹不匹配次数，超出即处置。
	MaxMismatches int `json:"max_mismatches"`
	// WindowHours 统计窗口（小时）。
	WindowHours int `json:"window_hours"`
	// Action 命中后的处置动作（flag / ban）。
	Action RiskAction `json:"action"`
}

// AccountRiskLoginGuardConfig 登录暴力破解防护配置。
// 该规则不封禁账号，只做临时锁定（按 IP 与按账号两个维度）。
type AccountRiskLoginGuardConfig struct {
	Enabled bool `json:"enabled"`
	// MaxIPFailures 窗口期内同一 IP 允许的登录失败次数。
	MaxIPFailures int `json:"max_ip_failures"`
	// IPWindowMinutes IP 维度的失败计数窗口（分钟）。
	IPWindowMinutes int `json:"ip_window_minutes"`
	// MaxAccountFailures 窗口期内同一账号允许的登录失败次数。
	MaxAccountFailures int `json:"max_account_failures"`
	// AccountWindowMinutes 账号维度的失败计数窗口（分钟）。
	AccountWindowMinutes int `json:"account_window_minutes"`
	// LockMinutes 触发后锁定多久（分钟）。
	LockMinutes int `json:"lock_minutes"`
}

// AccountRiskConfig 账号异常检测与自动封禁总配置。
// 整体存储在 system_settings.account_risk_config（JSON），修改后 ~30s 内生效。
//
// 扩展新规则时：在此追加一个子配置结构体 + 一个 rule 实现，
// 规则通过 Enabled 开关独立启停，无需改动其他规则。
type AccountRiskConfig struct {
	// Enabled 总开关：关闭后所有规则不再评估（登录防护也一并失效）。
	Enabled bool `json:"enabled"`

	RegisterProbe AccountRiskRegisterProbeConfig `json:"register_probe"`
	MultiAccount  AccountRiskMultiAccountConfig  `json:"multi_account"`
	SessionAbuse  AccountRiskSessionAbuseConfig  `json:"session_abuse"`
	LoginGuard    AccountRiskLoginGuardConfig    `json:"login_guard"`

	// AllowlistedUserIDs 免检用户（管理员始终免检，无需在此重复配置）。
	AllowlistedUserIDs []int64 `json:"allowlisted_user_ids"`
}

// DefaultAccountRiskConfig 返回内置默认配置。
// 默认值取"宁可放过、不可错杀"的保守口径：
// 只有明确的自动化客户端特征才直接封禁，指纹漂移只留痕。
func DefaultAccountRiskConfig() *AccountRiskConfig {
	return &AccountRiskConfig{
		Enabled: true,
		RegisterProbe: AccountRiskRegisterProbeConfig{
			Enabled:       true,
			WindowMinutes: 120,
			Action:        RiskActionBan,
		},
		MultiAccount: AccountRiskMultiAccountConfig{
			Enabled:          true,
			MaxAccountsPerIP: 3,
			WindowHours:      24,
			Action:           RiskActionBlock,
		},
		SessionAbuse: AccountRiskSessionAbuseConfig{
			Enabled:       true,
			MaxMismatches: 12,
			WindowHours:   24,
			Action:        RiskActionFlag,
		},
		LoginGuard: AccountRiskLoginGuardConfig{
			Enabled:              true,
			MaxIPFailures:        30,
			IPWindowMinutes:      10,
			MaxAccountFailures:   8,
			AccountWindowMinutes: 15,
			LockMinutes:          15,
		},
	}
}

// normalize 归一化配置：缺失字段补默认值，越界值收敛到合法区间。
// 返回错误仅用于提示明显的配置错误（未知动作名），调用方决定是否忽略。
func (c *AccountRiskConfig) normalize() error {
	if c == nil {
		return nil
	}
	def := DefaultAccountRiskConfig()

	if c.RegisterProbe.WindowMinutes <= 0 {
		c.RegisterProbe.WindowMinutes = def.RegisterProbe.WindowMinutes
	}
	if c.RegisterProbe.WindowMinutes > 24*60 {
		c.RegisterProbe.WindowMinutes = 24 * 60
	}
	if !isValidRiskAction(c.RegisterProbe.Action) {
		c.RegisterProbe.Action = def.RegisterProbe.Action
	}

	if c.MultiAccount.MaxAccountsPerIP <= 0 {
		c.MultiAccount.MaxAccountsPerIP = def.MultiAccount.MaxAccountsPerIP
	}
	if c.MultiAccount.MaxAccountsPerIP > 1000 {
		c.MultiAccount.MaxAccountsPerIP = 1000
	}
	if c.MultiAccount.WindowHours <= 0 {
		c.MultiAccount.WindowHours = def.MultiAccount.WindowHours
	}
	if c.MultiAccount.WindowHours > 24*30 {
		c.MultiAccount.WindowHours = 24 * 30
	}
	if !isValidRiskAction(c.MultiAccount.Action) {
		c.MultiAccount.Action = def.MultiAccount.Action
	}

	if c.SessionAbuse.MaxMismatches <= 0 {
		c.SessionAbuse.MaxMismatches = def.SessionAbuse.MaxMismatches
	}
	if c.SessionAbuse.MaxMismatches > 10000 {
		c.SessionAbuse.MaxMismatches = 10000
	}
	if c.SessionAbuse.WindowHours <= 0 {
		c.SessionAbuse.WindowHours = def.SessionAbuse.WindowHours
	}
	if c.SessionAbuse.WindowHours > 24*30 {
		c.SessionAbuse.WindowHours = 24 * 30
	}
	if !isValidRiskAction(c.SessionAbuse.Action) {
		c.SessionAbuse.Action = def.SessionAbuse.Action
	}

	if c.LoginGuard.MaxIPFailures <= 0 {
		c.LoginGuard.MaxIPFailures = def.LoginGuard.MaxIPFailures
	}
	if c.LoginGuard.IPWindowMinutes <= 0 {
		c.LoginGuard.IPWindowMinutes = def.LoginGuard.IPWindowMinutes
	}
	if c.LoginGuard.MaxAccountFailures <= 0 {
		c.LoginGuard.MaxAccountFailures = def.LoginGuard.MaxAccountFailures
	}
	if c.LoginGuard.AccountWindowMinutes <= 0 {
		c.LoginGuard.AccountWindowMinutes = def.LoginGuard.AccountWindowMinutes
	}
	if c.LoginGuard.LockMinutes <= 0 {
		c.LoginGuard.LockMinutes = def.LoginGuard.LockMinutes
	}
	if c.LoginGuard.LockMinutes > 24*60 {
		c.LoginGuard.LockMinutes = 24 * 60
	}

	// 去重并丢弃非法 ID，避免配置膨胀（同时保持顺序稳定，便于比对）。
	if len(c.AllowlistedUserIDs) > 0 {
		seen := make(map[int64]struct{}, len(c.AllowlistedUserIDs))
		out := make([]int64, 0, len(c.AllowlistedUserIDs))
		for _, id := range c.AllowlistedUserIDs {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
		c.AllowlistedUserIDs = out
	}
	return nil
}

// cloneAccountRiskConfig 返回一份深拷贝，避免调用方修改缓存中的实例。
func cloneAccountRiskConfig(cfg *AccountRiskConfig) *AccountRiskConfig {
	if cfg == nil {
		return DefaultAccountRiskConfig()
	}
	out := *cfg
	if len(cfg.AllowlistedUserIDs) > 0 {
		out.AllowlistedUserIDs = append([]int64(nil), cfg.AllowlistedUserIDs...)
	}
	return &out
}

// IsUserAllowlisted 判断用户是否在免检名单内。
func (c *AccountRiskConfig) IsUserAllowlisted(userID int64) bool {
	if c == nil || userID <= 0 {
		return false
	}
	for _, id := range c.AllowlistedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

const accountRiskConfigCacheTTL = 30 * time.Second
const accountRiskConfigErrorTTL = 5 * time.Second

// cachedAccountRiskConfig 配置的进程内缓存条目。
type cachedAccountRiskConfig struct {
	cfg        *AccountRiskConfig
	expiresAt  int64 // unix nano
	staleValue bool  // 本次是从 DB 读失败后回退的旧值/默认值
}

// GetAccountRiskConfig 返回账号风控配置（进程内缓存 ~30s）。
// 注册、登录、鉴权都会有调用，禁止在热路径上直接读 DB。
// 任何异常（设置缺失 / JSON 损坏 / DB 故障）都返回默认配置，绝不返回 nil。
func (s *SettingService) GetAccountRiskConfig(ctx context.Context) *AccountRiskConfig {
	if s == nil || s.settingRepo == nil {
		return DefaultAccountRiskConfig()
	}
	if cached, ok := s.accountRiskConfigCache.Load().(*cachedAccountRiskConfig); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.cfg
		}
	}

	s.accountRiskConfigMu.Lock()
	defer s.accountRiskConfigMu.Unlock()
	if cached, ok := s.accountRiskConfigCache.Load().(*cachedAccountRiskConfig); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.cfg
		}
	}

	previous, _ := s.accountRiskConfigCache.Load().(*cachedAccountRiskConfig)

	value, err := s.settingRepo.GetValue(ctx, SettingKeyAccountRiskConfig)
	if err != nil || strings.TrimSpace(value) == "" {
		// 读取失败或从未配置：沿用上一份有效配置，没有则用默认值。
		// 缓存 TTL 调短，配置恢复后能尽快生效。
		entry := &cachedAccountRiskConfig{
			expiresAt:  time.Now().Add(accountRiskConfigErrorTTL).UnixNano(),
			staleValue: true,
		}
		if previous != nil && previous.cfg != nil {
			entry.cfg = previous.cfg
		} else {
			entry.cfg = DefaultAccountRiskConfig()
		}
		s.accountRiskConfigCache.Store(entry)
		return entry.cfg
	}

	cfg := DefaultAccountRiskConfig()
	if err := json.Unmarshal([]byte(value), cfg); err != nil {
		// JSON 损坏：保留默认值，避免整站风控失效或误封。
		cfg = DefaultAccountRiskConfig()
	}
	_ = cfg.normalize()

	s.accountRiskConfigCache.Store(&cachedAccountRiskConfig{
		cfg:       cfg,
		expiresAt: time.Now().Add(accountRiskConfigCacheTTL).UnixNano(),
	})
	return cfg
}

// SetAccountRiskConfig 保存账号风控配置（幂等，会做归一化）。
func (s *SettingService) SetAccountRiskConfig(ctx context.Context, cfg *AccountRiskConfig) error {
	if s == nil || s.settingRepo == nil {
		return fmt.Errorf("setting service is not available")
	}
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	next := cloneAccountRiskConfig(cfg)
	if err := next.normalize(); err != nil {
		return err
	}
	data, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("marshal account risk config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyAccountRiskConfig, string(data)); err != nil {
		return err
	}
	// 立即刷新缓存，避免"保存后要等 30s 才生效"的困惑。
	s.accountRiskConfigCache.Store(&cachedAccountRiskConfig{
		cfg:       next,
		expiresAt: time.Now().Add(accountRiskConfigCacheTTL).UnixNano(),
	})
	return nil
}
