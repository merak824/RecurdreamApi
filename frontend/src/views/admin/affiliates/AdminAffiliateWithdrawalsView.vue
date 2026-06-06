<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-80">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="filters.search" type="text" class="input pl-10" :placeholder="t('admin.affiliates.records.searchPlaceholder')" @input="debounceLoad" />
          </div>
          <input v-model="filters.start_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.startAt')" @change="reloadFromFirstPage" />
          <input v-model="filters.end_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.endAt')" @change="reloadFromFirstPage" />
          <button class="btn btn-secondary px-2 md:px-3" :disabled="loading" :title="t('common.refresh')" @click="loadRecords">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :row-key="recordRowKey"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          sort-storage-key="admin-affiliate-withdrawals-table-sort"
          @sort="handleSort"
        >
          <template #cell-user="{ row }">
            <div class="space-y-0.5">
              <div class="font-mono text-sm text-gray-900 dark:text-white">#{{ row.user_id }}</div>
              <div class="max-w-56 truncate text-sm font-medium text-gray-900 dark:text-white">{{ row.user_email || '-' }}</div>
              <div class="max-w-56 truncate text-sm text-gray-500 dark:text-dark-400">{{ row.username || '-' }}</div>
            </div>
          </template>
          <template #cell-amount="{ row }">
            <span class="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrency(row.amount) }}</span>
          </template>
          <template #cell-destination="{ row }">
            <span :class="destinationClass(row.destination)">{{ destinationLabel(row.destination) }}</span>
          </template>
          <template #cell-status="{ row }">
            <span :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
          </template>
          <template #cell-collection_qr="{ row }">
            <ImageThumb :src="row.collection_qr_data" />
          </template>
          <template #cell-payment_proof="{ row }">
            <ImageThumb :src="row.payment_proof_data" />
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(row.created_at) || '-' }}</span>
          </template>
          <template #cell-processed_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(row.processed_at) || '-' }}</span>
          </template>
          <template #cell-note="{ row }">
            <span class="block max-w-56 truncate text-sm text-gray-700 dark:text-gray-300" :title="row.reject_reason || row.admin_note || ''">
              {{ row.reject_reason || row.admin_note || '-' }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <div v-if="canProcess(row)" class="flex items-center gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="row.status !== 'pending'" @click="openPaidDialog(row)">
                <Icon name="check" size="sm" />
                <span>{{ t('admin.affiliates.records.markPaid') }}</span>
              </button>
              <button class="btn btn-danger btn-sm" :disabled="row.status !== 'pending'" @click="openRejectDialog(row)">
                <Icon name="x" size="sm" />
                <span>{{ t('admin.affiliates.records.reject') }}</span>
              </button>
            </div>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog :show="paidDialog" :title="t('admin.affiliates.records.markPaid')" width="normal" @close="closeActionDialog">
      <form id="affiliate-paid-form" class="space-y-5" @submit.prevent="submitPaid">
        <div>
          <label class="input-label">{{ t('admin.affiliates.records.uploadProof') }}</label>
          <input type="file" accept="image/png,image/jpeg,image/webp" class="input" @change="handleProofChange" />
          <p class="input-hint">{{ t('admin.affiliates.records.imageHint') }}</p>
          <img v-if="paidForm.paymentProofData" :src="paidForm.paymentProofData" alt="payment proof" class="mt-3 h-32 w-32 rounded-lg border border-gray-200 object-cover dark:border-dark-700" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.affiliates.records.adminNote') }}</label>
          <textarea v-model="paidForm.adminNote" rows="3" class="input"></textarea>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="closeActionDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="submit" form="affiliate-paid-form" :disabled="submitting">
            <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('admin.affiliates.records.markPaid') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="rejectDialog" :title="t('admin.affiliates.records.reject')" width="normal" @close="closeActionDialog">
      <form id="affiliate-reject-form" class="space-y-5" @submit.prevent="submitReject">
        <div>
          <label class="input-label">{{ t('admin.affiliates.records.rejectReason') }}</label>
          <textarea v-model="rejectForm.rejectReason" rows="3" class="input"></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('admin.affiliates.records.adminNote') }}</label>
          <textarea v-model="rejectForm.adminNote" rows="3" class="input"></textarea>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="closeActionDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" type="submit" form="affiliate-reject-form" :disabled="submitting">
            <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('admin.affiliates.records.reject') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { defineComponent, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { affiliatesAPI, type AffiliateWithdrawalRecord, type ListAffiliateRecordsParams } from '@/api/admin/affiliates'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const submitting = ref(false)
const records = ref<AffiliateWithdrawalRecord[]>([])
const filters = reactive({ search: '', start_at: '', end_at: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const sortState = reactive({ sort_by: 'created_at', sort_order: 'desc' as 'asc' | 'desc' })
const selected = ref<AffiliateWithdrawalRecord | null>(null)
const paidDialog = ref(false)
const rejectDialog = ref(false)
const paidForm = reactive({ paymentProofData: '', adminNote: '' })
const rejectForm = reactive({ rejectReason: '', adminNote: '' })
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const columns: Column[] = [
  { key: 'user', label: t('admin.affiliates.records.user'), sortable: true },
  { key: 'amount', label: t('admin.affiliates.records.amount'), sortable: true },
  { key: 'destination', label: t('admin.affiliates.records.destination'), sortable: true },
  { key: 'status', label: t('admin.affiliates.records.status'), sortable: true },
  { key: 'collection_qr', label: t('admin.affiliates.records.collectionQr') },
  { key: 'payment_proof', label: t('admin.affiliates.records.paymentProof') },
  { key: 'created_at', label: t('admin.affiliates.records.createdAt'), sortable: true },
  { key: 'processed_at', label: t('admin.affiliates.records.processedAt'), sortable: true },
  { key: 'note', label: t('admin.affiliates.records.note') },
  { key: 'actions', label: t('admin.affiliates.records.actions') },
]

function recordRowKey(row: AffiliateWithdrawalRecord): string {
  return `${row.record_type}-${row.id}`
}

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

function buildParams(): ListAffiliateRecordsParams {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    search: filters.search.trim() || undefined,
    start_at: filters.start_at || undefined,
    end_at: filters.end_at || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order,
    timezone: userTimezone(),
  }
}

async function loadRecords() {
  loading.value = true
  try {
    const res = await affiliatesAPI.listWithdrawalRecords(buildParams())
    records.value = res.items || []
    pagination.total = res.total || 0
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function debounceLoad() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => reloadFromFirstPage(), 300)
}

function reloadFromFirstPage() {
  pagination.page = 1
  void loadRecords()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadRecords()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  void loadRecords()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadRecords()
}

function statusLabel(status: string): string {
  return t(`admin.affiliates.records.statuses.${status}`, status)
}

function destinationLabel(destination: string): string {
  return t(`admin.affiliates.records.destinations.${destination}`, destination)
}

function destinationClass(destination: string): string {
  const base = 'badge '
  return destination === 'balance' ? base + 'badge-primary' : base + 'badge-warning'
}

function statusClass(status: string): string {
  const base = 'badge '
  if (status === 'completed') return base + 'badge-success'
  if (status === 'paid') return base + 'badge-success'
  if (status === 'rejected') return base + 'badge-danger'
  return base + 'badge-warning'
}

function canProcess(row: AffiliateWithdrawalRecord): boolean {
  return row.record_type === 'withdrawal' && row.status === 'pending'
}

function openPaidDialog(row: AffiliateWithdrawalRecord) {
  if (!canProcess(row)) return
  selected.value = row
  paidForm.paymentProofData = ''
  paidForm.adminNote = ''
  paidDialog.value = true
}

function openRejectDialog(row: AffiliateWithdrawalRecord) {
  if (!canProcess(row)) return
  selected.value = row
  rejectForm.rejectReason = ''
  rejectForm.adminNote = ''
  rejectDialog.value = true
}

function closeActionDialog() {
  if (submitting.value) return
  paidDialog.value = false
  rejectDialog.value = false
  selected.value = null
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

async function handleProofChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    paidForm.paymentProofData = await readImageFile(file)
  } catch (error) {
    paidForm.paymentProofData = ''
    appStore.showError(error instanceof Error ? error.message : t('affiliate.withdraw.invalidImage'))
  }
}

async function submitPaid(): Promise<void> {
  if (!selected.value || submitting.value) return
  submitting.value = true
  try {
    await affiliatesAPI.markWithdrawalPaid(selected.value.id, {
      payment_proof_data: paidForm.paymentProofData,
      admin_note: paidForm.adminNote,
    })
    appStore.showSuccess(t('admin.affiliates.records.paidSuccess'))
    closeActionDialog()
    await loadRecords()
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    submitting.value = false
  }
}

async function submitReject(): Promise<void> {
  if (!selected.value || submitting.value) return
  submitting.value = true
  try {
    await affiliatesAPI.rejectWithdrawal(selected.value.id, {
      reject_reason: rejectForm.rejectReason,
      admin_note: rejectForm.adminNote,
    })
    appStore.showSuccess(t('admin.affiliates.records.rejectSuccess'))
    closeActionDialog()
    await loadRecords()
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    submitting.value = false
  }
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
  void loadRecords()
})
</script>
