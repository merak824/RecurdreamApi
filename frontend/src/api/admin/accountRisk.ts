/**
 * 账号风控（账号异常检测与自动封禁）管理 API。
 *
 * 命中记录不单独建表：后端把所有命中写入管理面审计日志
 * （action 前缀 security.risk.），这里直接复用审计日志的分页结构。
 */

import { apiClient } from '../client'
import type { AuditLog, AuditLogQuery, AuditLogListResponse } from './audit'

/** 规则命中后的处置动作。 */
export type AccountRiskAction = 'flag' | 'ban' | 'block'

export interface AccountRiskRegisterProbeConfig {
  enabled: boolean
  /** 注册后多少分钟内仍按"新账号"对待。 */
  window_minutes: number
  action: AccountRiskAction
}

export interface AccountRiskMultiAccountConfig {
  enabled: boolean
  /** 窗口期内同一 IP 允许成功注册的账号数。 */
  max_accounts_per_ip: number
  window_hours: number
  action: AccountRiskAction
}

export interface AccountRiskSessionAbuseConfig {
  enabled: boolean
  /** 窗口期内允许的指纹不匹配次数。 */
  max_mismatches: number
  window_hours: number
  action: AccountRiskAction
}

export interface AccountRiskLoginGuardConfig {
  enabled: boolean
  max_ip_failures: number
  ip_window_minutes: number
  max_account_failures: number
  account_window_minutes: number
  lock_minutes: number
}

export interface AccountRiskConfig {
  enabled: boolean
  register_probe: AccountRiskRegisterProbeConfig
  multi_account: AccountRiskMultiAccountConfig
  session_abuse: AccountRiskSessionAbuseConfig
  login_guard: AccountRiskLoginGuardConfig
  allowlisted_user_ids: number[]
}

export interface AccountRiskConfigResponse {
  config: AccountRiskConfig
  rules: string[]
}

export interface AccountRiskEventQuery {
  page?: number
  page_size?: number
  /** 精确动作名；留空返回全部 security.risk.* 事件。 */
  action?: string
  q?: string
}

const accountRiskAPI = {
  /** 读取当前生效配置（含已注册规则名）。 */
  async getConfig(): Promise<AccountRiskConfigResponse> {
    const { data } = await apiClient.get('/admin/account-risk/config')
    return data
  },

  /** 覆盖保存配置；后端会做归一化与区间收敛，返回落库后的实际值。 */
  async updateConfig(config: AccountRiskConfig): Promise<{ config: AccountRiskConfig }> {
    const { data } = await apiClient.put('/admin/account-risk/config', config)
    return data
  },

  /** 分页查询命中记录（读审计日志）。 */
  async listEvents(params: AccountRiskEventQuery): Promise<AuditLogListResponse> {
    const { data } = await apiClient.get('/admin/account-risk/events', { params })
    return data
  }
}

export default accountRiskAPI

export type { AuditLog, AuditLogQuery, AuditLogListResponse }
