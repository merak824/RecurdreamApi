import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import type { DashboardStats } from '@/types'
import DashboardView from '../DashboardView.vue'

const { getSnapshotV2, getUserUsageTrend, getUserSpendingRanking } = vi.hoisted(() => ({
  getSnapshotV2: vi.fn(),
  getUserUsageTrend: vi.fn(),
  getUserSpendingRanking: vi.fn()
}))

vi.mock('vue-chartjs', () => ({
  Line: { name: 'Line', props: ['data', 'options'], template: '<div class="line-chart" />' },
  Bar: { name: 'Bar', props: ['data', 'options'], template: '<div class="bar-chart" />' }
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getSnapshotV2,
      getUserUsageTrend,
      getUserSpendingRanking
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const createDashboardStats = (): DashboardStats => ({
  total_users: 0,
  today_new_users: 0,
  active_users: 0,
  hourly_active_users: 0,
  stats_updated_at: '',
  stats_stale: false,
  total_api_keys: 0,
  active_api_keys: 0,
  total_accounts: 0,
  normal_accounts: 0,
  error_accounts: 0,
  ratelimit_accounts: 0,
  overload_accounts: 0,
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  today_requests: 0,
  today_input_tokens: 0,
  today_output_tokens: 0,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 0,
  today_cost: 0,
  today_actual_cost: 0,
  average_duration_ms: 0,
  uptime: 0,
  rpm: 0,
  tpm: 0
})

// SelectStub 在上方定义；弹窗直通渲染，方便断言自定义区间。
const BaseDialogStub = {
  name: 'BaseDialog',
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div v-if="show" class="dialog-stub"><slot /><slot name="footer" /></div>'
}

const mountView = () =>
  mount(DashboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        LoadingSpinner: true,
        Icon: true,
        DateRangePicker: true,
        ModelDistributionChart: true,
        TokenUsageTrend: true,
        BaseDialog: BaseDialogStub
      }
    }
  })

// 主快照与利润快照分流：利润模块用自己的请求（include_profit: true）取值。
const withProfit = (profit: any) => {
  getSnapshotV2.mockImplementation((params: any) =>
    Promise.resolve(
      params?.include_profit
        ? { trend: [], models: [], profit }
        : { stats: createDashboardStats(), trend: [], models: [] }
    )
  )
}

const profitCalls = () =>
  getSnapshotV2.mock.calls.map((call) => call[0]).filter((params: any) => params?.include_profit === true)

describe('admin DashboardView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())

    getSnapshotV2.mockReset()
    getUserUsageTrend.mockReset()
    getUserSpendingRanking.mockReset()

    // 主快照不再带 profit，利润模块用自己的请求（include_profit: true）取值。
    getSnapshotV2.mockImplementation((params: any) =>
      Promise.resolve(
        params?.include_profit
          ? { trend: [], models: [], profit: undefined }
          : { stats: createDashboardStats(), trend: [], models: [] }
      )
    )
    getUserUsageTrend.mockResolvedValue({
      trend: [],
      start_date: '',
      end_date: '',
      granularity: 'hour'
    })
    getUserSpendingRanking.mockResolvedValue({
      ranking: [],
      total_actual_cost: 0,
      total_requests: 0,
      total_tokens: 0,
      start_date: '',
      end_date: ''
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('switches metrics and ignores a stale tokens response', async () => {
    let resolveTokens!: (value: any) => void
    getUserUsageTrend.mockImplementationOnce(() => new Promise(resolve => { resolveTokens = resolve }))
    getUserUsageTrend.mockResolvedValueOnce({ trend: [{ date: '2026-01-01', user_id: 2, username: 'spender', email: '', requests: 1, tokens: 10, cost: 5, actual_cost: 5 }] })
    const wrapper = mount(DashboardView, {
      global: { stubs: {
        AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
        Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true,
        TokenUsageTrend: true
      } }
    })
    await flushPromises()
    expect(getUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({ metric: 'tokens', limit: 12 }))
    await wrapper.findAll('button').find(button => button.text() === 'admin.dashboard.actualSpending')!.trigger('click')
    expect(getUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({ metric: 'actual_cost', limit: 12 }))
    await flushPromises()
    const chart = wrapper.findComponent({ name: 'Line' })
    expect(chart.props('data').datasets[0].data).toEqual([5])
    expect(chart.props('options').scales.y.ticks.callback(5)).toBe('$5.00')
    resolveTokens({ trend: [{ date: '2026-01-01', user_id: 1, username: 'tokens', email: '', requests: 1, tokens: 100, cost: 1, actual_cost: 1 }] })
    await flushPromises()
    expect(chart.props('data').datasets[0].data).toEqual([5])
  })

  it('uses last 24 hours as default dashboard range', async () => {
    mountView()

    await flushPromises()

    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour',
      include_profit: false
    }))
  })

  it('renders profit summary and switches dimension tabs from the profit snapshot', async () => {
    withProfit({
      generated_at: '2026-08-10T00:00:00Z',
      summary: {
        sales: 10,
        cost: 7,
        profit: 3,
        margin_percent: 30,
        requests: 2,
        tokens: 100,
        unknown_cost_count: 1,
        unverified_cost_count: 2,
        cost_source: 'mixed',
        verification_status: 'unverified'
      },
      trend: [],
      groups: [{ id: 1, name: 'default', requests: 2, tokens: 100, sales: 10, cost: 7, profit: 3, margin_percent: 30, cost_source: 'upstream_probe', verification_status: 'unverified' }],
      models: [{ name: 'gpt-test', requests: 2, tokens: 100, sales: 10, cost: 7, profit: 3, margin_percent: 30, cost_source: 'group_break_even_estimate', verification_status: 'unverified' }],
      accounts: []
    })

    const wrapper = mountView()
    await flushPromises()

    const monitor = wrapper.get('[data-testid="profit-monitor"]')
    expect(monitor.text()).toContain('admin.dashboard.profit')
    expect(monitor.text()).toContain('default')
    expect(monitor.text()).toContain('admin.dashboard.profitUpstreamProbe')
    expect(monitor.text()).toContain('admin.dashboard.profitPendingCostCount')

    const modelTab = monitor.findAll('button').find((button) => button.text() === 'admin.dashboard.profitModels')
    expect(modelTab).toBeDefined()
    await modelTab!.trigger('click')
    expect(monitor.text()).toContain('gpt-test')
    expect(monitor.text()).toContain('admin.dashboard.profitGroupEstimate')
  })

  it('renders automatic upstream reconciliation states and the mismatch amount', async () => {
    withProfit({
      generated_at: '2026-08-18T08:20:00Z',
      summary: {
        sales: 50,
        cost: 40,
        profit: 10,
        margin_percent: 20,
        requests: 5,
        tokens: 500,
        unknown_cost_count: 0,
        unverified_cost_count: 0,
        cost_source: 'upstream_probe',
        verification_status: 'unverified',
        reconciliation_status: 'difference',
        upstream_actual_cost: 42,
        reconciliation_difference: 2,
        reconciliation_difference_percent: 4.7619,
        reconciliation_observed_at: '2026-08-18T08:20:00Z'
      },
      trend: [],
      groups: [],
      models: [],
      accounts: [
        { id: 1, name: 'matched-account', requests: 1, tokens: 100, sales: 10, cost: 8, profit: 2, margin_percent: 20, cost_source: 'upstream_probe', verification_status: 'unverified', reconciliation_status: 'matched', upstream_actual_cost: 8.005, reconciliation_difference: 0.005 },
        { id: 2, name: 'difference-account', requests: 1, tokens: 100, sales: 10, cost: 8, profit: 2, margin_percent: 20, cost_source: 'upstream_probe', verification_status: 'unverified', reconciliation_status: 'difference', upstream_actual_cost: 10, reconciliation_difference: 2, reconciliation_difference_percent: 20 },
        { id: 3, name: 'pending-account', requests: 1, tokens: 100, sales: 10, cost: 8, profit: 2, margin_percent: 20, cost_source: 'upstream_probe', verification_status: 'unverified', reconciliation_status: 'pending' },
        { id: 4, name: 'unavailable-account', requests: 1, tokens: 100, sales: 10, cost: 8, profit: 2, margin_percent: 20, cost_source: 'upstream_probe', verification_status: 'unverified', reconciliation_status: 'unavailable' },
        { id: 5, name: 'estimated-account', requests: 1, tokens: 100, sales: 10, cost: 8, profit: 2, margin_percent: 20, cost_source: 'group_break_even_estimate', verification_status: 'unverified', reconciliation_status: 'estimated' }
      ]
    })

    const wrapper = mountView()
    await flushPromises()

    const monitor = wrapper.get('[data-testid="profit-monitor"]')
    const accountTab = monitor.findAll('button').find((button) => button.text() === 'admin.dashboard.profitAccounts')
    expect(accountTab).toBeDefined()
    await accountTab!.trigger('click')

    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationMatched')
    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationDifference')
    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationPending')
    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationUnavailable')
    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationEstimated')
    expect(monitor.text()).toContain('+$2.00')
    expect(monitor.text()).not.toContain('admin.dashboard.profitUnverified')
  })

  it('explains a missing start sample and refreshes after the next sampling time', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-19T07:26:00Z'))
    withProfit({
      generated_at: '2026-08-19T07:26:00Z',
      last_sample_at: '2026-08-19T07:20:13Z',
      next_sample_at: '2026-08-19T07:30:00Z',
      summary: {
        sales: 10,
        cost: 8,
        profit: 2,
        margin_percent: 20,
        requests: 1,
        tokens: 100,
        unknown_cost_count: 0,
        unverified_cost_count: 0,
        cost_source: 'upstream_probe',
        verification_status: 'unverified',
        reconciliation_status: 'missing_start'
      },
      trend: [],
      groups: [],
      models: [],
      accounts: [{
        id: 7,
        name: 'upstream-7',
        requests: 1,
        tokens: 100,
        sales: 10,
        cost: 8,
        profit: 2,
        margin_percent: 20,
        cost_source: 'upstream_probe',
        verification_status: 'unverified',
        reconciliation_status: 'missing_start',
        last_sample_at: '2026-08-19T07:20:13Z',
        next_sample_at: '2026-08-19T07:30:00Z'
      }]
    })

    const wrapper = mountView()
    await flushPromises()

    const monitor = wrapper.get('[data-testid="profit-monitor"]')
    expect(monitor.text()).toContain('admin.dashboard.profitReconciliationMissingStart')
    expect(monitor.text()).toContain('admin.dashboard.profitLastSample')
    expect(monitor.text()).toContain('admin.dashboard.profitNextSample')

    expect(profitCalls()).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(4 * 60 * 1000 + 21 * 1000)
    await flushPromises()
    expect(profitCalls()).toHaveLength(2)

    wrapper.unmount()
  })

  it('renders the profit trend as bars with margin on a second axis', async () => {
    withProfit({
      generated_at: '2026-08-10T00:00:00Z',
      summary: {
        sales: 10,
        cost: 7,
        profit: 3,
        margin_percent: 30,
        requests: 2,
        tokens: 100,
        unknown_cost_count: 0,
        unverified_cost_count: 0,
        cost_source: 'upstream_probe',
        verification_status: 'unverified'
      },
      trend: [
        { date: '2026-08-09T00:00:00Z', sales: 10, cost: 7, profit: 3, margin_percent: 30 },
        { date: '2026-08-10T00:00:00Z', sales: 4, cost: 5, profit: -1, margin_percent: -25 }
      ],
      groups: [],
      models: [],
      accounts: []
    })

    const wrapper = mountView()
    await flushPromises()

    const chart = wrapper.findComponent({ name: 'Bar' })
    expect(chart.exists()).toBe(true)
    const datasets = chart.props('data').datasets
    expect(datasets.map((dataset: any) => dataset.label)).toEqual([
      'admin.dashboard.profitSales',
      'admin.dashboard.profitCost',
      'admin.dashboard.profitMargin'
    ])
    expect(datasets[0].yAxisID).toBe('y')
    expect(datasets[1].yAxisID).toBe('y')
    expect(datasets[2].type).toBe('line')
    expect(datasets[2].yAxisID).toBe('margin')
    expect(datasets[2].data).toEqual([30, -25])
    expect(chart.props('options').scales.margin.ticks.callback(30)).toBe('30%')
    expect(chart.props('options').scales.y.ticks.callback(7)).toBe('$7.00')
  })

  it('switches the whole profit module to the last 7 days', async () => {
    withProfit({
      generated_at: '2026-08-10T00:00:00Z',
      summary: {
        sales: 10, cost: 7, profit: 3, margin_percent: 30, requests: 2, tokens: 100,
        unknown_cost_count: 0, unverified_cost_count: 0, cost_source: 'upstream_probe', verification_status: 'unverified'
      },
      trend: [],
      groups: [{ id: 1, name: 'default', requests: 2, tokens: 100, sales: 10, cost: 7, profit: 3, margin_percent: 30, cost_source: 'upstream_probe', verification_status: 'unverified' }],
      models: [],
      accounts: []
    })

    const wrapper = mountView()
    await flushPromises()
    expect(profitCalls()).toHaveLength(1)
    expect(wrapper.get('[data-testid="profit-monitor"]').text()).toContain('default')

    await wrapper.get('[data-testid="profit-range-last7"]').trigger('click')
    await flushPromises()

    expect(profitCalls()).toHaveLength(2)
    const request = profitCalls()[1] as any
    const now = new Date()
    const sixDaysAgo = new Date(now.getTime() - 6 * 24 * 60 * 60 * 1000)
    expect(request.start_date).toBe(formatLocalDate(sixDaysAgo))
    expect(request.end_date).toBe(formatLocalDate(now))
    expect(request.granularity).toBe('day')
    expect(request.include_trend).toBe(false)
    expect(request.include_stats).toBe(false)
  })

  it('applies a custom range chosen in the dialog', async () => {
    withProfit({
      generated_at: '2026-08-10T00:00:00Z',
      summary: {
        sales: 10, cost: 7, profit: 3, margin_percent: 30, requests: 2, tokens: 100,
        unknown_cost_count: 0, unverified_cost_count: 0, cost_source: 'upstream_probe', verification_status: 'unverified'
      },
      trend: [],
      groups: [],
      models: [],
      accounts: []
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="profit-range-custom"]').trigger('click')
    await wrapper.get('[data-testid="profit-custom-start"]').setValue('2026-09-01')
    await wrapper.get('[data-testid="profit-custom-end"]').setValue('2026-09-20')
    await wrapper.get('[data-testid="profit-custom-apply"]').trigger('click')
    await flushPromises()

    const request = profitCalls().at(-1) as any
    expect(request.start_date).toBe('2026-09-01')
    expect(request.end_date).toBe('2026-09-20')
    expect(request.granularity).toBe('day')
  })
})
