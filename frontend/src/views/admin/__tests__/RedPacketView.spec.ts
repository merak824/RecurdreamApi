import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RedPacketView from '../RedPacketView.vue'

const { list, create, update, publish, cancel, exportActivity, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  publish: vi.fn(),
  cancel: vi.fn(),
  exportActivity: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    redPackets: { list, create, update, publish, cancel, export: exportActivity }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => ({
      'admin.redPacket.title': '红包活动管理',
      'admin.redPacket.create': '创建活动',
      'admin.redPacket.save': '保存草稿',
      'admin.redPacket.publish': '发布活动',
      'admin.redPacket.cancel': '取消活动',
      'admin.redPacket.packetType': '红包类型',
      'admin.redPacket.lucky': '拼手气红包',
      'admin.redPacket.fixed': '固定红包',
      'admin.redPacket.totalAmount': '总金额（$）',
      'admin.redPacket.amountPreview': `合计 ${params?.amount ?? ''}`,
      'admin.redPacket.targetParticipants': '参与人数',
      'admin.redPacket.winnerCount': '中奖人数',
      'admin.redPacket.fixedDivisible': '固定红包总金额必须能被中奖人数整除',
      'admin.redPacket.winnerExceedsTarget': '中奖人数不能超过参与人数',
      'admin.redPacket.publishBlockedByRunning': '当前有正在进行的活动场次，请等本场结束后再发布新活动',
      'admin.redPacket.amountTooSmall': '总金额至少为中奖人数的 1 分之和',
      'admin.redPacket.readOnlyPublished': '已发布活动参数不可修改',
      'admin.redPacket.cancelConfirm': '确定取消这个进行中的活动吗？',
      'common.cancel': '取消',
      'common.confirm': '确认',
      'common.edit': '编辑'
      }[key] ?? key)
    })
  }
})

const draft = {
  id: 1,
  period_no: 8,
  name: '第8期红包',
  message: '达到人数自动开奖',
  packet_type: 'lucky',
  total_amount_cents: 1000,
  target_participants: 10,
  winner_count: 3,
  participant_count: 0,
  status: 'draft'
}

const TablePageLayoutStub = {
  template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
}

const DataTableStub = {
  props: ['data'],
  template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-actions" :row="row" /></div></div>'
}

describe('admin RedPacketView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue({ items: [draft], total: 1, page: 1, page_size: 20, pages: 1 })
    create.mockResolvedValue(draft)
    update.mockResolvedValue(draft)
    publish.mockResolvedValue({ ...draft, status: 'active' })
    cancel.mockResolvedValue(undefined)
  })

  it('previews red packet cents as US dollars', async () => {
    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          Select: true,
          ConfirmDialog: true,
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    expect(wrapper.find('[data-test="activity-overview"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-test="activity-overview-card"]')).toHaveLength(4)
    expect(wrapper.find('[data-test="activity-workspace"]').exists()).toBe(true)
    await wrapper.find('[data-test="create-button"]').trigger('click')

    expect(wrapper.find('[data-test="activity-config-panel"]').exists()).toBe(true)
    expect((wrapper.find('[data-test="total-amount"]').element as HTMLInputElement).value).toBe('1')
    expect(wrapper.text()).toContain('总金额（$）')
    expect(wrapper.text()).toContain('合计 $1.00')
    expect(wrapper.text()).not.toContain('¥')
  })

  it('converts dollar input to cents for the API payload', async () => {
    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          Select: true,
          ConfirmDialog: true,
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    await wrapper.find('[data-test="create-button"]').trigger('click')
    await wrapper.find('#red-packet-name').setValue('美元红包')
    await wrapper.find('[data-test="total-amount"]').setValue('100')
    await wrapper.find('[data-test="activity-form"]').trigger('submit')
    await flushPromises()

    expect(create).toHaveBeenCalledWith(expect.objectContaining({ total_amount_cents: 10000 }))
  })

  it('rejects indivisible fixed packets and winner counts above the target', async () => {
    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          Select: true,
          ConfirmDialog: true,
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    await wrapper.find('[data-test="create-button"]').trigger('click')
    await wrapper.find('[data-test="packet-type-fixed"]').setValue('fixed')
    await wrapper.find('[data-test="total-amount"]').setValue('500')
    await wrapper.find('[data-test="target-participants"]').setValue('2')
    await wrapper.find('[data-test="winner-count"]').setValue('3')
    await wrapper.find('[data-test="activity-form"]').trigger('submit')

    expect(wrapper.text()).toContain('中奖人数不能超过参与人数')
    expect(create).not.toHaveBeenCalled()
  })

  it('keeps published parameters read-only and exposes cancel confirmation for active activities', async () => {
    list.mockResolvedValue({
      items: [{ ...draft, status: 'active' }, { ...draft, id: 2, status: 'completed' }],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          Select: true,
          ConfirmDialog: {
            props: ['show', 'message'],
            template: '<div v-if="show" data-test="confirm-dialog">{{ message }}</div>'
          },
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    const editButton = wrapper.find('[data-test="edit-1"]')
    expect(editButton.exists()).toBe(true)
    await editButton.trigger('click')
    await flushPromises()
    expect((wrapper.find('[data-test="total-amount"]').element as HTMLInputElement).value).toBe('10')
    expect(wrapper.find('[data-test="total-amount"]').attributes('readonly')).toBeDefined()
    expect(wrapper.text()).toContain('已发布活动参数不可修改')

    await wrapper.find('[data-test="cancel-1"]').trigger('click')
    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('确定取消这个进行中的活动吗？')
  })

  it('explains why a draft cannot publish while another activity is active', async () => {
    list.mockResolvedValue({
      items: [{ ...draft, status: 'active' }, { ...draft, id: 2, period_no: 9, status: 'draft' }],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mount(RedPacketView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          Select: true,
          ConfirmDialog: true,
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()

    const publishButton = wrapper.find('[data-test="publish-2"]')
    expect(publishButton.exists()).toBe(true)
    expect((publishButton.element as HTMLButtonElement).disabled).toBe(false)

    await publishButton.trigger('click')
    expect(showError).toHaveBeenCalledWith('当前有正在进行的活动场次，请等本场结束后再发布新活动')
  })
})
