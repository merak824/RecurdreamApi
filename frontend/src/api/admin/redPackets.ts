import { apiClient } from '../client'
import type { RedPacketActivity, RedPacketType } from '../redPackets'

export interface RedPacketDraft {
  name: string
  message: string
  packet_type: RedPacketType
  total_amount_cents: number
  target_participants: number
  winner_count: number
}

export interface RedPacketAdminPage {
  items: RedPacketActivity[]
  total: number
  page: number
  page_size: number
  pages: number
}

export async function list(page = 1, pageSize = 20): Promise<RedPacketAdminPage> {
  const { data } = await apiClient.get<RedPacketAdminPage>('/admin/red-packets', {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function create(draft: RedPacketDraft): Promise<RedPacketActivity> {
  const { data } = await apiClient.post<RedPacketActivity>('/admin/red-packets', draft)
  return data
}

export async function update(id: number, draft: RedPacketDraft): Promise<RedPacketActivity> {
  const { data } = await apiClient.put<RedPacketActivity>(`/admin/red-packets/${id}`, draft)
  return data
}

export async function publish(id: number): Promise<RedPacketActivity> {
  const { data } = await apiClient.post<RedPacketActivity>(`/admin/red-packets/${id}/publish`)
  return data
}

export async function cancel(id: number): Promise<void> {
  await apiClient.post(`/admin/red-packets/${id}/cancel`)
}

export async function exportActivity(id: number): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/admin/red-packets/${id}/export`, {
    responseType: 'blob'
  })
  return data
}

const redPacketsAPI = {
  list,
  create,
  update,
  publish,
  cancel,
  export: exportActivity
}

export default redPacketsAPI
