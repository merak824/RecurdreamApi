package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// accountRiskUserRepo 只实现风控用到的两个方法；其余方法通过嵌入接口占位。
type accountRiskUserRepo struct {
	UserRepository
	users   map[int64]*User
	updates []UserUpdateFields
	byEmail map[string]*User
}

func newAccountRiskUserRepo(users ...*User) *accountRiskUserRepo {
	repo := &accountRiskUserRepo{
		users:   make(map[int64]*User),
		byEmail: make(map[string]*User),
	}
	for _, user := range users {
		repo.users[user.ID] = user
		repo.byEmail[user.Email] = user
	}
	return repo
}

func (r *accountRiskUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (r *accountRiskUserRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	if user, ok := r.byEmail[email]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (r *accountRiskUserRepo) Update(_ context.Context, user *User, fields UserUpdateFields) error {
	r.updates = append(r.updates, fields)
	r.users[user.ID] = user
	return nil
}

// accountRiskSettingRepo 记录 Set 调用的设置仓库。
type accountRiskSettingRepo struct {
	fakeSettingRepo
	sets map[string]string
}

func newAccountRiskSettingRepo(vals map[string]string) *accountRiskSettingRepo {
	if vals == nil {
		vals = map[string]string{}
	}
	return &accountRiskSettingRepo{
		fakeSettingRepo: fakeSettingRepo{vals: vals},
		sets:            map[string]string{},
	}
}

func (r *accountRiskSettingRepo) Set(_ context.Context, key, value string) error {
	r.sets[key] = value
	r.vals[key] = value
	return nil
}

func newAccountRiskSettingService(t *testing.T, cfg *AccountRiskConfig) (*SettingService, *accountRiskSettingRepo) {
	t.Helper()
	raw := ""
	if cfg != nil {
		data, err := json.Marshal(cfg)
		require.NoError(t, err)
		raw = string(data)
	}
	repo := newAccountRiskSettingRepo(map[string]string{SettingKeyAccountRiskConfig: raw})
	return &SettingService{settingRepo: repo}, repo
}

func TestLooksLikeUserAgentClassification(t *testing.T) {
	browserUAs := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 MicroMessenger/8.0.42",
	}
	for _, ua := range browserUAs {
		require.False(t, looksLikeAutomationUserAgent(ua), "browser UA must not be automation: %s", ua)
		require.True(t, looksLikeBrowserUserAgent(ua), "browser UA must be detected: %s", ua)
	}

	automationUAs := []string{
		"python-requests/2.32.3",
		"curl/8.7.1",
		"Go-http-client/1.1",
		"",
		"okhttp/4.12.0",
	}
	for _, ua := range automationUAs {
		require.True(t, looksLikeAutomationUserAgent(ua), "automation UA must be detected: %q", ua)
		require.False(t, looksLikeBrowserUserAgent(ua), "automation UA must not look like browser: %q", ua)
	}

	// 未知但非脚本的客户端：不算自动化，但也不算浏览器。
	unknown := "MyCustomClient/1.0"
	require.False(t, looksLikeAutomationUserAgent(unknown))
	require.False(t, looksLikeBrowserUserAgent(unknown))
}

func TestRegisterProbeRule(t *testing.T) {
	now := time.Now()
	rule := registerProbeRule{}

	newUser := &User{ID: 7, Email: "bot@example.com", CreatedAt: now.Add(-2 * time.Minute)}

	t.Run("new account with script client is banned", func(t *testing.T) {
		cfg := DefaultAccountRiskConfig()
		verdicts := rule.Evaluate(context.Background(), &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 7, UserAgent: "python-requests/2.32.3", Path: "/api/v1/user/profile"},
			Config: cfg,
			User:   newUser,
			Now:    now,
		})
		require.Len(t, verdicts, 1)
		require.Equal(t, RiskActionBan, verdicts[0].Action)
		require.Equal(t, "register_probe", verdicts[0].Rule)
	})

	t.Run("new account with browser is ignored", func(t *testing.T) {
		cfg := DefaultAccountRiskConfig()
		verdicts := rule.Evaluate(context.Background(), &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 7, UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/131.0.0.0 Safari/537.36"},
			Config: cfg,
			User:   newUser,
			Now:    now,
		})
		require.Empty(t, verdicts)
	})

	t.Run("unknown client only flags", func(t *testing.T) {
		cfg := DefaultAccountRiskConfig()
		verdicts := rule.Evaluate(context.Background(), &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 7, UserAgent: "MyCustomClient/1.0"},
			Config: cfg,
			User:   newUser,
			Now:    now,
		})
		require.Len(t, verdicts, 1)
		require.Equal(t, RiskActionFlag, verdicts[0].Action)
	})

	t.Run("old account is out of window", func(t *testing.T) {
		cfg := DefaultAccountRiskConfig()
		verdicts := rule.Evaluate(context.Background(), &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 7, UserAgent: "python-requests/2.32.3"},
			Config: cfg,
			User:   &User{ID: 7, CreatedAt: now.Add(-48 * time.Hour)},
			Now:    now,
		})
		require.Empty(t, verdicts)
	})

	t.Run("disabled rule does nothing", func(t *testing.T) {
		cfg := DefaultAccountRiskConfig()
		cfg.RegisterProbe.Enabled = false
		verdicts := rule.Evaluate(context.Background(), &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 7, UserAgent: "python-requests/2.32.3"},
			Config: cfg,
			User:   newUser,
			Now:    now,
		})
		require.Empty(t, verdicts)
	})
}

func TestMultiAccountRule(t *testing.T) {
	ctx := context.Background()
	store := newMemoryAccountRiskStore()
	cfg := DefaultAccountRiskConfig()
	cfg.MultiAccount.MaxAccountsPerIP = 2

	verdict := func() []AccountRiskVerdict {
		return multiAccountRule{}.Evaluate(ctx, &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventRegister, IP: "1.2.3.4"},
			Config: cfg,
			Store:  store,
			Now:    time.Now(),
		})
	}

	require.Empty(t, verdict(), "first registration on the ip must pass")
	key := accountRiskIPRegistrationKey("1.2.3.4")
	_, err := store.Incr(ctx, key, time.Hour)
	require.NoError(t, err)
	require.Empty(t, verdict(), "second registration on the ip must pass")

	_, err = store.Incr(ctx, key, time.Hour)
	require.NoError(t, err)
	got := verdict()
	require.Len(t, got, 1)
	// block 动作在前置检查里生效；走到这里说明是并发竞态，按封禁兜底。
	require.Equal(t, RiskActionBan, got[0].Action)
}

func TestSessionAbuseRule(t *testing.T) {
	ctx := context.Background()
	store := newMemoryAccountRiskStore()
	cfg := DefaultAccountRiskConfig()
	cfg.SessionAbuse.MaxMismatches = 3
	cfg.SessionAbuse.Action = RiskActionBan

	eval := func() []AccountRiskVerdict {
		return sessionAbuseRule{}.Evaluate(ctx, &AccountRiskEvalContext{
			Event:  &AccountRiskEvent{Type: RiskEventSessionMismatch, UserID: 42, IP: "9.9.9.9"},
			Config: cfg,
			Store:  store,
			Now:    time.Now(),
		})
	}

	require.Empty(t, eval())
	require.Empty(t, eval())
	got := eval()
	require.Len(t, got, 1)
	require.Equal(t, RiskActionBan, got[0].Action)
	require.Equal(t, "session_abuse", got[0].Rule)
}

func TestAccountRiskEvaluateBansScriptProbeRegistration(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultAccountRiskConfig()

	settingService, _ := newAccountRiskSettingService(t, cfg)
	user := &User{ID: 338, Email: "d39aad7ad7e5@mail3.example.invalid", Role: RoleUser, Status: StatusActive, CreatedAt: time.Now().Add(-7 * time.Second)}
	userRepo := newAccountRiskUserRepo(user)

	svc := NewAccountRiskService(settingService, userRepo, nil, nil, newMemoryAccountRiskStore())

	verdicts := svc.Evaluate(ctx, &AccountRiskEvent{
		Type:      RiskEventFirstAPI,
		UserID:    user.ID,
		Email:     user.Email,
		IP:        "64.83.24.224",
		UserAgent: "python-requests/2.32.3",
		Path:      "/api/v1/user/profile",
	})

	require.NotEmpty(t, verdicts)
	require.Len(t, userRepo.updates, 1)
	require.True(t, userRepo.updates[0].Status)
	require.Equal(t, StatusDisabled, userRepo.users[user.ID].Status)
}

func TestAccountRiskEvaluateSkipsAdminAndAllowlist(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultAccountRiskConfig()
	cfg.AllowlistedUserIDs = []int64{99}
	settingService, _ := newAccountRiskSettingService(t, cfg)

	admin := &User{ID: 1, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive, CreatedAt: time.Now()}
	listed := &User{ID: 99, Email: "vip@example.com", Role: RoleUser, Status: StatusActive, CreatedAt: time.Now()}
	userRepo := newAccountRiskUserRepo(admin, listed)
	svc := NewAccountRiskService(settingService, userRepo, nil, nil, newMemoryAccountRiskStore())

	for _, target := range []*User{admin, listed} {
		verdicts := svc.Evaluate(ctx, &AccountRiskEvent{
			Type:      RiskEventFirstAPI,
			UserID:    target.ID,
			UserAgent: "curl/8.7.1",
		})
		require.NotEmpty(t, verdicts, "命中仍要留痕，便于排查误判")
		require.Empty(t, userRepo.updates, "管理员与免检用户不允许被处置")
		require.Equal(t, StatusActive, target.Status)
	}
}

func TestAccountRiskDisabledDoesNothing(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultAccountRiskConfig()
	cfg.Enabled = false
	settingService, _ := newAccountRiskSettingService(t, cfg)

	user := &User{ID: 5, Email: "u@example.com", Role: RoleUser, Status: StatusActive, CreatedAt: time.Now()}
	userRepo := newAccountRiskUserRepo(user)
	svc := NewAccountRiskService(settingService, userRepo, nil, nil, newMemoryAccountRiskStore())

	require.Empty(t, svc.Evaluate(ctx, &AccountRiskEvent{Type: RiskEventFirstAPI, UserID: 5, UserAgent: "curl/8.7.1"}))
	require.Empty(t, userRepo.updates)
}

func TestCheckRegistrationBlocksRepeatedIP(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultAccountRiskConfig()
	cfg.MultiAccount.MaxAccountsPerIP = 1
	cfg.MultiAccount.Action = RiskActionBlock
	settingService, _ := newAccountRiskSettingService(t, cfg)

	store := newMemoryAccountRiskStore()
	svc := NewAccountRiskService(settingService, newAccountRiskUserRepo(), nil, nil, store)

	require.NoError(t, svc.CheckRegistration(ctx, "a@example.com", "1.1.1.1", "Mozilla/5.0 Chrome/131"))

	_, err := store.Incr(ctx, accountRiskIPRegistrationKey("1.1.1.1"), time.Hour)
	require.NoError(t, err)

	err = svc.CheckRegistration(ctx, "b@example.com", "1.1.1.1", "Mozilla/5.0 Chrome/131")
	require.ErrorIs(t, err, ErrAccountRiskRegistrationBlocked)

	// 换个 IP 不受影响。
	require.NoError(t, svc.CheckRegistration(ctx, "c@example.com", "2.2.2.2", "Mozilla/5.0 Chrome/131"))
	// 缺少可信 IP 时不做拦截。
	require.NoError(t, svc.CheckRegistration(ctx, "d@example.com", "", "Mozilla/5.0 Chrome/131"))
}

func TestLoginGuardLocksIPAndAccount(t *testing.T) {
	cfg := DefaultAccountRiskConfig()
	cfg.LoginGuard.MaxIPFailures = 5
	cfg.LoginGuard.MaxAccountFailures = 2
	cfg.LoginGuard.LockMinutes = 5
	settingService, _ := newAccountRiskSettingService(t, cfg)

	svc := NewAccountRiskService(settingService, newAccountRiskUserRepo(), nil, nil, newMemoryAccountRiskStore())

	loginCtx := func(ip string) context.Context {
		binding := &SessionBinding{IP: ip, UserAgent: "python-requests/2.32.3"}
		return WithSessionBinding(context.Background(), binding)
	}

	// 账号维度先到阈值（2 次）。
	ipCtx := loginCtx("5.5.5.5")
	require.NoError(t, svc.LoginAccountPreCheck(ipCtx, "victim@example.com"))
	require.NoError(t, svc.LoginFailed(ipCtx, "victim@example.com"))
	require.NoError(t, svc.LoginFailed(ipCtx, "victim@example.com"))
	require.ErrorIs(t, svc.LoginAccountPreCheck(ipCtx, "victim@example.com"), ErrAccountRiskLoginLocked)

	// 登录成功后账号维度解锁，但 IP 计数保留。
	svc.LoginSucceeded(ipCtx, "victim@example.com")
	require.NoError(t, svc.LoginAccountPreCheck(ipCtx, "victim@example.com"))

	// IP 维度：累计 5 次失败后锁定该 IP 的登录入口（前 2 次来自上面 victim 的失败）。
	require.NoError(t, svc.LoginFailed(ipCtx, "other1@example.com"))
	require.NoError(t, svc.LoginFailed(ipCtx, "other2@example.com"))
	require.NoError(t, svc.LoginPreCheck(ipCtx), "未到阈值前不应锁定")
	require.NoError(t, svc.LoginFailed(ipCtx, "other3@example.com"))
	require.ErrorIs(t, svc.LoginPreCheck(ipCtx), ErrAccountRiskLoginLocked)
}

func TestAccountRiskConfigNormalize(t *testing.T) {
	cfg := &AccountRiskConfig{
		Enabled:            true,
		RegisterProbe:      AccountRiskRegisterProbeConfig{WindowMinutes: -5, Action: "nonsense"},
		MultiAccount:       AccountRiskMultiAccountConfig{MaxAccountsPerIP: 0, Action: ""},
		SessionAbuse:       AccountRiskSessionAbuseConfig{WindowHours: 999999},
		LoginGuard:         AccountRiskLoginGuardConfig{LockMinutes: 999999},
		AllowlistedUserIDs: []int64{0, -1, 12, 12, 13},
	}
	require.NoError(t, cfg.normalize())

	defaults := DefaultAccountRiskConfig()
	require.Equal(t, defaults.RegisterProbe.WindowMinutes, cfg.RegisterProbe.WindowMinutes)
	require.Equal(t, defaults.RegisterProbe.Action, cfg.RegisterProbe.Action)
	require.Equal(t, defaults.MultiAccount.MaxAccountsPerIP, cfg.MultiAccount.MaxAccountsPerIP)
	require.Equal(t, 24*30, cfg.SessionAbuse.WindowHours)
	require.Equal(t, 24*60, cfg.LoginGuard.LockMinutes)
	require.Equal(t, []int64{12, 13}, cfg.AllowlistedUserIDs)
}

func TestSetAccountRiskConfigPersistsAndRefreshesCache(t *testing.T) {
	ctx := context.Background()
	settingService, repo := newAccountRiskSettingService(t, nil)

	cfg := DefaultAccountRiskConfig()
	cfg.MultiAccount.MaxAccountsPerIP = 7
	require.NoError(t, settingService.SetAccountRiskConfig(ctx, cfg))

	require.Contains(t, repo.sets, SettingKeyAccountRiskConfig)
	got := settingService.GetAccountRiskConfig(ctx)
	require.Equal(t, 7, got.MultiAccount.MaxAccountsPerIP)
}

func TestGetAccountRiskConfigFallsBackToDefaults(t *testing.T) {
	ctx := context.Background()

	t.Run("missing setting", func(t *testing.T) {
		settingService := &SettingService{settingRepo: newAccountRiskSettingRepo(map[string]string{})}
		cfg := settingService.GetAccountRiskConfig(ctx)
		require.NotNil(t, cfg)
		require.Equal(t, DefaultAccountRiskConfig().RegisterProbe.Action, cfg.RegisterProbe.Action)
	})

	t.Run("broken json", func(t *testing.T) {
		settingService := &SettingService{settingRepo: newAccountRiskSettingRepo(map[string]string{SettingKeyAccountRiskConfig: "{not json"})}
		cfg := settingService.GetAccountRiskConfig(ctx)
		require.NotNil(t, cfg)
		require.True(t, cfg.Enabled)
	})

	t.Run("nil service", func(t *testing.T) {
		var settingService *SettingService
		require.NotNil(t, settingService.GetAccountRiskConfig(ctx))
	})
}

func TestAccountRiskServiceNilSafety(t *testing.T) {
	var svc *AccountRiskService
	ctx := context.Background()

	require.NotPanics(t, func() {
		require.NoError(t, svc.CheckRegistration(ctx, "a@b.com", "1.1.1.1", "curl/8"))
		svc.ObserveRegistration(ctx, &User{ID: 1}, "1.1.1.1", "curl/8")
		svc.ObserveNewUserRequest(ctx, &User{ID: 1}, "1.1.1.1", "curl/8", "/x")
		svc.ObserveSessionMismatch(ctx, 1, "a@b.com", "1.1.1.1", "curl/8", "/x")
		require.False(t, svc.ShouldObserveNewUser(ctx, &User{ID: 1, CreatedAt: time.Now()}))
		require.Empty(t, svc.Evaluate(ctx, &AccountRiskEvent{Type: RiskEventFirstAPI}))
		require.NoError(t, svc.LoginPreCheck(ctx))
		require.NoError(t, svc.LoginFailed(ctx, "a@b.com"))
		svc.LoginSucceeded(ctx, "a@b.com")
	})
}
