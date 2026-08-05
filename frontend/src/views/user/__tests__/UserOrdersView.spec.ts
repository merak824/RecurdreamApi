import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UserOrdersView from '../UserOrdersView.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getMyOrders: vi.fn().mockResolvedValue({ data: { items: [], total: 0 } }),
    getRefundEligibleProviders: vi.fn().mockResolvedValue({ data: { provider_instance_ids: [] } }),
    cancelOrder: vi.fn(),
    requestRefund: vi.fn()
  }
}))

const currentDir = dirname(fileURLToPath(import.meta.url))
const viewSource = readFileSync(resolve(currentDir, '../UserOrdersView.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(currentDir, '../../../components/layout/AppSidebar.vue'), 'utf8')

describe('UserOrdersView balance history placement', () => {
  it('keeps balance history inside My Orders as a keyboard-accessible tab', () => {
    expect(viewSource).toContain('role="tablist"')
    expect(viewSource).toContain('payment.orders.orderTab')
    expect(viewSource).toContain('payment.orders.balanceHistoryTab')
    expect(viewSource).toContain("const activeSection = ref<'orders' | 'balance'>('orders')")
    expect(viewSource).toContain('<UserBalanceHistoryPanel')
    expect(viewSource).toContain('v-else')
  })

  it('does not add a dedicated balance-history sidebar item', () => {
    expect(sidebarSource).not.toContain("path: '/balance-history'")
  })

  it('switches tabs with the standard left and right arrow keys', async () => {
    const wrapper = shallowMount(UserOrdersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
          OrderTable: true,
          Pagination: true,
          Select: true,
          UserBalanceHistoryPanel: true
        }
      }
    })

    let tabs = wrapper.findAll('[role="tab"]')
    expect(tabs[0].attributes('aria-selected')).toBe('true')

    await tabs[0].trigger('keydown', { key: 'ArrowRight' })
    tabs = wrapper.findAll('[role="tab"]')
    expect(tabs[1].attributes('aria-selected')).toBe('true')

    await tabs[1].trigger('keydown', { key: 'ArrowLeft' })
    tabs = wrapper.findAll('[role="tab"]')
    expect(tabs[0].attributes('aria-selected')).toBe('true')
  })
})
