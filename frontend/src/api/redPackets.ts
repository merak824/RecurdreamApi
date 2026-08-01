import { apiClient } from './client'

export type RedPacketType = 'lucky' | 'fixed'
export type RedPacketStatus = 'draft' | 'active' | 'drawing' | 'completed' | 'canceled'
export type RedPacketQualification = 'recharge' | 'invitation_points'

export interface RedPacketActivity {
  id: number
  period_no: number
  name: string
  message: string
  packet_type: RedPacketType
  total_amount_cents: number
  target_participants: number
  winner_count: number
  participant_count: number
  status: RedPacketStatus
  recharge_threshold_cents: number
  invitation_points_threshold: number
  invitation_points_cost: number
  recharge_priority: boolean
  published_at?: string
  drawing_at?: string
  completed_at?: string
  canceled_at?: string
  created_at?: string
  updated_at?: string
  has_participated: boolean
  my_qualification_type?: RedPacketQualification
  my_reward_cents?: number
}

export interface RedPacketWinner {
  masked_username: string
  amount_cents: number
  is_luckiest: boolean
  is_current_user: boolean
  credited_at?: string
}

export interface RedPacketActivityDetail {
  activity: RedPacketActivity
  winners: RedPacketWinner[]
}

export interface RedPacketEligibility {
  net_recharge_cents: number
  recharge_threshold_cents: number
  lottery_points: number
  invitation_points_required: number
  invitation_points_cost: number
  recharge_qualified: boolean
  points_qualified: boolean
  preferred_qualification?: RedPacketQualification
  recharge_shortfall_cents: number
  points_shortfall: number
}

export interface RedPacketReward {
  activity_id: number
  period_no: number
  activity_name: string
  amount_cents: number
  credited_at: string
}

export interface RedPacketParticipationResult {
  activity: RedPacketActivity
  qualification_type: RedPacketQualification
  points_spent: number
  lottery_points: number
  triggered_drawing: boolean
}

export async function getCurrent(): Promise<RedPacketActivity | null> {
  const { data } = await apiClient.get<RedPacketActivity | null>('/red-packets/current')
  return data
}

export async function getEligibility(): Promise<RedPacketEligibility> {
  const { data } = await apiClient.get<RedPacketEligibility>('/red-packets/eligibility')
  return data
}

export async function getRecent(limit = 5): Promise<RedPacketActivity[]> {
  const { data } = await apiClient.get<RedPacketActivity[]>('/red-packets/recent', { params: { limit } })
  return data
}

export async function getRewards(limit = 50): Promise<RedPacketReward[]> {
  const { data } = await apiClient.get<RedPacketReward[]>('/red-packets/rewards', { params: { limit } })
  return data
}

export async function getActivity(id: number): Promise<RedPacketActivityDetail> {
  const { data } = await apiClient.get<RedPacketActivityDetail>(`/red-packets/${id}`)
  return data
}

export async function participate(id: number): Promise<RedPacketParticipationResult> {
  const { data } = await apiClient.post<RedPacketParticipationResult>(`/red-packets/${id}/participate`)
  return data
}

export const redPacketsAPI = {
  getCurrent,
  getEligibility,
  getRecent,
  getRewards,
  getActivity,
  participate
}

export default redPacketsAPI
