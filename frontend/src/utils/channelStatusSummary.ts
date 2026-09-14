import type { MonitorStatus } from '@/api/admin/channelMonitor'

export type ChannelStatusSummaryTone = 'operational' | 'degraded' | 'failed' | 'neutral'

export interface ChannelStatusSummary {
  label: string
  tone: ChannelStatusSummaryTone
}

function isFailedStatus(status: MonitorStatus | ''): boolean {
  return status === 'failed' || status === 'error'
}

export function summarizeChannelStatus(
  statuses: readonly (MonitorStatus | '')[],
): ChannelStatusSummary {
  if (statuses.length === 0) {
    return { label: '暂无监控', tone: 'neutral' }
  }

  const hasOperational = statuses.some(status => status === 'operational')
  const allFailed = statuses.every(isFailedStatus)

  if (allFailed) {
    return { label: '全部异常', tone: 'failed' }
  }

  if (hasOperational && statuses.some(status => status !== 'operational')) {
    return { label: '部分正常', tone: 'degraded' }
  }

  if (statuses.every(status => status === 'operational')) {
    return { label: '服务正常', tone: 'operational' }
  }

  if (statuses.some(status => status === 'degraded')) {
    return { label: '性能波动', tone: 'degraded' }
  }

  return { label: '部分待检测', tone: 'neutral' }
}
