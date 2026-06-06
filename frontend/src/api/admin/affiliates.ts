/**
 * Admin Affiliate API endpoints
 * Manage per-user affiliate (邀请返利) configurations:
 * exclusive invite codes (overrides aff_code) and exclusive rebate rates.
 */

import { apiClient } from '../client'
import type { AffiliateWithdrawal, PaginatedResponse } from '@/types'

export interface AffiliateAdminEntry {
  user_id: number
  email: string
  username: string
  role: string
  aff_code: string
  aff_code_custom: boolean
  aff_rebate_rate_percent?: number | null
  aff_count: number
}

export interface ListAffiliateUsersParams {
  page?: number
  page_size?: number
  search?: string
}

export interface ListAffiliateRecordsParams {
  page?: number
  page_size?: number
  search?: string
  start_at?: string
  end_at?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  timezone?: string
}

export interface AffiliateInviteRecord {
  inviter_id: number
  inviter_email: string
  inviter_username: string
  invitee_id: number
  invitee_email: string
  invitee_username: string
  aff_code: string
  total_rebate: number
  created_at: string
}

export interface AffiliateRebateRecord {
  order_id: number
  out_trade_no: string
  inviter_id: number
  inviter_email: string
  inviter_username: string
  invitee_id: number
  invitee_email: string
  invitee_username: string
  order_amount: number
  pay_amount: number
  rebate_amount: number
  payment_type: string
  order_status: string
  created_at: string
}

export interface AffiliateTransferRecord {
  ledger_id: number
  user_id: number
  user_email: string
  username: string
  amount: number
  balance_after?: number | null
  available_quota_after?: number | null
  frozen_quota_after?: number | null
  history_quota_after?: number | null
  snapshot_available: boolean
  created_at: string
}

export interface AffiliateWithdrawalRecord {
  id: number
  record_type: 'transfer' | 'withdrawal' | string
  destination: 'balance' | 'alipay_wechat' | string
  user_id: number
  user_email?: string
  username?: string
  amount: number
  status: 'pending' | 'paid' | 'rejected' | 'completed' | string
  collection_qr_data?: string
  collection_qr_mime?: string
  collection_qr_size?: number
  payment_proof_data?: string
  payment_proof_mime?: string
  payment_proof_size?: number
  reject_reason?: string
  admin_note?: string
  processed_by?: number | null
  processed_at?: string | null
  balance_after?: number | null
  available_quota_after?: number | null
  frozen_quota_after?: number | null
  history_quota_after?: number | null
  snapshot_available?: boolean
  created_at: string
  updated_at: string
}

export interface AffiliateUserOverview {
  user_id: number
  email: string
  username: string
  aff_code: string
  rebate_rate_percent: number
  invited_count: number
  rebated_invitee_count: number
  available_quota: number
  history_quota: number
}

export interface AffiliateAgentDailyUsage {
  date: string
  total_tokens: number
  request_count: number
  actual_cost: number
}

export interface AffiliateAgentInviteeUsage {
  user_id: number
  email: string
  username: string
  total_tokens: number
  average_daily_tokens: number
  request_count: number
  actual_cost: number
}

export interface AffiliateAgentUsageSummary {
  agent_user_id: number
  week_start: string
  week_end: string
  total_tokens: number
  average_daily_tokens: number
  request_count: number
  actual_cost: number
  invitee_count: number
  active_invitee_count: number
  suggested_rebate_rate_percent: number
  daily: AffiliateAgentDailyUsage[]
  invitees: AffiliateAgentInviteeUsage[]
}

export interface GetAffiliateAgentUsageParams {
  start_at?: string
  end_at?: string
  timezone?: string
}

export interface UpdateAffiliateUserRequest {
  aff_code?: string
  aff_rebate_rate_percent?: number | null
  /** Set true to explicitly clear the per-user rate (sets it to NULL). */
  clear_rebate_rate?: boolean
}

export interface BatchSetRateRequest {
  user_ids: number[]
  aff_rebate_rate_percent?: number | null
  /** Set true to clear rates instead of setting. */
  clear?: boolean
}

export interface MarkWithdrawalPaidRequest {
  payment_proof_data?: string
  admin_note?: string
}

export interface RejectWithdrawalRequest {
  reject_reason?: string
  admin_note?: string
}

export interface SimpleUser {
  id: number
  email: string
  username: string
}

export async function listUsers(
  params: ListAffiliateUsersParams = {},
): Promise<PaginatedResponse<AffiliateAdminEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateAdminEntry>>(
    '/admin/affiliates/users',
    {
      params: {
        page: params.page ?? 1,
        page_size: params.page_size ?? 20,
        search: params.search ?? '',
      },
    },
  )
  return data
}

export async function lookupUsers(q: string): Promise<SimpleUser[]> {
  const { data } = await apiClient.get<SimpleUser[]>(
    '/admin/affiliates/users/lookup',
    { params: { q } },
  )
  return data
}

export async function updateUserSettings(
  userId: number,
  payload: UpdateAffiliateUserRequest,
): Promise<{ user_id: number }> {
  const { data } = await apiClient.put<{ user_id: number }>(
    `/admin/affiliates/users/${userId}`,
    payload,
  )
  return data
}

export async function clearUserSettings(
  userId: number,
): Promise<{ user_id: number }> {
  const { data } = await apiClient.delete<{ user_id: number }>(
    `/admin/affiliates/users/${userId}`,
  )
  return data
}

export async function batchSetRate(
  payload: BatchSetRateRequest,
): Promise<{ affected: number }> {
  const { data } = await apiClient.post<{ affected: number }>(
    '/admin/affiliates/users/batch-rate',
    payload,
  )
  return data
}

function recordParams(params: ListAffiliateRecordsParams = {}) {
  return {
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
    search: params.search ?? '',
    start_at: params.start_at || undefined,
    end_at: params.end_at || undefined,
    sort_by: params.sort_by || undefined,
    sort_order: params.sort_order || undefined,
    timezone: params.timezone || undefined,
  }
}

export async function listInviteRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateInviteRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateInviteRecord>>(
    '/admin/affiliates/invites',
    { params: recordParams(params) },
  )
  return data
}

export async function listRebateRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateRebateRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateRebateRecord>>(
    '/admin/affiliates/rebates',
    { params: recordParams(params) },
  )
  return data
}

export async function listTransferRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateTransferRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateTransferRecord>>(
    '/admin/affiliates/transfers',
    { params: recordParams(params) },
  )
  return data
}

export async function listWithdrawalRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateWithdrawalRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateWithdrawalRecord>>(
    '/admin/affiliates/withdrawals',
    { params: recordParams(params) },
  )
  return data
}

export async function markWithdrawalPaid(
  id: number,
  payload: MarkWithdrawalPaidRequest,
): Promise<AffiliateWithdrawal> {
  const { data } = await apiClient.post<AffiliateWithdrawal>(
    `/admin/affiliates/withdrawals/${id}/paid`,
    payload,
  )
  return data
}

export async function rejectWithdrawal(
  id: number,
  payload: RejectWithdrawalRequest,
): Promise<AffiliateWithdrawal> {
  const { data } = await apiClient.post<AffiliateWithdrawal>(
    `/admin/affiliates/withdrawals/${id}/reject`,
    payload,
  )
  return data
}

export async function getUserOverview(
  userId: number,
): Promise<AffiliateUserOverview> {
  const { data } = await apiClient.get<AffiliateUserOverview>(
    `/admin/affiliates/users/${userId}/overview`,
  )
  return data
}

export async function getAgentUsage(
  userId: number,
  params: GetAffiliateAgentUsageParams = {},
): Promise<AffiliateAgentUsageSummary> {
  const { data } = await apiClient.get<AffiliateAgentUsageSummary>(
    `/admin/affiliates/users/${userId}/usage`,
    {
      params: {
        start_at: params.start_at || undefined,
        end_at: params.end_at || undefined,
        timezone: params.timezone || undefined,
      },
    },
  )
  return data
}

export const affiliatesAPI = {
  listUsers,
  lookupUsers,
  updateUserSettings,
  clearUserSettings,
  batchSetRate,
  listInviteRecords,
  listRebateRecords,
  listTransferRecords,
  listWithdrawalRecords,
  markWithdrawalPaid,
  rejectWithdrawal,
  getUserOverview,
  getAgentUsage,
}

export default affiliatesAPI
