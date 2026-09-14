import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get },
}))

import { listPublic } from '@/api/channelMonitor'

describe('public channel monitor API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads the unauthenticated redacted status feed', async () => {
    const controller = new AbortController()
    const payload = {
      items: [
        {
          name: 'GPT PLUS',
          provider: 'openai',
          primary_model: 'gpt-5.6-sol',
          primary_status: 'operational',
          primary_latency_ms: 1200,
          primary_ping_latency_ms: 90,
          availability_7d: 99.5,
          timeline: [],
        },
      ],
    }
    get.mockResolvedValue({ data: payload })

    await expect(listPublic({ signal: controller.signal })).resolves.toEqual(payload)
    expect(get).toHaveBeenCalledWith('/public/channel-monitors', {
      signal: controller.signal,
    })
  })
})
