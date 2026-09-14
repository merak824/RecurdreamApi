import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AdminAffiliateWithdrawalsView from '../AdminAffiliateWithdrawalsView.vue'

const { listWithdrawalRecords, markWithdrawalPaid, rejectWithdrawal } = vi.hoisted(() => ({
  listWithdrawalRecords: vi.fn(),
  markWithdrawalPaid: vi.fn(),
  rejectWithdrawal: vi.fn(),
}))

vi.mock('@/api/admin/affiliates', () => {
  const affiliatesAPI = {
    listWithdrawalRecords,
    markWithdrawalPaid,
    rejectWithdrawal,
  }
  return { affiliatesAPI, default: affiliatesAPI }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

describe('AdminAffiliateWithdrawalsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listWithdrawalRecords.mockResolvedValue({
      items: [{
        id: 9,
        record_type: 'withdrawal',
        destination: 'alipay_wechat',
        user_id: 7,
        user_email: 'exclusive@example.com',
        username: 'exclusive',
        amount: 12.5,
        payment_method: 'alipay',
        status: 'pending',
        collection_qr_data: 'data:image/png;base64,cXJjb2Rl',
        created_at: '2026-07-31T10:00:00Z',
        updated_at: '2026-07-31T10:00:00Z',
      }],
      total: 1,
      page: 1,
      page_size: 20,
    })
  })

  it('renders withdrawal channel, QR code, amount, status, and review actions', async () => {
    const wrapper = mount(AdminAffiliateWithdrawalsView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: { template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>' },
          DataTable: {
            props: ['data'],
            template: `
              <div>
                <div v-for="row in data" :key="row.id">
                  <slot name="cell-amount" :row="row" />
                  <slot name="cell-payment_method" :row="row" />
                  <slot name="cell-status" :row="row" />
                  <slot name="cell-collection_qr" :row="row" />
                  <slot name="cell-actions" :row="row" />
                </div>
              </div>
            `,
          },
          Pagination: true,
          BaseDialog: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('12.50')
    expect(wrapper.text()).toContain('admin.affiliates.records.paymentMethods.alipay')
    expect(wrapper.text()).toContain('admin.affiliates.records.statuses.pending')
    expect(wrapper.find('img[src^="data:image/png;base64,"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('admin.affiliates.records.markPaid')
    expect(wrapper.text()).toContain('admin.affiliates.records.reject')
  })

  it('closes the paid dialog after a successful review', async () => {
    markWithdrawalPaid.mockResolvedValue({})
    const wrapper = mount(AdminAffiliateWithdrawalsView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          TablePageLayout: { template: '<section><slot name="table" /></section>' },
          DataTable: {
            props: ['data'],
            template: `
              <div>
                <div v-for="row in data" :key="row.id">
                  <slot name="cell-actions" :row="row" />
                </div>
              </div>
            `,
          },
          Pagination: true,
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show" data-testid="action-dialog"><slot /><slot name="footer" /></div>',
          },
          Icon: true,
        },
      },
    })

    await flushPromises()
    const markPaidButton = wrapper.findAll('button').find(button => button.text().includes('admin.affiliates.records.markPaid'))
    expect(markPaidButton).toBeDefined()
    await markPaidButton!.trigger('click')
    expect(wrapper.find('#affiliate-paid-form').exists()).toBe(true)

    await wrapper.find('#affiliate-paid-form').trigger('submit')
    await flushPromises()

    expect(markWithdrawalPaid).toHaveBeenCalledWith(9, {
      payment_proof_data: '',
      admin_note: '',
    })
    expect(wrapper.find('[data-testid="action-dialog"]').exists()).toBe(false)
  })
})

describe('affiliate withdrawal review navigation', () => {
  const currentDir = dirname(fileURLToPath(import.meta.url))
  const routerSource = readFileSync(resolve(currentDir, '../../../../router/index.ts'), 'utf8')
  const sidebarSource = readFileSync(resolve(currentDir, '../../../../components/layout/AppSidebar.vue'), 'utf8')

  it('registers the withdrawal review route and sidebar item without removing red packets', () => {
    expect(routerSource).toContain("path: '/admin/affiliates/withdrawals'")
    expect(routerSource).toContain("name: 'AdminAffiliateWithdrawals'")
    expect(sidebarSource).toContain("path: '/admin/affiliates/withdrawals'")
    expect(sidebarSource).toContain("path: '/admin/red-packets'")
  })
})
