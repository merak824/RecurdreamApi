package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrAccountRiskRegistrationBlocked 同 IP 批量注册被风控拦截。
// 返回 429：既表明是限流类拒绝，又不向注册机暴露"封禁"这类语义。
var ErrAccountRiskRegistrationBlocked = infraerrors.TooManyRequests(
	"REGISTRATION_BLOCKED_RISK",
	"too many accounts registered from this network, please try again later or contact support",
)

// automationUserAgentMarkers 明确的自动化 / 脚本客户端 UA 片段。
// 命中即视为"非人类浏览器"，这是注册机最高置信度的信号。
var automationUserAgentMarkers = []string{
	"python-requests", "python-urllib", "python-httpx", "aiohttp",
	"requests/", "httpx/", "curl/", "wget/", "libwww-perl", "lwp-trivial",
	"go-http-client", "okhttp", "apache-httpclient", "httpclient/",
	"java/", "scrapy", "node-fetch", "axios/", "undici", "got (",
	"guzzlehttp", "restsharp", "reqwest", "dart:io", "dio/",
	"postmanruntime", "insomnia", "httpie",
}

// browserUserAgentMarkers 人类浏览器的核心标识片段。
var browserUserAgentMarkers = []string{
	"mozilla/", "chrome/", "firefox/", "safari/", "edg/", "opr/",
	"micromessenger", "qqbrowser", "ucbrowser", "headlesschrome",
}

// looksLikeAutomationUserAgent 判断 UA 是否为明确的脚本/自动化客户端。
func looksLikeAutomationUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return true
	}
	for _, marker := range automationUserAgentMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// looksLikeBrowserUserAgent 判断 UA 是否像一个真实浏览器。
func looksLikeBrowserUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return false
	}
	// 明确脚本 UA 即使带 Mozilla 前缀（部分库会伪装）也不算浏览器。
	for _, marker := range automationUserAgentMarkers {
		if strings.Contains(ua, marker) {
			return false
		}
	}
	for _, marker := range browserUserAgentMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// registerProbeRule "注册即探测"规则。
//
// 判定逻辑：账号注册后 window 分钟内，若出现非浏览器客户端的鉴权请求，
// 说明注册流程是脚本驱动的（注册机典型特征：注册完立刻用 python/curl 拉取 profile
// 验证账号是否可用）。分级处置：
//   - 明确脚本 UA / 空 UA → 按配置动作处理（默认封禁）
//   - 无法识别的非浏览器 UA → 只留痕，避免误杀自研客户端
type registerProbeRule struct{}

func (registerProbeRule) Name() string { return "register_probe" }

func (registerProbeRule) Events() []AccountRiskEventType {
	return []AccountRiskEventType{RiskEventFirstAPI}
}

func (registerProbeRule) Evaluate(_ context.Context, eval *AccountRiskEvalContext) []AccountRiskVerdict {
	if eval == nil || eval.Config == nil || !eval.Config.RegisterProbe.Enabled {
		return nil
	}
	user := eval.User
	if user == nil || user.ID <= 0 || user.CreatedAt.IsZero() {
		return nil
	}
	window := eval.Config.RegisterProbe.WindowMinutes
	if window <= 0 {
		return nil
	}
	age := eval.Now.Sub(user.CreatedAt)
	if age < 0 {
		age = 0
	}
	if age > time.Duration(window)*time.Minute {
		return nil // 老账号不参与"注册即探测"判定
	}

	ua := eval.Event.UserAgent
	signals := map[string]any{
		"account_age_seconds": int64(age.Seconds()),
		"window_minutes":      window,
		"user_agent":          truncateRiskString(ua, 200),
		"path":                eval.Event.Path,
	}

	if looksLikeAutomationUserAgent(ua) {
		return []AccountRiskVerdict{{
			Rule:    registerProbeRule{}.Name(),
			Action:  eval.Config.RegisterProbe.Action,
			Score:   95,
			Reason:  "new account used a non-browser client right after registration",
			Signals: signals,
		}}
	}
	if !looksLikeBrowserUserAgent(ua) {
		// 未知 UA：可能是自研客户端，保留观察而不处置。
		return []AccountRiskVerdict{{
			Rule:    registerProbeRule{}.Name(),
			Action:  RiskActionFlag,
			Score:   50,
			Reason:  "new account used an unrecognized client right after registration",
			Signals: signals,
		}}
	}
	return nil
}

// multiAccountRule 同 IP 批量注册规则。
//
// 计数在注册成功时由 AccountRiskService.ObserveRegistration 自增，
// 前置拦截由 CheckRegistration 完成；本规则负责在"已经建号"的场景下
// 按配置补一刀（action=ban 时封禁新账号）。
type multiAccountRule struct{}

func (multiAccountRule) Name() string { return "multi_account_ip" }

func (multiAccountRule) Events() []AccountRiskEventType {
	return []AccountRiskEventType{RiskEventRegister}
}

func (multiAccountRule) Evaluate(ctx context.Context, eval *AccountRiskEvalContext) []AccountRiskVerdict {
	if eval == nil || eval.Config == nil || !eval.Config.MultiAccount.Enabled || eval.Store == nil {
		return nil
	}
	ip := strings.TrimSpace(eval.Event.IP)
	if ip == "" {
		return nil
	}
	maxAccounts := eval.Config.MultiAccount.MaxAccountsPerIP
	if maxAccounts <= 0 {
		return nil
	}
	count, err := eval.Store.Get(ctx, accountRiskIPRegistrationKey(ip))
	if err != nil || count < int64(maxAccounts) {
		return nil
	}
	action := eval.Config.MultiAccount.Action
	if action == RiskActionBlock {
		// 前置拦截已经生效；走到这里说明是并发竞态，按封禁兜底。
		action = RiskActionBan
	}
	return []AccountRiskVerdict{{
		Rule:   multiAccountRule{}.Name(),
		Action: action,
		Score:  80,
		Reason: fmt.Sprintf("ip registered %d accounts within %dh", count, eval.Config.MultiAccount.WindowHours),
		Signals: map[string]any{
			"ip":           ip,
			"count":        count,
			"max_accounts": maxAccounts,
			"window_hours": eval.Config.MultiAccount.WindowHours,
		},
	}}
}

// sessionAbuseRule 会话指纹异常规则。
//
// 会话绑定每次不匹配都会撤销整条会话家族；同一账号在窗口内反复不匹配，
// 说明 token 被复制到了多个网络环境（共享/盗用）。
type sessionAbuseRule struct{}

func (sessionAbuseRule) Name() string { return "session_abuse" }

func (sessionAbuseRule) Events() []AccountRiskEventType {
	return []AccountRiskEventType{RiskEventSessionMismatch}
}

func (sessionAbuseRule) Evaluate(ctx context.Context, eval *AccountRiskEvalContext) []AccountRiskVerdict {
	if eval == nil || eval.Config == nil || !eval.Config.SessionAbuse.Enabled || eval.Store == nil {
		return nil
	}
	userID := eval.Event.UserID
	if userID <= 0 {
		return nil
	}
	maxMismatches := eval.Config.SessionAbuse.MaxMismatches
	if maxMismatches <= 0 {
		return nil
	}
	window := time.Duration(eval.Config.SessionAbuse.WindowHours) * time.Hour
	count, err := eval.Store.Incr(ctx, accountRiskSessionMismatchKey(userID), window)
	if err != nil || count < int64(maxMismatches) {
		return nil
	}
	return []AccountRiskVerdict{{
		Rule:   sessionAbuseRule{}.Name(),
		Action: eval.Config.SessionAbuse.Action,
		Score:  85,
		Reason: fmt.Sprintf("session fingerprint mismatched %d times within %dh", count, eval.Config.SessionAbuse.WindowHours),
		Signals: map[string]any{
			"count":          count,
			"max_mismatches": maxMismatches,
			"window_hours":   eval.Config.SessionAbuse.WindowHours,
			"ip":             eval.Event.IP,
			"path":           eval.Event.Path,
		},
	}}
}

// defaultAccountRiskRules 内置规则集（注册顺序即评估顺序）。
// 新增异常类型：实现 AccountRiskRule 后追加到此处即可。
func defaultAccountRiskRules() []AccountRiskRule {
	return []AccountRiskRule{
		registerProbeRule{},
		multiAccountRule{},
		sessionAbuseRule{},
	}
}

func truncateRiskString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
