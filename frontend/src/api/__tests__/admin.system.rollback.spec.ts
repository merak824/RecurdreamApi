import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import {
  getRollbackVersions,
  getUpdateStatus,
  performUpdate,
  rollback,
  type RollbackVersionInfo
} from '@/api/admin/system'

describe('admin system rollback API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('getRollbackVersions fetches the rollback version list', async () => {
    const versions: RollbackVersionInfo[] = [
      {
        version: '0.1.146',
        published_at: '2026-07-07T00:00:00Z',
        html_url: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.146'
      }
    ]
    get.mockResolvedValue({ data: { versions } })

    const result = await getRollbackVersions()

    expect(get).toHaveBeenCalledWith('/admin/system/rollback-versions')
    expect(result.versions).toEqual(versions)
  })

  it('rollback posts the target version in the request body', async () => {
    post.mockResolvedValue({ data: { message: 'ok', need_restart: true } })

    const result = await rollback('0.1.146')

    expect(post).toHaveBeenCalledWith(
      '/admin/system/rollback',
      { version: '0.1.146' },
      { timeout: 15 * 60 * 1000 }
    )
    expect(result.need_restart).toBe(true)
  })

  it('rollback without a version posts no body (legacy backup rollback)', async () => {
    post.mockResolvedValue({ data: { message: 'ok', need_restart: true } })

    await rollback()

    expect(post).toHaveBeenCalledWith(
      '/admin/system/rollback',
      undefined,
      { timeout: 15 * 60 * 1000 }
    )
  })

  it('performUpdate accepts an asynchronous Docker operation', async () => {
    post.mockResolvedValue({
      data: {
        operation_id: 'update-abc',
        status: 'queued',
        version: '0.2.5',
        message: 'update accepted',
        need_restart: false
      }
    })

    const result = await performUpdate()

    expect(post).toHaveBeenCalledWith('/admin/system/update', undefined, {
      timeout: 15 * 60 * 1000
    })
    expect(result.operation_id).toBe('update-abc')
    expect(result.status).toBe('queued')
  })

  it('getUpdateStatus fetches the host-agent operation state', async () => {
    get.mockResolvedValue({
      data: {
        operation_id: 'update-abc',
        status: 'succeeded',
        version: '0.2.5',
        message: 'deployment completed',
        need_restart: false
      }
    })

    const result = await getUpdateStatus('update-abc')

    expect(get).toHaveBeenCalledWith('/admin/system/update/update-abc', {
      timeout: 15 * 1000
    })
    expect(result.status).toBe('succeeded')
  })
})
