<template>
  <section class="overflow-hidden border-y border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800 sm:rounded-lg sm:border">
    <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('payment.orders.balanceHistoryTitle') }}
        </h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('payment.orders.balanceHistorySubtitle') }}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Select
          v-model="typeFilter"
          :options="typeOptions"
          class="min-w-0 flex-1 sm:w-44 sm:flex-none"
          @change="handleFilterChange"
        />
        <button
          type="button"
          class="btn btn-secondary h-11 w-11 shrink-0 p-0"
          :disabled="loading"
          :aria-label="t('common.refresh')"
          :title="t('common.refresh')"
          @click="loadHistory(pagination.page)"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin motion-reduce:animate-none' : ''" />
        </button>
      </div>
    </div>

    <div v-if="loading && items.length === 0" class="flex min-h-56 items-center justify-center" aria-live="polite">
      <Icon name="refresh" size="lg" class="animate-spin text-primary-500 motion-reduce:animate-none" />
    </div>

    <div v-else-if="items.length === 0" class="flex min-h-56 flex-col items-center justify-center px-6 text-center">
      <span class="flex h-11 w-11 items-center justify-center rounded-lg bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500">
        <Icon name="clock" size="md" aria-hidden="true" />
      </span>
      <p class="mt-3 text-sm font-medium text-gray-700 dark:text-gray-200">
        {{ t('payment.orders.noBalanceHistory') }}
      </p>
    </div>

    <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700" aria-live="polite">
      <li
        v-for="item in items"
        :key="item.id"
        class="flex min-w-0 items-center gap-3 px-4 py-3 sm:px-5"
      >
        <span :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', iconTone(item)]">
          <Icon :name="iconName(item)" size="sm" aria-hidden="true" />
        </span>

        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
              {{ itemTitle(item) }}
            </p>
            <span v-if="sourceLabel(item)" class="text-xs text-gray-400 dark:text-gray-500">
              {{ sourceLabel(item) }}
            </span>
          </div>
          <p v-if="item.description" class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400" :title="item.description">
            {{ item.description }}
          </p>
          <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
            {{ formatDateTime(item.occurred_at) }}
          </p>
        </div>

        <p :class="['shrink-0 text-right text-sm font-semibold tabular-nums', amountTone(item)]">
          {{ formatAmount(item.amount) }}
        </p>
      </li>
    </ul>

    <Pagination
      v-if="pagination.total > pagination.page_size"
      :page="pagination.page"
      :total="pagination.total"
      :page-size="pagination.page_size"
      :show-page-size-selector="false"
      @update:page="loadHistory"
      @update:pageSize="handlePageSizeChange"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { redeemAPI, type BalanceHistoryItem, type BalanceHistoryType } from '@/api/redeem'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const items = ref<BalanceHistoryItem[]>([])
const typeFilter = ref<BalanceHistoryType | ''>('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const typeOptions = computed(() => [
  { value: '', label: t('payment.orders.balanceTypes.all') },
  { value: 'balance', label: t('payment.orders.balanceTypes.recharge') },
  { value: 'admin_balance', label: t('payment.orders.balanceTypes.admin') },
  { value: 'affiliate_balance', label: t('payment.orders.balanceTypes.affiliate') },
  { value: 'red_packet_reward', label: t('payment.orders.balanceTypes.redPacket') }
])

async function loadHistory(page: number) {
  loading.value = true
  pagination.page = page
  try {
    const result = await redeemAPI.getBalanceHistory(
      page,
      pagination.page_size,
      typeFilter.value || undefined
    )
    items.value = result.items || []
    pagination.total = result.total || 0
  } catch {
    appStore.showError(t('payment.orders.balanceHistoryLoadFailed'))
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  loadHistory(1)
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  loadHistory(1)
}

function itemTitle(item: BalanceHistoryItem): string {
  if (item.type === 'red_packet_reward') return t('payment.orders.balanceTypes.redPacket')
  if (item.type === 'affiliate_balance') return t('payment.orders.balanceTypes.affiliate')
  if (item.type === 'admin_balance') {
    return item.amount >= 0
      ? t('payment.orders.balanceTypes.adminCredit')
      : t('payment.orders.balanceTypes.adminDebit')
  }
  return t('payment.orders.balanceTypes.recharge')
}

function sourceLabel(item: BalanceHistoryItem): string {
  if (item.type === 'red_packet_reward' && item.reference) {
    return t('payment.orders.redPacketPeriod', { period: item.reference })
  }
  if (item.type === 'balance' && item.reference) {
    return item.reference.length > 10 ? `${item.reference.slice(0, 8)}...` : item.reference
  }
  return ''
}

function iconName(item: BalanceHistoryItem): 'gift' | 'users' | 'dollar' {
  if (item.type === 'red_packet_reward') return 'gift'
  if (item.type === 'affiliate_balance') return 'users'
  return 'dollar'
}

function iconTone(item: BalanceHistoryItem): string {
  if (item.type === 'red_packet_reward') return 'bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300'
  if (item.type === 'affiliate_balance') return 'bg-sky-50 text-sky-600 dark:bg-sky-950/40 dark:text-sky-300'
  if (item.amount < 0) return 'bg-amber-50 text-amber-600 dark:bg-amber-950/40 dark:text-amber-300'
  return 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300'
}

function amountTone(item: BalanceHistoryItem): string {
  return item.amount >= 0
    ? 'text-emerald-600 dark:text-emerald-400'
    : 'text-red-600 dark:text-red-400'
}

function formatAmount(amount: number): string {
  const sign = amount >= 0 ? '+' : '-'
  return `${sign}$${Math.abs(amount).toFixed(2)}`
}

onMounted(() => loadHistory(1))
</script>
