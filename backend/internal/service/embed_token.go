package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
)

// iframe 嵌入用的受限短期令牌。
//
// 背景：自定义菜单 / 首页内容可以把第三方页面以 iframe 嵌入工作台，
// 嵌入页面需要知道"当前登录用户是谁"。早期实现直接把 24 小时有效、
// 会自动续期的会话 JWT 拼进 iframe URL，第三方站点的 JS、日志、Referer
// 都能拿到它 —— 一旦泄露等于把整个账号交出去（会话绑定与 CORS 只是兜底）。
//
// 本文件提供替代方案：签发一个
//   - 5 分钟有效；
//   - 作用域 scope=embed；
//   - 不带会话 ID 与指纹绑定（第三方服务端来源 IP 必然不同，不能绑）；
//   - 只能访问只读的白名单路由（见 IsEmbedScopePathAllowed）；
//   - 签发时校验目标 origin 在我方 iframe 白名单内（不能签给任意站点）。
// 的窄权限令牌。泄露窗口从 24 小时缩到 5 分钟，且无法调用任何写接口。

const (
	// JWTScopeEmbed 受限嵌入令牌的作用域标记。
	JWTScopeEmbed = "embed"
	// EmbedTokenTTL 受限嵌入令牌有效期。
	// 取值权衡：太短会让用户在 iframe 内操作中途失效，太长则放大泄露风险。
	EmbedTokenTTL = 5 * time.Minute
	// embedScopeMaxTargetURLLength 限制请求体里的目标 URL 长度，避免被塞入超长串。
	embedScopeMaxTargetURLLength = 2048
)

var (
	// ErrEmbedTargetInvalid 目标 URL 非法（非 http/https 或缺 host）。
	ErrEmbedTargetInvalid = infraerrors.BadRequest("EMBED_TARGET_INVALID", "target url must be an absolute http(s) url")
	// ErrEmbedOriginNotAllowed 目标 origin 不在 iframe 白名单内。
	ErrEmbedOriginNotAllowed = infraerrors.BadRequest(
		"EMBED_ORIGIN_NOT_ALLOWED",
		"target origin is not allowed for embedding; add it to the iframe allowlist first",
	)
)

// embedScopeAllowedRoutes 受限嵌入令牌允许访问的路由（method + gin 路由模板）。
// 只放只读、且是"确认当前用户身份"所必需的接口；新增项需评估泄露后的影响。
var embedScopeAllowedRoutes = map[string]struct{}{
	"GET /api/v1/user/profile": {},
}

// IsEmbedScopePathAllowed 判断受限嵌入令牌能否访问该路由。
func IsEmbedScopePathAllowed(method, routeTemplate string) bool {
	if strings.EqualFold(method, "") || routeTemplate == "" {
		return false
	}
	_, ok := embedScopeAllowedRoutes[strings.ToUpper(method)+" "+routeTemplate]
	return ok
}

// EmbedTargetOrigin 从目标 URL 提取 origin（scheme://host[:port]）。
func EmbedTargetOrigin(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(rawURL) > embedScopeMaxTargetURLLength {
		return "", ErrEmbedTargetInvalid
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", ErrEmbedTargetInvalid
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrEmbedTargetInvalid
	}
	host := strings.ToLower(parsed.Host)
	if host == "" {
		return "", ErrEmbedTargetInvalid
	}
	return scheme + "://" + host, nil
}

// IssueEmbedToken 为 iframe 嵌入签发受限短期令牌，返回令牌与有效期（秒）。
//
// 目标 origin 必须已存在于 iframe 白名单（与 CSP frame-src 同一份来源，
// 见 SettingService.GetFrameSrcOrigins），避免本接口变成"给任意站点签发凭证"。
func (s *AuthService) IssueEmbedToken(ctx context.Context, user *User, targetURL string) (string, int, error) {
	if s == nil || user == nil || user.ID <= 0 {
		return "", 0, ErrServiceUnavailable
	}
	origin, err := EmbedTargetOrigin(targetURL)
	if err != nil {
		return "", 0, err
	}
	if s.settingService == nil {
		return "", 0, ErrServiceUnavailable
	}
	allowed, err := s.settingService.GetFrameSrcOrigins(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("load frame origins: %w", err)
	}
	if !embedOriginAllowed(origin, allowed) {
		return "", 0, ErrEmbedOriginNotAllowed
	}

	now := time.Now()
	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: resolvedTokenVersion(user),
		Scope:        JWTScopeEmbed,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(EmbedTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Audience:  jwt.ClaimStrings{origin},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", 0, fmt.Errorf("sign embed token: %w", err)
	}
	return tokenString, int(EmbedTokenTTL.Seconds()), nil
}

// embedOriginAllowed 判断 origin 是否在白名单内（忽略大小写与末尾斜杠）。
func embedOriginAllowed(origin string, allowed []string) bool {
	target := strings.ToLower(strings.TrimRight(strings.TrimSpace(origin), "/"))
	if target == "" {
		return false
	}
	for _, item := range allowed {
		if strings.ToLower(strings.TrimRight(strings.TrimSpace(item), "/")) == target {
			return true
		}
	}
	return false
}
