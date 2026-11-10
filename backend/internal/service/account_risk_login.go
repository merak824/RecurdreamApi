package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrAccountRiskLoginLocked 登录失败次数超限，临时锁定。
// 使用 429 而不是 403：这是"稍后再试"，不是"你没有权限"。
var ErrAccountRiskLoginLocked = infraerrors.TooManyRequests(
	"LOGIN_BLOCKED_RISK",
	"too many failed login attempts, please try again later",
)

// 登录暴力破解防护：按 IP 与按账号双维度计数，超限后短期锁定。
//
// 与其他风控规则的区别：这里**不封禁账号**，只做临时锁定（默认 15 分钟），
// 因为密码撞库的主要受害者恰恰是账号主人，永久封禁会误伤真实用户。
//
// 攻击者在同一 IP 上轮换账号（撞库）由 IP 维度拦截；
// 攻击者分散 IP 打同一个账号（撞库/喷库）由账号维度拦截。

const (
	accountRiskLoginIPKeyPrefix       = "login_ip:"
	accountRiskLoginAccountKeyPrefix  = "login_acct:"
	accountRiskLoginIPLockPrefix      = "login_ip_lock:"
	accountRiskLoginAccountLockPrefix = "login_acct_lock:"
)

// LoginGuardEnabled 报告登录防护是否生效（配置总开关 + 子开关）。
func (s *AccountRiskService) LoginGuardEnabled(ctx context.Context) bool {
	cfg := s.Config(ctx)
	if cfg == nil || !cfg.Enabled || !cfg.LoginGuard.Enabled {
		return false
	}
	return true
}

// LoginPreCheck 登录前置检查：命中锁定则直接拒绝，不再校验密码（省一次 bcrypt）。
func (s *AccountRiskService) LoginPreCheck(ctx context.Context) error {
	if s == nil || s.store == nil || !s.LoginGuardEnabled(ctx) {
		return nil
	}
	ip := riskClientIP(ctx)
	if ip != "" && s.store.HasFlag(ctx, accountRiskKeyPrefix+accountRiskLoginIPLockPrefix+ip) {
		return ErrAccountRiskLoginLocked
	}
	// 账号维度的锁定键依赖邮箱，这里拿不到（此时还没解析请求体之外的上下文），
	// 因此在 LoginFailed 侧统一判断；预检查只覆盖 IP 维度。
	return nil
}

// LoginAccountPreCheck 账号维度前置检查（调用方已知邮箱时使用）。
func (s *AccountRiskService) LoginAccountPreCheck(ctx context.Context, email string) error {
	if s == nil || s.store == nil || !s.LoginGuardEnabled(ctx) {
		return nil
	}
	key := accountRiskLoginAccountKey(email)
	if key == "" {
		return nil
	}
	if s.store.HasFlag(ctx, accountRiskKeyPrefix+accountRiskLoginAccountLockPrefix+key) {
		return ErrAccountRiskLoginLocked
	}
	return nil
}

// LoginFailed 记录一次登录失败并判断是否触发锁定。
// 返回 ErrAccountRiskLoginLocked 表示本次失败已达到阈值，调用方可以据此
// 把错误提升为"锁定"而不是"密码错误"（避免暴露账号是否存在）。
func (s *AccountRiskService) LoginFailed(ctx context.Context, email string) error {
	if s == nil || s.store == nil || !s.LoginGuardEnabled(ctx) {
		return nil
	}
	cfg := s.Config(ctx)
	guard := cfg.LoginGuard
	ip := riskClientIP(ctx)

	locked := ""
	var ipCount, accountCount int64

	if ip != "" {
		window := time.Duration(guard.IPWindowMinutes) * time.Minute
		if count, err := s.store.Incr(ctx, accountRiskKeyPrefix+accountRiskLoginIPKeyPrefix+ip, window); err != nil {
			slog.Warn("account_risk.login_ip_counter_failed", "ip", ip, "error", err)
		} else {
			ipCount = count
			if count == int64(guard.MaxIPFailures) {
				if _, err := s.store.SetFlag(ctx, accountRiskKeyPrefix+accountRiskLoginIPLockPrefix+ip, time.Duration(guard.LockMinutes)*time.Minute); err == nil {
					locked = "ip"
				}
			}
		}
	}

	if accountKey := accountRiskLoginAccountKey(email); accountKey != "" {
		window := time.Duration(guard.AccountWindowMinutes) * time.Minute
		if count, err := s.store.Incr(ctx, accountRiskKeyPrefix+accountRiskLoginAccountKeyPrefix+accountKey, window); err != nil {
			slog.Warn("account_risk.login_account_counter_failed", "error", err)
		} else {
			accountCount = count
			if count == int64(guard.MaxAccountFailures) {
				if _, err := s.store.SetFlag(ctx, accountRiskKeyPrefix+accountRiskLoginAccountLockPrefix+accountKey, time.Duration(guard.LockMinutes)*time.Minute); err == nil {
					if locked == "" {
						locked = "account"
					} else {
						locked = "ip+account"
					}
				}
			}
		}
	}

	if locked == "" {
		// 未达阈值：仅在高频失败时留一条低噪日志，便于事后取证。
		if ipCount == 5 || accountCount == 3 {
			slog.Warn("account_risk.login_failures_rising",
				"ip_failures", ipCount,
				"account_failures", accountCount,
				"ip", ip,
			)
		}
		return nil
	}

	s.recordAudit(ctx, &AccountRiskEvent{
		Type:      "login_failure",
		Email:     email,
		IP:        ip,
		UserAgent: riskClientUserAgent(ctx),
	}, []AccountRiskVerdict{{
		Rule:   "login_bruteforce",
		Action: RiskActionFlag,
		Score:  90,
		Reason: fmt.Sprintf("login failures exceeded threshold (ip=%d/%d, account=%d/%d), locked %dmin",
			ipCount, guard.MaxIPFailures, accountCount, guard.MaxAccountFailures, guard.LockMinutes),
		Signals: map[string]any{
			"locked_dimension":      locked,
			"ip_failures":           ipCount,
			"ip_failure_limit":      guard.MaxIPFailures,
			"account_failures":      accountCount,
			"account_failure_limit": guard.MaxAccountFailures,
			"lock_minutes":          guard.LockMinutes,
		},
	}}, AuditActionAccountRiskLoginLocked, 429)

	slog.Warn("account_risk.login_locked",
		"dimension", locked,
		"ip", ip,
		"ip_failures", ipCount,
		"account_failures", accountCount,
		"lock_minutes", guard.LockMinutes,
	)
	return nil
}

// LoginSucceeded 登录成功后清除账号维度的失败计数（IP 计数保留，仍反映攻击面）。
func (s *AccountRiskService) LoginSucceeded(ctx context.Context, email string) {
	if s == nil || s.store == nil || !s.LoginGuardEnabled(ctx) {
		return
	}
	key := accountRiskLoginAccountKey(email)
	if key == "" {
		return
	}
	s.store.Reset(ctx, accountRiskKeyPrefix+accountRiskLoginAccountKeyPrefix+key)
	s.store.Reset(ctx, accountRiskKeyPrefix+accountRiskLoginAccountLockPrefix+key)
}

// accountRiskLoginAccountKey 把邮箱哈希成计数键，避免在 Redis 键里落明文邮箱。
func accountRiskLoginAccountKey(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:8])
}

// riskClientIP 从请求上下文取安全客户端 IP（与审计/会话绑定同源）。
func riskClientIP(ctx context.Context) string {
	if binding := SessionBindingFromContext(ctx); binding != nil {
		return strings.TrimSpace(binding.IP)
	}
	return ""
}

// riskClientUserAgent 从请求上下文取 UA。
func riskClientUserAgent(ctx context.Context) string {
	if binding := SessionBindingFromContext(ctx); binding != nil {
		return binding.UserAgent
	}
	return ""
}
