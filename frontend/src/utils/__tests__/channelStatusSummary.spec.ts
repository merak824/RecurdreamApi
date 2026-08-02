import { describe, expect, it } from 'vitest'

import { summarizeChannelStatus } from '../channelStatusSummary'

describe('summarizeChannelStatus', () => {
  it('shows 部分正常 when at least one channel is operational', () => {
    expect(summarizeChannelStatus(['operational', 'failed'])).toEqual({
      label: '部分正常',
      tone: 'degraded',
    })

    expect(summarizeChannelStatus(['operational', 'degraded'])).toEqual({
      label: '部分正常',
      tone: 'degraded',
    })
  })

  it('shows 全部异常 only when every channel is failed or errored', () => {
    expect(summarizeChannelStatus(['failed', 'error'])).toEqual({
      label: '全部异常',
      tone: 'failed',
    })
  })

  it('shows 服务正常 when every channel is operational', () => {
    expect(summarizeChannelStatus(['operational', 'operational'])).toEqual({
      label: '服务正常',
      tone: 'operational',
    })
  })
})
