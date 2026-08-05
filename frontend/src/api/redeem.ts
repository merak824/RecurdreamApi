/**
 * Redeem code API endpoints
 * Handles redeem code redemption for users
 */

import { apiClient } from './client'
import type { RedeemCodeRequest } from '@/types'

export interface RedeemHistoryItem {
  id: number
  code: string
  type: string
  value: number
  status: string
  used_at: string
  created_at: string
  // Notes from admin for admin_balance/admin_concurrency types
  notes?: string
  // Subscription-specific fields
  group_id?: number
  validity_days?: number
  group?: {
    id: number
    name: string
  }
}

export type BalanceHistoryType =
  | 'balance'
  | 'admin_balance'
  | 'affiliate_balance'
  | 'red_packet_reward'

export interface BalanceHistoryItem {
  id: string
  type: BalanceHistoryType
  amount: number
  occurred_at: string
  reference: string
  description: string
}

export interface BalanceHistoryResponse {
  items: BalanceHistoryItem[]
  total: number
  page: number
  page_size: number
  pages: number
}

/**
 * Redeem a code
 * @param code - Redeem code string
 * @returns Redemption result with updated balance or concurrency
 */
export async function redeem(code: string): Promise<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
}> {
  const payload: RedeemCodeRequest = { code }

  const { data } = await apiClient.post<{
    message: string
    type: string
    value: number
    new_balance?: number
    new_concurrency?: number
  }>('/redeem', payload)

  return data
}

/**
 * Get user's redemption history
 * @returns List of redeemed codes
 */
export async function getHistory(): Promise<RedeemHistoryItem[]> {
  const { data } = await apiClient.get<RedeemHistoryItem[]>('/redeem/history')
  return data
}

export async function getBalanceHistory(
  page = 1,
  pageSize = 20,
  type?: BalanceHistoryType
): Promise<BalanceHistoryResponse> {
  const { data } = await apiClient.get<BalanceHistoryResponse>('/balance-history', {
    params: {
      page,
      page_size: pageSize,
      ...(type ? { type } : {})
    }
  })
  return data
}

export const redeemAPI = {
  redeem,
  getHistory,
  getBalanceHistory
}

export default redeemAPI
