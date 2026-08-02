<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12" role="status" :aria-label="t('common.loading')">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(currentQuota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(currentHistory) }}
            </p>
          </div>
        </div>

        <section class="card p-6" aria-labelledby="affiliate-invite-heading">
          <div>
            <h2 id="affiliate-invite-heading" class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('affiliate.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>
          </div>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button type="button" class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button type="button" class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 border-l-4 border-primary-400 bg-primary-50 px-4 py-3 dark:border-primary-500 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
            </ul>
          </div>
        </section>

        <div class="grid gap-4 lg:grid-cols-2">
          <section class="card p-6" aria-labelledby="affiliate-transfer-heading">
            <h2 id="affiliate-transfer-heading" class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('affiliate.transfer.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>

            <form class="mt-5 space-y-3" @submit.prevent="submitTransfer">
              <label for="transfer-amount" class="input-label">{{ t('affiliate.transfer.amount') }}</label>
              <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                <div class="min-w-0 flex-1">
                  <input
                    id="transfer-amount"
                    v-model.number="transferForm.amount"
                    name="transfer-amount"
                    type="number"
                    min="0.01"
                    :max="currentQuota"
                    step="0.01"
                    inputmode="decimal"
                    class="input"
                  />
                </div>
                <button type="submit" class="btn btn-primary min-h-11 sm:shrink-0" :disabled="transferring || !transferAmountValid">
                  <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
                  <Icon v-else name="dollar" size="sm" />
                  <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
                </button>
              </div>
              <p class="input-hint">{{ t('affiliate.transfer.available', { amount: formatCurrency(currentQuota) }) }}</p>
            </form>
          </section>

          <section v-if="canWithdraw" class="card p-6" aria-labelledby="affiliate-withdraw-heading">
            <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 id="affiliate-withdraw-heading" class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('affiliate.withdraw.title') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.withdraw.description') }}</p>
              </div>
              <button
                type="button"
                data-testid="withdraw-button"
                class="btn btn-secondary min-h-11 sm:shrink-0"
                :disabled="currentQuota < minimumWithdrawal"
                @click="openWithdrawModal"
              >
                <Icon name="upload" size="sm" />
                <span>{{ t('affiliate.withdraw.button') }}</span>
              </button>
            </div>
            <p class="mt-4 text-sm text-gray-600 dark:text-dark-300">
              {{ t('affiliate.withdraw.minimum', { amount: formatCurrency(minimumWithdrawal) }) }}
            </p>
          </section>
        </div>

        <section v-if="canWithdraw" class="card p-6" aria-labelledby="affiliate-withdraw-records-heading">
          <h2 id="affiliate-withdraw-records-heading" class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('affiliate.withdraw.records') }}
          </h2>
          <div v-if="withdrawals.length === 0" class="mt-4 border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.withdraw.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[760px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.amount') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.paymentMethod') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.status') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.collectionQr') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.proof') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.createdAt') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.note') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in withdrawals" :key="item.id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3 font-medium text-gray-900 dark:text-white">{{ formatCurrency(item.amount) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ paymentMethodLabel(item.payment_method) }}</td>
                  <td class="px-3 py-3"><span :class="withdrawStatusClass(item.status)">{{ withdrawStatusLabel(item.status) }}</span></td>
                  <td class="px-3 py-3"><ImageThumb :src="item.collection_qr_data" /></td>
                  <td class="px-3 py-3"><ImageThumb :src="item.payment_proof_data" /></td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                  <td class="max-w-56 px-3 py-3 text-gray-700 dark:text-gray-300">
                    <span class="block truncate" :title="item.reject_reason || item.admin_note || ''">{{ item.reject_reason || item.admin_note || '-' }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="card p-6" aria-labelledby="affiliate-invitees-heading">
          <h2 id="affiliate-invitees-heading" class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('affiliate.invitees.title') }}
          </h2>
          <div v-if="detail.invitees.length === 0" class="mt-4 border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 text-right font-medium">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>

    <BaseDialog :show="withdrawModal" :title="t('affiliate.withdraw.title')" width="normal" @close="closeWithdrawModal">
      <form id="affiliate-withdraw-form" class="space-y-5" @submit.prevent="submitWithdrawal">
        <div>
          <label for="withdraw-amount" class="input-label">{{ t('affiliate.withdraw.amount') }}</label>
          <input
            id="withdraw-amount"
            v-model.number="withdrawForm.amount"
            name="withdraw-amount"
            type="number"
            :min="minimumWithdrawal"
            :max="currentQuota"
            step="0.01"
            inputmode="decimal"
            class="input"
          />
          <p class="input-hint">{{ t('affiliate.withdraw.available', { amount: formatCurrency(currentQuota) }) }}</p>
        </div>

        <fieldset>
          <legend class="input-label">{{ t('affiliate.withdraw.paymentMethod') }}</legend>
          <div class="grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
            <button
              v-for="method in paymentMethods"
              :key="method"
              type="button"
              :data-payment-method="method"
              :aria-pressed="withdrawForm.paymentMethod === method"
              :class="[
                'min-h-11 rounded-md px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500',
                withdrawForm.paymentMethod === method
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
                  : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white',
              ]"
              @click="withdrawForm.paymentMethod = method"
            >
              {{ paymentMethodLabel(method) }}
            </button>
          </div>
        </fieldset>

        <div>
          <label for="collection-qr" class="input-label">{{ t('affiliate.withdraw.collectionQr') }}</label>
          <input
            id="collection-qr"
            name="collection-qr"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            class="input"
            @change="handleCollectionQRChange"
          />
          <p class="input-hint">{{ t('affiliate.withdraw.imageHint') }}</p>
          <img
            v-if="withdrawForm.collectionQRData"
            :src="withdrawForm.collectionQRData"
            :alt="t('affiliate.withdraw.collectionQrPreview')"
            class="mt-3 h-32 w-32 rounded-lg border border-gray-200 object-cover dark:border-dark-700"
          />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeWithdrawModal">{{ t('common.cancel') }}</button>
          <button type="submit" form="affiliate-withdraw-form" class="btn btn-primary" :disabled="withdrawing || !withdrawAmountValid || !withdrawForm.collectionQRData">
            <Icon v-if="withdrawing" name="refresh" size="sm" class="animate-spin" />
            <span>{{ withdrawing ? t('affiliate.withdraw.submitting') : t('affiliate.withdraw.submit') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AffiliateWithdrawal, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

type AffiliatePaymentMethod = 'wechat' | 'alipay'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const minimumTransfer = 0.01
const minimumWithdrawal = 1
const paymentMethods: AffiliatePaymentMethod[] = ['wechat', 'alipay']
const loading = ref(true)
const transferring = ref(false)
const withdrawing = ref(false)
const withdrawModal = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const transferForm = reactive({ amount: 1 })
const withdrawForm = reactive({
  amount: minimumWithdrawal,
  paymentMethod: 'wechat' as AffiliatePaymentMethod,
  collectionQRData: '',
})

const currentQuota = computed(() => detail.value?.aff_quota ?? 0)
const currentHistory = computed(() => detail.value?.aff_history_quota ?? 0)
const canWithdraw = computed(() => detail.value?.withdrawal_enabled === true)
const withdrawals = computed<AffiliateWithdrawal[]>(() => detail.value?.withdrawals || [])
const transferAmountValid = computed(() => validCurrencyAmount(transferForm.amount, minimumTransfer))
const withdrawAmountValid = computed(() => canWithdraw.value && validCurrencyAmount(withdrawForm.amount, minimumWithdrawal))

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

const formattedRebateRate = computed(() => {
  const value = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(value * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function validCurrencyAmount(value: number, minimum: number): boolean {
  return Number.isFinite(value) && value >= minimum && value <= currentQuota.value
}

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) loading.value = true
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) loading.value = false
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function submitTransfer(): Promise<void> {
  if (!detail.value || transferring.value) return
  if (!transferAmountValid.value) {
    appStore.showError(t('affiliate.transfer.invalidAmount'))
    return
  }
  transferring.value = true
  try {
    const response = await userAPI.transferAffiliateQuota({ amount: transferForm.amount })
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(response.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

function openWithdrawModal(): void {
  if (!canWithdraw.value || currentQuota.value < minimumWithdrawal) return
  withdrawForm.amount = minimumWithdrawal
  withdrawForm.paymentMethod = 'wechat'
  withdrawForm.collectionQRData = ''
  withdrawModal.value = true
}

function closeWithdrawModal(): void {
  if (withdrawing.value) return
  withdrawModal.value = false
}

async function readImageFile(file: File): Promise<string> {
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    throw new Error(t('affiliate.withdraw.invalidImage'))
  }
  if (file.size > 2 * 1024 * 1024) {
    throw new Error(t('affiliate.withdraw.imageTooLarge'))
  }
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error(t('affiliate.withdraw.invalidImage')))
    reader.readAsDataURL(file)
  })
}

async function handleCollectionQRChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    withdrawForm.collectionQRData = await readImageFile(file)
  } catch (error) {
    withdrawForm.collectionQRData = ''
    appStore.showError(error instanceof Error ? error.message : t('affiliate.withdraw.invalidImage'))
  }
}

async function submitWithdrawal(): Promise<void> {
  if (!detail.value || withdrawing.value || !canWithdraw.value) return
  if (!withdrawAmountValid.value) {
    appStore.showError(t('affiliate.withdraw.invalidAmount'))
    return
  }
  if (!withdrawForm.collectionQRData) {
    appStore.showError(t('affiliate.withdraw.qrRequired'))
    return
  }
  withdrawing.value = true
  try {
    await userAPI.createAffiliateWithdrawal({
      amount: withdrawForm.amount,
      payment_method: withdrawForm.paymentMethod,
      collection_qr_data: withdrawForm.collectionQRData,
    })
    appStore.showSuccess(t('affiliate.withdraw.success'))
    withdrawModal.value = false
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdraw.failed')))
  } finally {
    withdrawing.value = false
  }
}

function paymentMethodLabel(method?: string): string {
  return method ? t(`affiliate.withdraw.paymentMethods.${method}`, method) : '-'
}

function withdrawStatusLabel(status: string): string {
  return t(`affiliate.withdraw.statuses.${status}`, status)
}

function withdrawStatusClass(status: string): string {
  const base = 'badge '
  if (status === 'paid') return base + 'badge-success'
  if (status === 'rejected') return base + 'badge-danger'
  return base + 'badge-warning'
}

const ImageThumb = defineComponent({
  props: { src: { type: String, default: '' } },
  setup(props) {
    return () => props.src
      ? h('a', { href: props.src, target: '_blank', rel: 'noreferrer' }, [
          h('img', { src: props.src, alt: '', class: 'h-12 w-12 rounded border border-gray-200 object-cover dark:border-dark-700' }),
        ])
      : h('span', { class: 'text-sm text-gray-400 dark:text-dark-500' }, '-')
  },
})

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
