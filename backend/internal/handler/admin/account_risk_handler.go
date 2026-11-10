package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountRiskHandler 账号风控（账号异常检测与自动封禁）管理接口。
//
// 命中记录不单独建表：所有命中都写入管理面审计日志
// （action 前缀 security.risk.），共享既有的保留期清理与查询能力。
type AccountRiskHandler struct {
	settingService  *service.SettingService
	auditLogService *service.AuditLogService
	riskService     *service.AccountRiskService
}

// NewAccountRiskHandler 创建风控管理处理器。
func NewAccountRiskHandler(
	settingService *service.SettingService,
	auditLogService *service.AuditLogService,
	riskService *service.AccountRiskService,
) *AccountRiskHandler {
	return &AccountRiskHandler{
		settingService:  settingService,
		auditLogService: auditLogService,
		riskService:     riskService,
	}
}

// accountRiskEventActionPrefix 风控事件在审计日志中的 action 前缀。
// 与 service.AuditActionAccountRisk* 保持一致，用前缀匹配一次取全。
const accountRiskEventActionPrefix = "security.risk."

// GetConfig 返回当前生效的风控配置与已注册规则。
// GET /api/v1/admin/account-risk/config
func (h *AccountRiskHandler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()
	cfg := h.settingService.GetAccountRiskConfig(ctx)

	rules := []string{}
	if h.riskService != nil {
		rules = h.riskService.Rules()
	}
	response.Success(c, gin.H{
		"config": cfg,
		"rules":  rules,
	})
}

// UpdateConfig 覆盖保存风控配置（服务端会做归一化与区间收敛）。
// PUT /api/v1/admin/account-risk/config
func (h *AccountRiskHandler) UpdateConfig(c *gin.Context) {
	var req service.AccountRiskConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.settingService.SetAccountRiskConfig(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 返回落库后的实际值，便于前端立即回显归一化结果。
	response.Success(c, gin.H{
		"config": h.settingService.GetAccountRiskConfig(c.Request.Context()),
	})
}

// ListEvents 分页查询风控命中记录（读审计日志）。
// GET /api/v1/admin/account-risk/events?page=&page_size=&status=&action=&q=
//
// action 传完整动作名可精确过滤，传空则返回全部 security.risk.* 事件。
func (h *AccountRiskHandler) ListEvents(c *gin.Context) {
	if h.auditLogService == nil {
		response.InternalError(c, "audit log service is not available")
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}

	action := strings.TrimSpace(c.Query("action"))
	if action == "" {
		action = accountRiskEventActionPrefix
	}

	list, err := h.auditLogService.List(c.Request.Context(), &service.AuditLogFilter{
		Page:     page,
		PageSize: pageSize,
		Action:   action,
		Query:    strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 必须走 response.Paginated 统一成 {items,total,page,page_size,pages}：
	// 直接序列化 Service 的返回结构会得到 {Logs,Total,Page,PageSize}（大驼峰），
	// 与前端 AuditLogListResponse（PaginatedResponse.items）对不上，表格会空白。
	response.Paginated(c, list.Logs, int64(list.Total), list.Page, list.PageSize)
}
