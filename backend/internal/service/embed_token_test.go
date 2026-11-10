package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestEmbedTargetOrigin(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "https plain", input: "https://catfk.com/shop/1", want: "https://catfk.com"},
		{name: "with port and caps", input: "HTTPS://CatFK.com:8443/x?y=1", want: "https://catfk.com:8443"},
		{name: "http allowed", input: "http://localhost:3000/page", want: "http://localhost:3000"},
		{name: "trailing spaces", input: "  https://a.example.com  ", want: "https://a.example.com"},
		{name: "empty", input: "", wantErr: true},
		{name: "relative", input: "/shop/1", wantErr: true},
		{name: "javascript scheme", input: "javascript:alert(1)", wantErr: true},
		{name: "data scheme", input: "data:text/html,<p>x</p>", wantErr: true},
		{name: "no host", input: "https:///path", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EmbedTargetOrigin(tc.input)
			if tc.wantErr {
				require.ErrorIs(t, err, ErrEmbedTargetInvalid)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestEmbedOriginAllowed(t *testing.T) {
	allowed := []string{"https://catfk.com", "https://Shop.Example.com/"}
	require.True(t, embedOriginAllowed("https://catfk.com", allowed))
	require.True(t, embedOriginAllowed("https://CATFK.com/", allowed), "忽略大小写与末尾斜杠")
	require.True(t, embedOriginAllowed("https://shop.example.com", allowed))
	require.False(t, embedOriginAllowed("https://evil.com", allowed))
	require.False(t, embedOriginAllowed("https://catfk.com.evil.com", allowed), "不允许后缀混淆")
	require.False(t, embedOriginAllowed("", allowed))
}

func TestIsEmbedScopePathAllowed(t *testing.T) {
	require.True(t, IsEmbedScopePathAllowed("GET", "/api/v1/user/profile"))
	require.True(t, IsEmbedScopePathAllowed("get", "/api/v1/user/profile"), "方法大小写不敏感")
	// 除只读 profile 外一律拒绝：写接口、其他读接口、空路由都不放行。
	require.False(t, IsEmbedScopePathAllowed("POST", "/api/v1/user/profile"))
	require.False(t, IsEmbedScopePathAllowed("GET", "/api/v1/user/aff"))
	require.False(t, IsEmbedScopePathAllowed("GET", "/api/v1/user/api-keys"))
	require.False(t, IsEmbedScopePathAllowed("GET", ""))
	require.False(t, IsEmbedScopePathAllowed("", "/api/v1/user/profile"))
}

// embedSettingRepo 支持批量读取的设置仓库替身。
// GetPublicSettings 走 GetMultiple，fakeSettingRepo 未实现该方法，这里补上。
type embedSettingRepo struct {
	fakeSettingRepo
}

func (r *embedSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.vals[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func newEmbedTokenTestService(t *testing.T, menuItems string) (*AuthService, *SettingService) {
	t.Helper()
	vals := map[string]string{}
	if menuItems != "" {
		vals[SettingKeyCustomMenuItems] = menuItems
	}
	settingService := &SettingService{settingRepo: &embedSettingRepo{fakeSettingRepo: fakeSettingRepo{vals: vals}}}
	cfg := &config.Config{}
	cfg.JWT.Secret = "embed-token-test-secret"
	return &AuthService{cfg: cfg, settingService: settingService}, settingService
}

func TestIssueEmbedTokenRejectsOriginOutsideAllowlist(t *testing.T) {
	svc, _ := newEmbedTokenTestService(t, `[{"label":"充值","url":"https://catfk.com/shop/1"}]`)
	user := &User{ID: 7, Email: "u@example.com", Role: RoleUser}

	_, _, err := svc.IssueEmbedToken(context.Background(), user, "https://evil.example.net/steal")
	require.ErrorIs(t, err, ErrEmbedOriginNotAllowed)

	_, _, err = svc.IssueEmbedToken(context.Background(), user, "not-a-url")
	require.ErrorIs(t, err, ErrEmbedTargetInvalid)
}

func TestIssueEmbedTokenSignsScopedShortLivedToken(t *testing.T) {
	svc, _ := newEmbedTokenTestService(t, `[{"label":"充值","url":"https://catfk.com/shop/1"}]`)
	user := &User{ID: 7, Email: "u@example.com", Role: RoleUser}

	token, expiresIn, err := svc.IssueEmbedToken(context.Background(), user, "https://catfk.com/shop/1")
	require.NoError(t, err)
	require.Equal(t, int(EmbedTokenTTL.Seconds()), expiresIn)
	require.NotEmpty(t, token)

	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return []byte(svc.cfg.JWT.Secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	require.Equal(t, JWTScopeEmbed, claims.Scope)
	require.Equal(t, user.ID, claims.UserID)
	require.Empty(t, claims.SessionID, "嵌入令牌不携带会话 ID")
	require.Empty(t, claims.BindingHash, "嵌入令牌不绑指纹（第三方来源 IP 必然不同）")
	require.NotNil(t, claims.ExpiresAt)
	require.WithinDuration(t, time.Now().Add(EmbedTokenTTL), claims.ExpiresAt.Time, 30*time.Second)
	require.Equal(t, []string{"https://catfk.com"}, []string(claims.Audience))
}

func TestIssueEmbedTokenRequiresUserAndTarget(t *testing.T) {
	svc, _ := newEmbedTokenTestService(t, `[{"label":"充值","url":"https://catfk.com/shop/1"}]`)

	_, _, err := svc.IssueEmbedToken(context.Background(), nil, "https://catfk.com/shop/1")
	require.Error(t, err)

	_, _, err = svc.IssueEmbedToken(context.Background(), &User{ID: 0}, "https://catfk.com/shop/1")
	require.Error(t, err)
}
