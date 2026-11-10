package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 账号异常检测与自动封禁框架。
//
// 设计目标：把"判定"和"处置"彻底解耦，新增一类异常只需要
//   1) 在 AccountRiskEventType 里加一个事件；
//   2) 在 AccountRiskConfig 里加一段子配置；
//   3) 写一个实现 AccountRiskRule 的规则并注册到 defaultAccountRiskRules()。
// 处置动作（记录 / 标记 / 封禁 / 拦截）由框架统一执行、统一留痕，
// 规则本身不碰数据库、不发邮件、不写审计。
//
// 现有事件来源：
//   - register            注册成功（AuthService.RegisterWithVerification）
//   - first_api           新账号注册后的首批鉴权请求（JWT 中间件）
//   - session_mismatch    会话指纹（IP+UA）不匹配（会话绑定中间件）
//   - login_failure       登录失败（AuthService.Login，供登录防护使用）

// AccountRiskEventType 风控事件类型。
type AccountRiskEventType string

const (
	// RiskEventRegister 注册成功事件。
	RiskEventRegister AccountRiskEventType = "register"
	// RiskEventFirstAPI 新账号的鉴权请求事件（"注册即探测"的观测点）。
	RiskEventFirstAPI AccountRiskEventType = "first_api"
	// RiskEventSessionMismatch 会话指纹不匹配事件。
	RiskEventSessionMismatch AccountRiskEventType = "session_mismatch"
)

// AccountRiskEvent 一次待评估的风控事件。
type AccountRiskEvent struct {
	Type       AccountRiskEventType
	UserID     int64
	Email      string
	IP         string
	UserAgent  string
	Path       string
	OccurredAt time.Time
	// Extra 承载规则自定义的附加信号（如注册时命中次数），便于审计留痕。
	Extra map[string]any
}

// AccountRiskVerdict 单条规则的判定结果。
type AccountRiskVerdict struct {
	// Rule 规则名（写入审计，便于回溯是哪条规则命中的）。
	Rule string
	// Action 建议的处置动作。
	Action RiskAction
	// Reason 人类可读的命中原因（写入审计）。
	Reason string
	// Score 置信度 0-100，多规则同时命中时用于挑选主因。
	Score int
	// Signals 结构化信号（写入审计 Extra）。
	Signals map[string]any
}

// AccountRiskEvalContext 规则评估上下文。
// 由框架预加载好用户、配置与计数器，规则实现保持纯粹、可直接单测。
type AccountRiskEvalContext struct {
	Event  *AccountRiskEvent
	Config *AccountRiskConfig
	// User 事件关联的用户；用户不存在时为 nil。
	User *User
	// Store 按时间窗口计数的信号存储（Redis，测试可换成内存实现）。
	Store AccountRiskCounterStore
	Now   time.Time
}

// AccountRiskRule 风控规则。新增规则实现该接口并注册即可。
type AccountRiskRule interface {
	// Name 规则唯一名（与配置字段一一对应）。
	Name() string
	// Events 声明本规则关注的事件类型，框架只在匹配事件上调用 Evaluate。
	Events() []AccountRiskEventType
	// Evaluate 返回零到多条判定；返回空表示未命中。错误由框架记录后忽略。
	Evaluate(ctx context.Context, eval *AccountRiskEvalContext) []AccountRiskVerdict
}

// AccountRiskCounterStore 窗口计数存储抽象。
type AccountRiskCounterStore interface {
	// Incr 自增并返回当前值，首次自增时设置 TTL。
	Incr(ctx context.Context, key string, window time.Duration) (int64, error)
	// Get 读取当前值，不存在返回 0。
	Get(ctx context.Context, key string) (int64, error)
	// Reset 清除计数。
	Reset(ctx context.Context, key string)
	// SetFlag 写入一个带 TTL 的标记（已存在则不覆盖）并返回是否新建。
	SetFlag(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// HasFlag 判断标记是否存在。
	HasFlag(ctx context.Context, key string) bool
}

const accountRiskKeyPrefix = "account_risk:"

// memoryAccountRiskStore 进程内实现，仅用于单测与无 Redis 的降级场景。
type memoryAccountRiskStore struct {
	mu      sync.Mutex
	entries map[string]memoryAccountRiskEntry
}

type memoryAccountRiskEntry struct {
	value     int64
	expiresAt time.Time
}

func newMemoryAccountRiskStore() *memoryAccountRiskStore {
	return &memoryAccountRiskStore{entries: make(map[string]memoryAccountRiskEntry)}
}

func (s *memoryAccountRiskStore) Incr(_ context.Context, key string, window time.Duration) (int64, error) {
	if window <= 0 {
		window = time.Hour
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	entry, ok := s.entries[key]
	if !ok || now.After(entry.expiresAt) {
		entry = memoryAccountRiskEntry{expiresAt: now.Add(window)}
	}
	entry.value++
	s.entries[key] = entry
	return entry.value, nil
}

func (s *memoryAccountRiskStore) Get(_ context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, nil
	}
	return entry.value, nil
}

func (s *memoryAccountRiskStore) Reset(_ context.Context, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
}

func (s *memoryAccountRiskStore) SetFlag(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if entry, ok := s.entries[key]; ok && now.Before(entry.expiresAt) {
		return false, nil
	}
	s.entries[key] = memoryAccountRiskEntry{value: 1, expiresAt: now.Add(ttl)}
	return true, nil
}

func (s *memoryAccountRiskStore) HasFlag(_ context.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return false
	}
	return true
}

// AccountRiskService 账号异常检测与自动封禁服务。
type AccountRiskService struct {
	settingService       *SettingService
	userRepo             UserRepository
	auditService         *AuditLogService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	store                AccountRiskCounterStore
	rules                []AccountRiskRule
}

// NewAccountRiskService 创建风控服务。
// 除 settingService 外，其余依赖都允许为 nil（缺依赖时对应能力静默降级），
// 便于在只读场景或单测中构造。
func NewAccountRiskService(
	settingService *SettingService,
	userRepo UserRepository,
	auditService *AuditLogService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	store AccountRiskCounterStore,
) *AccountRiskService {
	if store == nil {
		store = newMemoryAccountRiskStore()
	}
	return &AccountRiskService{
		settingService:       settingService,
		userRepo:             userRepo,
		auditService:         auditService,
		authCacheInvalidator: authCacheInvalidator,
		store:                store,
		rules:                defaultAccountRiskRules(),
	}
}

// SetRules 替换规则集（仅用于测试或按部署裁剪规则）。
func (s *AccountRiskService) SetRules(rules ...AccountRiskRule) {
	if s == nil {
		return
	}
	s.rules = rules
}

// Rules 返回当前已注册规则名，便于自检与文档生成。
func (s *AccountRiskService) Rules() []string {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.rules))
	for _, rule := range s.rules {
		names = append(names, rule.Name())
	}
	return names
}

// Config 返回当前生效配置（带缓存）。
func (s *AccountRiskService) Config(ctx context.Context) *AccountRiskConfig {
	if s == nil || s.settingService == nil {
		return DefaultAccountRiskConfig()
	}
	return s.settingService.GetAccountRiskConfig(ctx)
}

// ---------------------------------------------------------------------------
// 事件入口
// ---------------------------------------------------------------------------

// CheckRegistration 注册前置拦截。
// 返回非 nil 表示本次注册应被拒绝（调用方直接把 error 返回给客户端）。
// 该检查是同步的：必须在创建账号之前完成，避免先建号再封号。
func (s *AccountRiskService) CheckRegistration(ctx context.Context, email, ip, userAgent string) error {
	if s == nil {
		return nil
	}
	cfg := s.Config(ctx)
	if cfg == nil || !cfg.Enabled || !cfg.MultiAccount.Enabled {
		return nil
	}
	if strings.TrimSpace(ip) == "" {
		// 拿不到可信客户端 IP 时不做阻断，避免把全站用户当成同一个 IP。
		return nil
	}
	if cfg.MultiAccount.Action == RiskActionFlag {
		return nil
	}

	count, err := s.store.Get(ctx, accountRiskIPRegistrationKey(ip))
	if err != nil {
		slog.Warn("account_risk.ip_counter_read_failed", "ip", ip, "error", err)
		return nil
	}
	if count < int64(cfg.MultiAccount.MaxAccountsPerIP) {
		return nil
	}

	s.recordAudit(ctx, &AccountRiskEvent{
		Type:      RiskEventRegister,
		Email:     email,
		IP:        ip,
		UserAgent: userAgent,
	}, []AccountRiskVerdict{{
		Rule:   "multi_account_ip",
		Action: RiskActionBlock,
		Reason: fmt.Sprintf("ip registered %d accounts within %dh", count, cfg.MultiAccount.WindowHours),
		Score:  80,
		Signals: map[string]any{
			"ip":           ip,
			"count":        count,
			"max_accounts": cfg.MultiAccount.MaxAccountsPerIP,
			"window_hours": cfg.MultiAccount.WindowHours,
		},
	}}, AuditActionAccountRiskBlocked, 429)

	return ErrAccountRiskRegistrationBlocked
}

// ObserveRegistration 注册成功后记录信号并评估"注册类"规则。
// 与请求解耦（异步），不阻塞注册响应。
func (s *AccountRiskService) ObserveRegistration(ctx context.Context, user *User, ip, userAgent string) {
	if s == nil || user == nil || user.ID <= 0 {
		return
	}
	cfg := s.Config(ctx)
	if cfg == nil || !cfg.Enabled {
		return
	}
	if ip = strings.TrimSpace(ip); ip != "" {
		if _, err := s.store.Incr(ctx, accountRiskIPRegistrationKey(ip), time.Duration(cfg.MultiAccount.WindowHours)*time.Hour); err != nil {
			slog.Warn("account_risk.ip_counter_incr_failed", "ip", ip, "error", err)
		}
	}
	s.dispatch(ctx, &AccountRiskEvent{
		Type:      RiskEventRegister,
		UserID:    user.ID,
		Email:     user.Email,
		IP:        ip,
		UserAgent: userAgent,
	})
}

// ShouldObserveNewUser 判断本次鉴权请求是否需要进入"注册即探测"观测。
// 热路径专用：只读进程内缓存配置 + 比较创建时间，不产生任何 IO 或 goroutine。
func (s *AccountRiskService) ShouldObserveNewUser(ctx context.Context, user *User) bool {
	if s == nil || user == nil || user.ID <= 0 || user.CreatedAt.IsZero() {
		return false
	}
	cfg := s.Config(ctx)
	if cfg == nil || !cfg.Enabled || !cfg.RegisterProbe.Enabled {
		return false
	}
	window := time.Duration(cfg.RegisterProbe.WindowMinutes) * time.Minute
	if window <= 0 {
		return false
	}
	return time.Since(user.CreatedAt) <= window
}

// ObserveNewUserRequest 新账号的鉴权请求（"注册即探测"观测点）。
// 调用方只需保证 user 非空；窗口判断在规则内完成。
func (s *AccountRiskService) ObserveNewUserRequest(ctx context.Context, user *User, ip, userAgent, path string) {
	if s == nil || user == nil || user.ID <= 0 {
		return
	}
	s.dispatch(ctx, &AccountRiskEvent{
		Type:      RiskEventFirstAPI,
		UserID:    user.ID,
		Email:     user.Email,
		IP:        ip,
		UserAgent: userAgent,
		Path:      path,
	})
}

// ObserveSessionMismatch 会话指纹不匹配事件。
func (s *AccountRiskService) ObserveSessionMismatch(ctx context.Context, userID int64, email, ip, userAgent, path string) {
	if s == nil || userID <= 0 {
		return
	}
	s.dispatch(ctx, &AccountRiskEvent{
		Type:      RiskEventSessionMismatch,
		UserID:    userID,
		Email:     email,
		IP:        ip,
		UserAgent: userAgent,
		Path:      path,
	})
}

// dispatch 异步评估（脱离请求生命周期，避免请求取消导致风控中断）。
func (s *AccountRiskService) dispatch(ctx context.Context, event *AccountRiskEvent) {
	if s == nil || event == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	detached := context.WithoutCancel(ctx)
	go func() {
		evalCtx, cancel := context.WithTimeout(detached, 5*time.Second)
		defer cancel()
		s.Evaluate(evalCtx, event)
	}()
}

// ---------------------------------------------------------------------------
// 评估与处置
// ---------------------------------------------------------------------------

// Evaluate 同步评估一个事件：跑规则 → 选出最高优先级动作 → 执行处置 → 留痕。
// 导出以便单测与运维手动重放。
func (s *AccountRiskService) Evaluate(ctx context.Context, event *AccountRiskEvent) []AccountRiskVerdict {
	if s == nil || event == nil {
		return nil
	}
	cfg := s.Config(ctx)
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	user := s.loadUser(ctx, event.UserID)
	if user != nil {
		if event.Email == "" {
			event.Email = user.Email
		}
		// 管理员与免检名单永远不处置，但仍允许记录（便于排查误判）。
		if user.IsAdmin() || cfg.IsUserAllowlisted(user.ID) {
			event.Extra = mergeRiskExtra(event.Extra, map[string]any{"exempt": true})
		}
	}

	eval := &AccountRiskEvalContext{
		Event:  event,
		Config: cfg,
		User:   user,
		Store:  s.store,
		Now:    time.Now(),
	}

	var verdicts []AccountRiskVerdict
	for _, rule := range s.rules {
		if !ruleApplies(rule, event.Type) {
			continue
		}
		got := rule.Evaluate(ctx, eval)
		for _, verdict := range got {
			if strings.TrimSpace(verdict.Rule) == "" {
				verdict.Rule = rule.Name()
			}
			verdicts = append(verdicts, verdict)
		}
	}
	if len(verdicts) == 0 {
		return nil
	}

	exempt := false
	if v, ok := event.Extra["exempt"].(bool); ok {
		exempt = v
	}

	best := pickRiskVerdict(verdicts)
	action := best.Action
	auditAction := AuditActionAccountRiskTriggered
	statusCode := 0

	switch {
	case exempt:
		// 免检用户：仅留痕，不处置。
		action = RiskActionFlag
		auditAction = AuditActionAccountRiskTriggered
	case action == RiskActionBan:
		if s.banUser(ctx, user) {
			auditAction = AuditActionAccountRiskBanned
			statusCode = 401
		} else {
			// 封禁失败（用户不存在 / 已是封禁态 / 更新失败）时降级为留痕。
			action = RiskActionFlag
		}
	case action == RiskActionBlock:
		auditAction = AuditActionAccountRiskBlocked
		statusCode = 429
	default:
		action = RiskActionFlag
	}

	s.recordAudit(ctx, event, verdicts, auditAction, statusCode)
	slog.Warn("account_risk.triggered",
		"rule", best.Rule,
		"action", string(action),
		"user_id", event.UserID,
		"email", event.Email,
		"ip", event.IP,
		"event", string(event.Type),
		"path", event.Path,
		"detail", best.Reason,
	)
	return verdicts
}

// banUser 封禁账号：置为 disabled + 失效认证缓存。
// 返回 true 表示本次调用真正执行了封禁。
func (s *AccountRiskService) banUser(ctx context.Context, user *User) bool {
	if s == nil || s.userRepo == nil || user == nil || user.ID <= 0 {
		return false
	}
	if user.IsAdmin() {
		return false
	}
	if user.Status == StatusDisabled {
		return false
	}
	user.Status = StatusDisabled
	if err := s.userRepo.Update(ctx, user, UserUpdateFields{Status: true}); err != nil {
		slog.Warn("account_risk.ban_update_failed", "user_id", user.ID, "error", err)
		return false
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)
	}
	return true
}

// loadUser 读取用户；失败或无仓库时返回 nil（规则需容忍 nil）。
func (s *AccountRiskService) loadUser(ctx context.Context, userID int64) *User {
	if s == nil || s.userRepo == nil || userID <= 0 {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	return user
}

// recordAudit 把命中结果写入审计日志（异步非阻塞队列）。
func (s *AccountRiskService) recordAudit(ctx context.Context, event *AccountRiskEvent, verdicts []AccountRiskVerdict, action string, statusCode int) {
	if s == nil || s.auditService == nil || event == nil {
		return
	}
	rules := make([]string, 0, len(verdicts))
	extra := make(map[string]any, len(verdicts)+4)
	for _, verdict := range verdicts {
		rules = append(rules, verdict.Rule)
		if len(verdict.Signals) > 0 {
			extra[verdict.Rule] = verdict.Signals
		}
	}
	extra["event"] = string(event.Type)
	extra["rules"] = strings.Join(rules, ",")
	if len(event.Extra) > 0 {
		extra["context"] = event.Extra
	}
	if len(verdicts) > 0 {
		extra["reason"] = verdicts[0].Reason
	}

	var actorID *int64
	if event.UserID > 0 {
		uid := event.UserID
		actorID = &uid
	}

	entry := &AuditLog{
		ActorUserID: actorID,
		ActorEmail:  event.Email,
		AuthMethod:  AuditAuthMethodSystemService,
		Action:      action,
		Method:      "SYSTEM",
		Path:        event.Path,
		ClientIP:    event.IP,
		UserAgent:   event.UserAgent,
		StatusCode:  statusCode,
		Extra:       extra,
	}
	if entry.Path == "" {
		entry.Path = "/internal/account-risk"
	}
	s.auditService.Record(entry)
}

func ruleApplies(rule AccountRiskRule, eventType AccountRiskEventType) bool {
	for _, candidate := range rule.Events() {
		if candidate == eventType {
			return true
		}
	}
	return false
}

// riskActionPriority 处置动作的优先级，数值越大越优先。
func riskActionPriority(action RiskAction) int {
	switch action {
	case RiskActionBan:
		return 3
	case RiskActionBlock:
		return 2
	case RiskActionFlag:
		return 1
	default:
		return 0
	}
}

// pickRiskVerdict 从多条判定中选出主判定：先看动作优先级，再看置信度。
func pickRiskVerdict(verdicts []AccountRiskVerdict) AccountRiskVerdict {
	best := verdicts[0]
	for _, candidate := range verdicts[1:] {
		if riskActionPriority(candidate.Action) > riskActionPriority(best.Action) {
			best = candidate
			continue
		}
		if riskActionPriority(candidate.Action) == riskActionPriority(best.Action) && candidate.Score > best.Score {
			best = candidate
		}
	}
	return best
}

func mergeRiskExtra(base map[string]any, extra map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any, len(extra))
	}
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func accountRiskIPRegistrationKey(ip string) string {
	return accountRiskKeyPrefix + "reg_ip:" + strings.TrimSpace(ip)
}

func accountRiskSessionMismatchKey(userID int64) string {
	return fmt.Sprintf("%s%s%d", accountRiskKeyPrefix, "sbm_uid:", userID)
}

// ---------------------------------------------------------------------------
// 进程级注册（供中间件等无法注入服务实例的位置使用）
// ---------------------------------------------------------------------------

var accountRiskServiceRef atomic.Pointer[AccountRiskService]

// SetAccountRiskService 注册进程级风控实例；传 nil 可关闭（测试用）。
func SetAccountRiskService(svc *AccountRiskService) {
	accountRiskServiceRef.Store(svc)
}

// AccountRisk 返回进程级风控实例，未初始化时返回 nil（所有方法都对 nil 安全）。
func AccountRisk() *AccountRiskService {
	return accountRiskServiceRef.Load()
}
