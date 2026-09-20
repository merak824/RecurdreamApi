import { describe, expect, it } from 'vitest'

import { pollUpdateStatus } from '@/utils/updatePolling'

describe('pollUpdateStatus', () => {
  it('retries a transient status failure and resolves after deployment succeeds', async () => {
    const states = [
      { operation_id: 'update-abc', status: 'queued', version: '0.2.5', need_restart: false },
      new Error('temporary nginx reconnect'),
      { operation_id: 'update-abc', status: 'running', version: '0.2.5', need_restart: false },
      {
        operation_id: 'update-abc',
        status: 'succeeded',
        version: '0.2.5',
        message: 'deployment completed',
        need_restart: false
      }
    ]
    let now = 0
    let calls = 0

    const result = await pollUpdateStatus('update-abc', {
      getStatus: async () => {
        const state = states[calls++]
        if (state instanceof Error) throw state
        return state
      },
      intervalMs: 3000,
      timeoutMs: 20000,
      sleep: async (delay) => {
        now += delay
      },
      now: () => now
    })

    expect(result.status).toBe('succeeded')
    expect(calls).toBe(4)
  })

  it('returns a terminal failure without continuing to poll', async () => {
    let calls = 0

    const result = await pollUpdateStatus('update-abc', {
      getStatus: async () => {
        calls += 1
        return {
          operation_id: 'update-abc',
          status: 'failed',
          version: '0.2.5',
          message: 'deployment failed: health check timed out',
          need_restart: false
        }
      },
      sleep: async () => undefined
    })

    expect(result.status).toBe('failed')
    expect(result.message).toContain('health check timed out')
    expect(calls).toBe(1)
  })
})
