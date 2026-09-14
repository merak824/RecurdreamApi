import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateView from '../AffiliateView.vue'

const { copyToClipboard, createAffiliateWithdrawal, getAffiliateDetail, transferAffiliateQuota } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
  createAffiliateWithdrawal: vi.fn(),
  getAffiliateDetail: vi.fn(),
  transferAffiliateQuota: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    createAffiliateWithdrawal,
    getAffiliateDetail,
    transferAffiliateQuota,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    refreshUser: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AffiliateView', () => {
  const affiliateCode = 'affiliate-code-that-is-long-enough-to-overflow-a-mobile-viewport'

  function affiliateDetail(overrides: Record<string, unknown> = {}) {
    return {
      user_id: 1,
      aff_code: affiliateCode,
      inviter_id: null,
      aff_count: 0,
      aff_quota: 10,
      aff_history_quota: 28.5,
      effective_rebate_rate_percent: 10,
      withdrawal_enabled: false,
      invitees: [],
      withdrawals: [],
      ...overrides,
    }
  }

  function mountView() {
    return mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          BaseDialog: {
            props: ['show'],
            template: '<section v-if="show"><slot /><slot name="footer" /></section>',
          },
          Icon: true,
        },
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    copyToClipboard.mockResolvedValue(true)
    getAffiliateDetail.mockResolvedValue(affiliateDetail())
    createAffiliateWithdrawal.mockResolvedValue({ id: 1 })
    transferAffiliateQuota.mockResolvedValue({ transferred_quota: 1.23, balance: 1.23 })
  })

  it('stacks long values and copy controls on mobile while retaining desktop rows', async () => {
    const wrapper = mountView()

    await flushPromises()

    const values = wrapper.findAll('code')
    expect(values).toHaveLength(2)
    for (const value of values) {
      expect(value.classes()).toEqual(expect.arrayContaining([
        'min-w-0',
        'break-all',
        'sm:flex-1',
        'sm:truncate',
      ]))
      expect(Array.from(value.element.parentElement?.classList ?? [])).toEqual(expect.arrayContaining([
        'flex-col',
        'items-stretch',
        'sm:flex-row',
        'sm:items-center',
      ]))
    }

    const copyButtons = wrapper.findAll('button').filter((button) =>
      ['affiliate.copyCode', 'affiliate.copyLink'].includes(button.text()),
    )
    expect(copyButtons).toHaveLength(2)
    for (const button of copyButtons) {
      expect(button.classes()).toEqual(expect.arrayContaining([
        'w-full',
        'sm:w-auto',
        'sm:shrink-0',
      ]))
    }

    await copyButtons[0].trigger('click')
    await copyButtons[1].trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenNthCalledWith(1, affiliateCode, 'affiliate.codeCopied')
    expect(copyToClipboard).toHaveBeenNthCalledWith(
      2,
      `${window.location.origin}/register?aff=${encodeURIComponent(affiliateCode)}`,
      'affiliate.linkCopied',
    )
  })

  it('uses one rebate wallet and hides withdrawal controls without permission', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(wrapper.text()).not.toContain('affiliate.agent.identity')
    expect(wrapper.text()).not.toContain('affiliate.stats.frozenQuota')
    expect(wrapper.find('input[name="transfer-amount"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="withdraw-button"]').exists()).toBe(false)
  })

  it('submits an alipay withdrawal for an enabled exclusive user', async () => {
    getAffiliateDetail.mockResolvedValue(affiliateDetail({ withdrawal_enabled: true }))
    const wrapper = mountView()

    await flushPromises()
    await wrapper.get('[data-testid="withdraw-button"]').trigger('click')
    await wrapper.get('input[name="withdraw-amount"]').setValue('1.23')
    await wrapper.get('[data-payment-method="alipay"]').trigger('click')

    const fileInput = wrapper.get('input[name="collection-qr"]')
    const file = new File(['png'], 'collection.png', { type: 'image/png' })
    Object.defineProperty(fileInput.element, 'files', { value: [file] })
    await fileInput.trigger('change')
    await vi.waitFor(() => {
      expect(wrapper.find('img[alt="affiliate.withdraw.collectionQrPreview"]').exists()).toBe(true)
    })
    await wrapper.get('#affiliate-withdraw-form').trigger('submit')
    await flushPromises()

    expect(createAffiliateWithdrawal).toHaveBeenCalledWith({
      amount: 1.23,
      payment_method: 'alipay',
      collection_qr_data: expect.stringMatching(/^data:image\/png;base64,/),
    })
  })
})
