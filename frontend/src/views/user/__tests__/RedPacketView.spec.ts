import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RedPacketView from '../RedPacketView.vue'

const { getCurrent, getEligibility, getRecent, getRewards, getActivity, participate } = vi.hoisted(() => ({
  getCurrent: vi.fn(),
  getEligibility: vi.fn(),
  getRecent: vi.fn(),
  getRewards: vi.fn(),
  getActivity: vi.fn(),
  participate: vi.fn()
}))

vi.mock('@/api/redPackets', () => ({
  redPacketsAPI: { getCurrent, getEligibility, getRecent, getRewards, getActivity, participate }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => ({
      'redPacket.title': '红包活动',
      'redPacket.subtitle': '第几期由后台发布，达到人数后自动开奖',
      'redPacket.qualificationEither': '充值达标或邀请积分满足其一即可参与',
      'redPacket.rechargeQualified': '充值资格已满足',
      'redPacket.rechargeRequired': '充值满 $1.00',
      'redPacket.pointsQualified': '邀请积分可用',
      'redPacket.pointsRequired': '需要 2 积分',
      'redPacket.pointsCost': '参与消耗 2 积分',
      'redPacket.pointsRemaining': `剩余 ${params?.count ?? 0} 积分`,
      'redPacket.history': '往期活动',
      'redPacket.viewRecords': '查看开奖记录',
      'redPacket.records': '开奖记录',
      'redPacket.me': '\u6211',
      'redPacket.participate': '立即参与',
      'redPacket.participating': '参与中',
      'redPacket.current': '当前活动',
      'redPacket.rules': '活动规则',
      'redPacket.latestFive': '最近五期',
      'common.close': '关闭'
      }[key] ?? key)
    })
  }
})

const activity = {
  id: 1,
  period_no: 8,
  name: '第8期红包',
  message: '达到人数自动开奖',
  packet_type: 'lucky',
  total_amount_cents: 1000,
  target_participants: 10,
  winner_count: 3,
  participant_count: 4,
  status: 'active',
  recharge_threshold_cents: 100,
  invitation_points_threshold: 2,
  invitation_points_cost: 2,
  recharge_priority: true,
  has_participated: false
}

const eligibility = {
  net_recharge_cents: 100,
  recharge_threshold_cents: 100,
  lottery_points: 5,
  invitation_points_required: 2,
  invitation_points_cost: 2,
  recharge_qualified: true,
  points_qualified: true,
  preferred_qualification: 'recharge',
  recharge_shortfall_cents: 0,
  points_shortfall: 0
}

const history = Array.from({ length: 5 }, (_, index) => ({
  ...activity,
  id: index + 1,
  period_no: 8 - index,
  name: `第${8 - index}期红包`,
  status: 'completed',
  participant_count: 10,
  has_participated: index === 0
}))

describe('RedPacketView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getCurrent.mockResolvedValue(activity)
    getEligibility.mockResolvedValue(eligibility)
    getRecent.mockResolvedValue(history)
    getRewards.mockResolvedValue([])
    getActivity.mockResolvedValue({ activity, winners: [] })
    participate.mockResolvedValue({
      activity: { ...activity, has_participated: true, participant_count: 5 },
      qualification_type: 'invitation_points',
      points_spent: 2,
      lottery_points: 3,
      triggered_drawing: false
    })
  })

  it('shows recharge-or-invitation eligibility and exactly five history rows', async () => {
    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('充值达标或邀请积分满足其一即可参与')
    expect(wrapper.get('[data-test="current-activity-name"]').text()).toBe(activity.name)
    expect(wrapper.get('[data-test="current-activity-message"]').text()).toBe(activity.message)
    expect(wrapper.get('[data-test="history-activity-name"]').text()).toBe(history[0].name)
    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.text()).not.toContain('¥')
    expect(wrapper.findAll('[data-test="history-row"]')).toHaveLength(5)
    expect(wrapper.text()).not.toContain('最近参与')
    expect(wrapper.find('[data-test="red-packet-workspace"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="current-activity-card"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-test="activity-stat"]')).toHaveLength(3)
    expect(wrapper.find('[data-test="eligibility-panel"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="rules-panel"]').exists()).toBe(true)
    expect(wrapper.find('table[data-test="history-table"]').exists()).toBe(true)
    expect(getRecent).toHaveBeenCalledWith(5)
  })

  it('consumes two invitation points and refreshes the remaining point balance', async () => {
    getEligibility.mockResolvedValue({ ...eligibility, recharge_qualified: false, preferred_qualification: 'invitation_points' })
    const wrapper = mount(RedPacketView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    await flushPromises()

    await wrapper.find('[data-test="participate-button"]').trigger('click')
    await flushPromises()

    expect(participate).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('参与消耗 2 积分')
    expect(wrapper.text()).toContain('剩余 3 积分')
  })

  it('does not poll after a participation that did not trigger drawing', async () => {
    vi.useFakeTimers()
    const wrapper = mount(RedPacketView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    try {
      await flushPromises()
      await wrapper.find('[data-test="participate-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(6000)

      expect(participate).toHaveBeenCalledWith(activity.id)
      expect(getActivity).not.toHaveBeenCalled()
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('shows a drawing entry as pending instead of not won', async () => {
    const drawingActivity = { ...activity, status: 'drawing', has_participated: true, participant_count: 10 }
    getCurrent.mockResolvedValue(drawingActivity)
    getRecent.mockResolvedValue([drawingActivity])
    getRewards.mockResolvedValue([])

    const wrapper = mount(RedPacketView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    await flushPromises()

    const rowText = wrapper.find('[data-test="history-row"]').text()
    expect(rowText).toContain('redPacket.status.drawing')
    expect(rowText).not.toContain('redPacket.notWon')
    wrapper.unmount()
  })

  it('does not poll an already joined drawing after opening the page', async () => {
    vi.useFakeTimers()
    const drawingActivity = { ...activity, status: 'drawing', has_participated: true, participant_count: 10 }
    getCurrent.mockResolvedValue(drawingActivity)
    getRecent.mockResolvedValue([drawingActivity])

    const wrapper = mount(RedPacketView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    try {
      await flushPromises()
      await vi.advanceTimersByTimeAsync(6000)

      expect(getActivity).not.toHaveBeenCalled()
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('polls after this participation triggers drawing and refreshes the reward result', async () => {
    vi.useFakeTimers()
    const readyActivity = { ...activity, participant_count: 9 }
    const drawingActivity = { ...readyActivity, status: 'drawing', has_participated: true, participant_count: 10 }
    const completedActivity = { ...drawingActivity, status: 'completed', my_reward_cents: 1000 }
    const reward = {
      activity_id: completedActivity.id,
      period_no: completedActivity.period_no,
      activity_name: completedActivity.name,
      amount_cents: 1000,
      credited_at: '2026-07-31T22:00:00.000Z'
    }

    getCurrent.mockResolvedValue(readyActivity)
    participate.mockResolvedValue({
      activity: drawingActivity,
      qualification_type: 'recharge',
      points_spent: 0,
      lottery_points: eligibility.lottery_points,
      triggered_drawing: true
    })
    getRecent.mockReset()
    getRecent
      .mockResolvedValueOnce([readyActivity])
      .mockResolvedValueOnce([drawingActivity])
      .mockResolvedValueOnce([completedActivity])
    getRewards.mockReset()
    getRewards.mockResolvedValueOnce([]).mockResolvedValueOnce([reward])
    getActivity.mockReset().mockResolvedValue({ activity: completedActivity, winners: [] })

    const wrapper = mount(RedPacketView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    try {
      await flushPromises()
      await wrapper.find('[data-test="participate-button"]').trigger('click')
      await flushPromises()

      await vi.advanceTimersByTimeAsync(2000)
      await flushPromises()

      expect(getActivity).toHaveBeenCalledWith(completedActivity.id)
      expect(getRewards).toHaveBeenCalledTimes(2)
      expect(wrapper.find('[data-test="history-row"]').text()).toContain('+$10.00')
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('closes records with Escape and restores focus to the detail trigger', async () => {
    const wrapper = mount(RedPacketView, {
      attachTo: document.body,
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    await flushPromises()

    const trigger = wrapper.find('[data-test="history-detail-1"]').element as HTMLButtonElement
    trigger.focus()
    await wrapper.find('[data-test="history-detail-1"]').trigger('click')
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).not.toBeNull()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
	await new Promise(resolve => window.setTimeout(resolve, 350))

    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
    wrapper.unmount()
  })

  it('marks only the signed-in winner and keeps detail statistics responsive', async () => {
    getActivity.mockResolvedValue({
      activity: { ...activity, status: 'completed' },
      winners: [
        {
          masked_username: 'r***1@example.com',
          amount_cents: 600,
          is_luckiest: true,
          is_current_user: true
        },
        {
          masked_username: 'a***r@example.com',
          amount_cents: 400,
          is_luckiest: false,
          is_current_user: false
        }
      ]
    })
    const wrapper = mount(RedPacketView, {
      attachTo: document.body,
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } }
    })
    await flushPromises()

    await wrapper.find('[data-test="history-detail-1"]').trigger('click')
    await flushPromises()

    expect(document.querySelector('[data-test="detail-activity-name"]')?.textContent).toBe(activity.name)
    expect(document.querySelector('[data-test="detail-activity-message"]')?.textContent).toBe(activity.message)
    const badges = document.querySelectorAll('[data-test="current-user-badge"]')
    expect(badges).toHaveLength(1)
    expect(badges[0].textContent).toBe('\u6211')
    const stats = document.querySelector('[data-test="detail-stats"]')
    expect(stats).not.toBeNull()
    expect(stats?.classList.contains('grid-cols-2')).toBe(true)
    expect(stats?.classList.contains('sm:grid-cols-4')).toBe(true)
    expect(stats?.classList.contains('border-y')).toBe(true)
    wrapper.unmount()
  })
})
