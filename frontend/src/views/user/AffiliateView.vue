<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <div class="flex items-center justify-between gap-3">
              <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
                <Icon name="dollar" size="sm" class="text-primary-500" />
                {{ t('affiliate.stats.rebateRate') }}
              </p>
              <span :class="['badge', isAgent ? 'badge-blue' : 'badge-gray']">
                {{ isAgent ? t('affiliate.agent.identity') : t('affiliate.normal.identity') }}
              </span>
            </div>
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
            <p v-if="isAgent" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.agent.pending') }}: {{ formatCurrency(detail.agent_withdraw_pending || 0) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(currentHistory) }}
            </p>
            <p v-if="currentFrozen > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(currentFrozen) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>
            </div>
            <span v-if="isAgent" class="badge badge-blue">{{ t('affiliate.agent.mode') }}</span>
          </div>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ isAgent ? t('affiliate.agent.tip') : t('affiliate.tips.line3') }}</li>
              <li v-if="currentFrozen > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-[1fr_1fr]">
          <div class="card p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
              </div>
              <button
                class="btn btn-primary"
                :disabled="transferring || currentQuota < minActionAmount"
                @click="openTransferModal"
              >
                <Icon name="dollar" size="sm" />
                <span>{{ t('affiliate.transfer.button') }}</span>
              </button>
            </div>
            <p class="mt-4 text-sm text-gray-600 dark:text-dark-300">
              {{ t('affiliate.transfer.available', { amount: formatCurrency(currentQuota) }) }}
            </p>
            <p v-if="currentQuota < minActionAmount" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
              {{ t('affiliate.transfer.empty') }}
            </p>
          </div>

          <div v-if="isAgent" class="card p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.withdraw.title') }}</h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.withdraw.description') }}</p>
              </div>
              <button class="btn btn-secondary" :disabled="currentQuota < minActionAmount" @click="openWithdrawModal">
                <Icon name="upload" size="sm" />
                <span>{{ t('affiliate.withdraw.button') }}</span>
              </button>
            </div>
            <div class="mt-4 grid grid-cols-2 gap-3 text-sm">
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-dark-400">{{ t('affiliate.agent.pending') }}</div>
                <div class="mt-1 font-semibold text-amber-600 dark:text-amber-400">{{ formatCurrency(detail.agent_withdraw_pending || 0) }}</div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-dark-400">{{ t('affiliate.agent.paid') }}</div>
                <div class="mt-1 font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrency(detail.agent_withdraw_paid || 0) }}</div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="isAgent" class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.withdraw.records') }}</h3>
          <div v-if="withdrawals.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.withdraw.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[720px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.withdraw.amount') }}</th>
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
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
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
        </div>
      </template>
    </div>

    <BaseDialog :show="transferModal" :title="t('affiliate.transfer.title')" width="normal" @close="closeTransferModal">
      <form id="affiliate-transfer-form" class="space-y-5" @submit.prevent="submitTransfer">
        <div>
          <label class="input-label">{{ t('affiliate.transfer.amount') }}</label>
          <div class="grid grid-cols-4 gap-2">
            <button
              v-for="amount in actionAmounts"
              :key="amount"
              type="button"
              :class="['rounded-lg border px-3 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-45', transferForm.amount === amount ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 text-gray-700 hover:bg-gray-50 dark:border-dark-700 dark:text-dark-300 dark:hover:bg-dark-800']"
              :disabled="amount > currentQuota"
              @click="transferForm.amount = amount"
            >
              {{ formatCurrency(amount) }}
            </button>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.available', { amount: formatCurrency(currentQuota) }) }}</p>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeTransferModal">{{ t('common.cancel') }}</button>
          <button type="submit" form="affiliate-transfer-form" class="btn btn-primary" :disabled="transferring">
            <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
            <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.submit') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="withdrawModal" :title="t('affiliate.withdraw.title')" width="normal" @close="closeWithdrawModal">
      <form id="affiliate-withdraw-form" class="space-y-5" @submit.prevent="submitWithdrawal">
        <div>
          <label class="input-label">{{ t('affiliate.withdraw.amount') }}</label>
          <div class="grid grid-cols-4 gap-2">
            <button
              v-for="amount in actionAmounts"
              :key="amount"
              type="button"
              :class="['rounded-lg border px-3 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-45', withdrawForm.amount === amount ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 text-gray-700 hover:bg-gray-50 dark:border-dark-700 dark:text-dark-300 dark:hover:bg-dark-800']"
              :disabled="amount > currentQuota"
              @click="withdrawForm.amount = amount"
            >
              {{ formatCurrency(amount) }}
            </button>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.withdraw.available', { amount: formatCurrency(currentQuota) }) }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('affiliate.withdraw.collectionQr') }}</label>
          <input type="file" accept="image/png,image/jpeg,image/webp" class="input" @change="handleCollectionQRChange" />
          <p class="input-hint">{{ t('affiliate.withdraw.imageHint') }}</p>
          <img v-if="withdrawForm.collectionQRData" :src="withdrawForm.collectionQRData" alt="collection qr" class="mt-3 h-32 w-32 rounded-lg border border-gray-200 object-cover dark:border-dark-700" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeWithdrawModal">{{ t('common.cancel') }}</button>
          <button type="submit" form="affiliate-withdraw-form" class="btn btn-primary" :disabled="withdrawing">
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

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const withdrawing = ref(false)
const transferModal = ref(false)
const withdrawModal = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const actionAmounts = [5, 10, 20, 50, 100, 200, 500, 1000]
const minActionAmount = actionAmounts[0]
const transferForm = reactive({ amount: 5 })
const withdrawForm = reactive({ amount: 5, collectionQRData: '' })

const isAgent = computed(() => detail.value?.current_mode === 'agent')
const currentQuota = computed(() => detail.value?.current_aff_quota ?? detail.value?.aff_quota ?? 0)
const currentFrozen = computed(() => detail.value?.current_aff_frozen_quota ?? detail.value?.aff_frozen_quota ?? 0)
const currentHistory = computed(() => detail.value?.current_aff_history_quota ?? detail.value?.aff_history_quota ?? 0)
const withdrawals = computed<AffiliateWithdrawal[]>(() => detail.value?.withdrawals || [])

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

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

function firstAllowedActionAmount(): number {
  return actionAmounts.find((amount) => amount <= currentQuota.value) || minActionAmount
}

function isValidActionAmount(amount: number): boolean {
  return actionAmounts.includes(amount) && amount <= currentQuota.value
}

function openTransferModal(): void {
  if (currentQuota.value < minActionAmount) return
  transferForm.amount = firstAllowedActionAmount()
  transferModal.value = true
}

function closeTransferModal(): void {
  if (transferring.value) return
  transferModal.value = false
}

async function submitTransfer(): Promise<void> {
  if (!detail.value || transferring.value) return
  if (!isValidActionAmount(transferForm.amount)) {
    appStore.showError(t('affiliate.transfer.invalidAmount'))
    return
  }
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota({ amount: transferForm.amount })
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    transferModal.value = false
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
  if (currentQuota.value < minActionAmount) return
  withdrawForm.amount = firstAllowedActionAmount()
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
  if (!detail.value || withdrawing.value) return
  if (!isValidActionAmount(withdrawForm.amount)) {
    appStore.showError(t('affiliate.withdraw.invalidAmount'))
    return
  }
  if (!withdrawForm.collectionQRData) {
    appStore.showError(t('affiliate.withdraw.qrRequired'))
    return
  }
  withdrawing.value = true
  try {
    await userAPI.createAffiliateWithdrawal({ amount: withdrawForm.amount, collection_qr_data: withdrawForm.collectionQRData })
    appStore.showSuccess(t('affiliate.withdraw.success'))
    withdrawModal.value = false
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdraw.failed')))
  } finally {
    withdrawing.value = false
  }
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
          h('img', { src: props.src, class: 'h-12 w-12 rounded border border-gray-200 object-cover dark:border-dark-700' }),
        ])
      : h('span', { class: 'text-sm text-gray-400 dark:text-dark-500' }, '-')
  },
})

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
