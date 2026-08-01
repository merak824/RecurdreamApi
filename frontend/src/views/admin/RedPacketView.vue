<template>
  <AppLayout>
    <div class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">{{ t('admin.redPacket.title') }}</h1>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{{ t('admin.redPacket.description') }}</p>
      </div>
      <button type="button" class="btn btn-primary min-h-11 w-full sm:w-auto" data-test="create-button" @click="openCreate">
        <Icon name="plus" size="sm" aria-hidden="true" />
        {{ t('admin.redPacket.create') }}
      </button>
    </div>

    <section data-test="activity-overview" class="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <div v-for="item in activityOverview" :key="item.key" data-test="activity-overview-card" class="flex min-h-24 items-center gap-4 rounded-lg border border-gray-200 bg-white px-4 py-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <span :class="item.iconClass" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
          <Icon :name="item.icon" size="md" aria-hidden="true" />
        </span>
        <div class="min-w-0">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-1 text-xl font-semibold tabular-nums text-gray-950 dark:text-white">{{ item.value }}</p>
        </div>
      </div>
    </section>

    <div data-test="activity-workspace">
    <TablePageLayout>
      <template #actions>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ t('admin.redPacket.activityList') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.redPacket.totalActivities', { count: pagination.total }) }}
            </p>
          </div>
          <span class="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1.5 text-xs font-medium text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">
            <span class="h-1.5 w-1.5 rounded-full bg-primary-500" aria-hidden="true"></span>
            {{ t('admin.redPacket.autoDrawEnabled') }}
          </span>
          <p v-if="hasRunningActivity" class="basis-full text-xs text-amber-700 dark:text-amber-300">
            {{ t('admin.redPacket.publishAfterCurrentEnds') }}
          </p>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="activities"
          :loading="loading"
          row-key="id"
          :actions-count="3"
        >
          <template #cell-period_no="{ row }">
            <div class="min-w-0">
              <p class="font-medium text-gray-950 dark:text-white">{{ t('admin.redPacket.period', { period: row.period_no }) }}</p>
              <p class="mt-0.5 max-w-56 truncate text-xs text-gray-500 dark:text-gray-400">{{ row.name }}</p>
            </div>
          </template>
          <template #cell-packet_type="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ packetTypeLabel(row.packet_type) }}</span>
          </template>
          <template #cell-total_amount_cents="{ row }">
            <span class="font-medium tabular-nums text-gray-900 dark:text-gray-100">{{ formatMoney(row.total_amount_cents) }}</span>
          </template>
          <template #cell-progress="{ row }">
            <div class="min-w-32">
              <div class="flex justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
                <span>{{ row.participant_count }} / {{ row.target_participants }}</span>
                <span>{{ row.winner_count }} {{ t('admin.redPacket.winnersUnit') }}</span>
              </div>
              <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-primary-600" :style="{ width: `${rowProgress(row)}%` }"></div>
              </div>
            </div>
          </template>
          <template #cell-status="{ row }">
            <span :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex min-w-max items-center justify-end gap-1">
              <button
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
                :data-test="`edit-${row.id}`"
                :aria-label="row.status === 'draft' ? t('common.edit') : t('admin.redPacket.view')"
                :title="row.status === 'draft' ? t('common.edit') : t('admin.redPacket.view')"
                @click.stop="openActivity(row)"
              >
                <Icon :name="row.status === 'draft' ? 'edit' : 'eye'" size="sm" aria-hidden="true" />
              </button>
              <button
                v-if="row.status === 'draft'"
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-lg text-primary-700 transition-colors hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-primary-300 dark:hover:bg-primary-950/40"
                :data-test="`publish-${row.id}`"
                :aria-label="t('admin.redPacket.publish')"
                :title="t('admin.redPacket.publish')"
                :disabled="publishingId === row.id"
                @click.stop="handlePublish(row)"
              >
                <Icon :name="publishingId === row.id ? 'refresh' : 'play'" size="sm" :class="publishingId === row.id ? 'animate-spin motion-reduce:animate-none' : ''" aria-hidden="true" />
              </button>
              <button
                v-if="row.status === 'active'"
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-lg text-red-600 transition-colors hover:bg-red-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:text-red-400 dark:hover:bg-red-950/40"
                :data-test="`cancel-${row.id}`"
                :aria-label="t('admin.redPacket.cancel')"
                :title="t('admin.redPacket.cancel')"
                @click.stop="requestCancel(row)"
              >
                <Icon name="ban" size="sm" aria-hidden="true" />
              </button>
              <button
                v-if="row.status === 'completed'"
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
                :aria-label="t('common.export')"
                :title="t('common.export')"
                @click.stop="handleExport(row)"
              >
                <Icon name="download" size="sm" aria-hidden="true" />
              </button>
            </div>
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
          @update:page-size="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
    </div>

    <BaseDialog :show="showFormDialog" :title="dialogTitle" width="wide" @close="closeForm">
      <form id="red-packet-activity-form" class="space-y-5" data-test="activity-form" @submit.prevent="handleSave">
        <div v-if="isReadOnly" class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('admin.redPacket.readOnlyPublished') }}
        </div>

        <div data-test="activity-config-panel" class="grid gap-4 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('admin.redPacket.basicInfo') }}</h3>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('admin.redPacket.formHint') }}</p>
          </div>

          <div class="sm:col-span-2">
            <label for="red-packet-name" class="input-label">{{ t('admin.redPacket.name') }}</label>
            <input id="red-packet-name" v-model.trim="form.name" type="text" maxlength="100" class="input" :readonly="isReadOnly" required />
            <p v-if="errors.name" class="input-error-text">{{ errors.name }}</p>
          </div>

          <div class="sm:col-span-2">
            <label for="red-packet-message" class="input-label">{{ t('admin.redPacket.message') }}</label>
            <textarea id="red-packet-message" v-model.trim="form.message" rows="2" maxlength="255" class="input" :readonly="isReadOnly"></textarea>
          </div>

          <fieldset class="sm:col-span-2" :disabled="isReadOnly">
            <legend class="input-label">{{ t('admin.redPacket.packetType') }}</legend>
            <div class="grid gap-3 sm:grid-cols-2">
              <label :class="packetTypeOptionClass('lucky')">
                <input v-model="form.packet_type" class="sr-only" type="radio" value="lucky" data-test="packet-type-lucky" />
                <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300"><Icon name="sparkles" size="sm" aria-hidden="true" /></span>
                <span>
                  <span class="block text-sm font-semibold">{{ t('admin.redPacket.lucky') }}</span>
                  <span class="mt-1 block text-xs leading-5 opacity-80">{{ t('admin.redPacket.luckyDescription') }}</span>
                </span>
              </label>
              <label :class="packetTypeOptionClass('fixed')">
                <input v-model="form.packet_type" class="sr-only" type="radio" value="fixed" data-test="packet-type-fixed" />
                <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-sky-50 text-sky-600 dark:bg-sky-950/40 dark:text-sky-300"><Icon name="users" size="sm" aria-hidden="true" /></span>
                <span>
                  <span class="block text-sm font-semibold">{{ t('admin.redPacket.fixed') }}</span>
                  <span class="mt-1 block text-xs leading-5 opacity-80">{{ t('admin.redPacket.fixedDescription') }}</span>
                </span>
              </label>
            </div>
          </fieldset>

          <div>
            <label for="red-packet-amount" class="input-label">{{ t('admin.redPacket.totalAmount') }}</label>
            <input id="red-packet-amount" v-model.number="form.total_amount_dollars" type="number" min="0.01" step="0.01" class="input tabular-nums" :readonly="isReadOnly" data-test="total-amount" />
            <p v-if="errors.total_amount_dollars" class="input-error-text">{{ errors.total_amount_dollars }}</p>
            <p class="input-hint">{{ t('admin.redPacket.amountPreview', { amount: formatMoney(dollarsToCents(Number(form.total_amount_dollars))) }) }}</p>
          </div>

          <div>
            <label for="red-packet-target" class="input-label">{{ t('admin.redPacket.targetParticipants') }}</label>
            <input id="red-packet-target" v-model.number="form.target_participants" type="number" min="1" step="1" class="input tabular-nums" :readonly="isReadOnly" data-test="target-participants" />
            <p v-if="errors.target_participants" class="input-error-text">{{ errors.target_participants }}</p>
          </div>

          <div>
            <label for="red-packet-winners" class="input-label">{{ t('admin.redPacket.winnerCount') }}</label>
            <input id="red-packet-winners" v-model.number="form.winner_count" type="number" min="1" step="1" class="input tabular-nums" :readonly="isReadOnly" data-test="winner-count" />
            <p v-if="errors.winner_count" class="input-error-text">{{ errors.winner_count }}</p>
          </div>

          <div v-if="form.packet_type === 'fixed'">
            <label class="input-label">{{ t('admin.redPacket.perWinner') }}</label>
            <div class="input flex min-h-11 items-center bg-gray-50 font-medium tabular-nums text-gray-900 dark:bg-dark-800 dark:text-gray-100">
              {{ fixedPerWinner }}
            </div>
          </div>
        </div>

        <section v-if="editingActivity && isReadOnly" class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('admin.redPacket.eligibilitySnapshot') }}</h3>
          <dl class="mt-3 grid gap-3 text-sm sm:grid-cols-3">
            <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.redPacket.rechargeThreshold') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-gray-100">{{ formatMoney(editingActivity.recharge_threshold_cents) }}</dd></div>
            <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.redPacket.pointsThreshold') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-gray-100">{{ editingActivity.invitation_points_threshold }}</dd></div>
            <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.redPacket.pointsCost') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-gray-100">{{ editingActivity.invitation_points_cost }}</dd></div>
          </dl>
        </section>
      </form>

      <template #footer>
        <div class="flex w-full justify-end gap-3">
          <button type="button" class="btn btn-secondary min-h-11" @click="closeForm">{{ t('common.cancel') }}</button>
          <button v-if="!isReadOnly" type="submit" form="red-packet-activity-form" class="btn btn-primary min-h-11" :disabled="saving">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin motion-reduce:animate-none" aria-hidden="true" />
            {{ t('admin.redPacket.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showCancelDialog"
      :title="t('admin.redPacket.cancel')"
      :message="t('admin.redPacket.cancelConfirm')"
      :confirm-text="t('admin.redPacket.cancel')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmCancel"
      @cancel="closeCancel"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { RedPacketDraft } from '@/api/admin/redPackets'
import type { RedPacketActivity, RedPacketStatus, RedPacketType } from '@/api/redPackets'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

type RedPacketForm = Omit<RedPacketDraft, 'total_amount_cents'> & {
  total_amount_dollars: number
}

const { t } = useI18n()
const appStore = useAppStore()

const activities = ref<RedPacketActivity[]>([])
const loading = ref(false)
const saving = ref(false)
const publishingId = ref<number | null>(null)
const showFormDialog = ref(false)
const showCancelDialog = ref(false)
const editingActivity = ref<RedPacketActivity | null>(null)
const cancelingActivity = ref<RedPacketActivity | null>(null)

const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const form = reactive<RedPacketForm>({
  name: '',
  message: '',
  packet_type: 'lucky',
  total_amount_dollars: 1,
  target_participants: 10,
  winner_count: 1
})
const errors = reactive<Partial<Record<keyof RedPacketForm, string>>>({})

const columns = computed<Column[]>(() => [
  { key: 'period_no', label: t('admin.redPacket.columns.period') },
  { key: 'packet_type', label: t('admin.redPacket.columns.type') },
  { key: 'total_amount_cents', label: t('admin.redPacket.columns.amount') },
  { key: 'progress', label: t('admin.redPacket.columns.progress') },
  { key: 'status', label: t('admin.redPacket.columns.status') },
  { key: 'actions', label: t('common.actions') }
])

const activityOverview = computed(() => [
  {
    key: 'total',
    label: t('admin.redPacket.overview.total'),
    value: pagination.total,
    icon: 'gift' as const,
    iconClass: 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300'
  },
  {
    key: 'active',
    label: t('admin.redPacket.overview.active'),
    value: activities.value.filter(item => item.status === 'active' || item.status === 'drawing').length,
    icon: 'play' as const,
    iconClass: 'bg-primary-50 text-primary-700 dark:bg-primary-950/50 dark:text-primary-300'
  },
  {
    key: 'draft',
    label: t('admin.redPacket.overview.draft'),
    value: activities.value.filter(item => item.status === 'draft').length,
    icon: 'edit' as const,
    iconClass: 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
  },
  {
    key: 'completed',
    label: t('admin.redPacket.overview.completed'),
    value: activities.value.filter(item => item.status === 'completed').length,
    icon: 'checkCircle' as const,
    iconClass: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
  }
])

const hasRunningActivity = computed(() => activities.value.some(activity => activity.status === 'active' || activity.status === 'drawing'))

const isReadOnly = computed(() => Boolean(editingActivity.value && editingActivity.value.status !== 'draft'))
const dialogTitle = computed(() => {
  if (!editingActivity.value) return t('admin.redPacket.create')
  return isReadOnly.value ? t('admin.redPacket.view') : t('admin.redPacket.edit')
})
const fixedPerWinner = computed(() => {
  const winners = Number(form.winner_count)
  const total = dollarsToCents(Number(form.total_amount_dollars))
  if (!Number.isInteger(winners) || winners <= 0 || !Number.isInteger(total) || total < 0 || total % winners !== 0) return '-'
  return formatMoney(total / winners)
})

function dollarsToCents(dollars: number): number {
  return Number.isFinite(dollars) ? Math.round(dollars * 100) : 0
}

function formatMoney(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

function packetTypeLabel(type: RedPacketType): string {
  return t(type === 'fixed' ? 'admin.redPacket.fixed' : 'admin.redPacket.lucky')
}

function statusLabel(status: RedPacketStatus): string {
  return t(`admin.redPacket.status.${status}`)
}

function statusClass(status: RedPacketStatus): string {
  const base = 'inline-flex rounded px-2 py-1 text-xs font-medium'
  if (status === 'draft') return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
  if (status === 'active') return `${base} bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300`
  if (status === 'drawing') return `${base} bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300`
  if (status === 'completed') return `${base} bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300`
  return `${base} bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300`
}

function rowProgress(activity: RedPacketActivity): number {
  if (activity.target_participants <= 0) return 0
  return Math.min(100, Math.max(0, Math.round(activity.participant_count / activity.target_participants * 100)))
}

function packetTypeOptionClass(type: RedPacketType): string {
  return [
    'flex min-h-20 cursor-pointer items-start gap-3 rounded-lg border p-3 text-left transition-colors',
    form.packet_type === type
      ? 'border-primary-300 bg-primary-50/70 text-primary-800 dark:border-primary-800 dark:bg-primary-950/30 dark:text-primary-200'
      : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-400 dark:hover:border-dark-600 dark:hover:bg-dark-800',
    isReadOnly.value ? 'cursor-default opacity-75' : ''
  ].join(' ')
}

async function loadActivities(): Promise<void> {
  loading.value = true
  try {
    const result = await adminAPI.redPackets.list(pagination.page, pagination.page_size)
    activities.value = result.items
    pagination.total = result.total
  } catch (error) {
    appStore.showError(t('admin.redPacket.loadFailed'))
  } finally {
    loading.value = false
  }
}

function resetForm(): void {
  Object.assign(form, {
    name: '',
    message: '',
    packet_type: 'lucky',
    total_amount_dollars: 1,
    target_participants: 10,
    winner_count: 1
  })
  clearErrors()
}

function clearErrors(): void {
  for (const key of Object.keys(errors) as Array<keyof RedPacketForm>) delete errors[key]
}

function openCreate(): void {
  editingActivity.value = null
  resetForm()
  showFormDialog.value = true
}

function openActivity(activity: RedPacketActivity): void {
  editingActivity.value = activity
  Object.assign(form, {
    name: activity.name,
    message: activity.message,
    packet_type: activity.packet_type,
    total_amount_dollars: activity.total_amount_cents / 100,
    target_participants: activity.target_participants,
    winner_count: activity.winner_count
  })
  clearErrors()
  showFormDialog.value = true
}

function closeForm(): void {
  showFormDialog.value = false
  editingActivity.value = null
  clearErrors()
}

function validateForm(): boolean {
  clearErrors()
  const amount = Number(form.total_amount_dollars)
  const amountCents = dollarsToCents(amount)
  const target = Number(form.target_participants)
  const winners = Number(form.winner_count)
  const hasAtMostTwoDecimals = Number.isFinite(amount) && Math.round(amount * 100) / 100 === amount
  if (!form.name.trim()) errors.name = t('admin.redPacket.nameRequired')
  if (!Number.isInteger(target) || target <= 0) errors.target_participants = t('admin.redPacket.positiveInteger')
  if (!Number.isInteger(winners) || winners <= 0) errors.winner_count = t('admin.redPacket.positiveInteger')
  else if (Number.isInteger(target) && winners > target) errors.winner_count = t('admin.redPacket.winnerExceedsTarget')
  if (!Number.isFinite(amount) || amount <= 0 || !hasAtMostTwoDecimals || amountCents <= 0) errors.total_amount_dollars = t('admin.redPacket.positiveAmount')
  else if (Number.isInteger(winners) && winners > 0 && amountCents < winners) errors.total_amount_dollars = t('admin.redPacket.amountTooSmall')
  else if (form.packet_type === 'fixed' && Number.isInteger(winners) && winners > 0 && amountCents % winners !== 0) {
    errors.total_amount_dollars = t('admin.redPacket.fixedDivisible')
  }
  return Object.keys(errors).length === 0
}

async function handleSave(): Promise<void> {
  if (isReadOnly.value || saving.value || !validateForm()) return
  saving.value = true
  try {
    const payload: RedPacketDraft = {
      name: form.name,
      message: form.message,
      packet_type: form.packet_type,
      total_amount_cents: dollarsToCents(Number(form.total_amount_dollars)),
      target_participants: form.target_participants,
      winner_count: form.winner_count
    }
    if (editingActivity.value) await adminAPI.redPackets.update(editingActivity.value.id, payload)
    else await adminAPI.redPackets.create(payload)
    appStore.showSuccess(t(editingActivity.value ? 'admin.redPacket.updated' : 'admin.redPacket.created'))
    closeForm()
    await loadActivities()
  } catch (error) {
    appStore.showError(t('admin.redPacket.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handlePublish(activity: RedPacketActivity): Promise<void> {
  if (activity.status !== 'draft' || publishingId.value != null) return
  if (hasRunningActivity.value) {
    appStore.showError(t('admin.redPacket.publishBlockedByRunning'))
    return
  }
  publishingId.value = activity.id
  try {
    await adminAPI.redPackets.publish(activity.id)
    appStore.showSuccess(t('admin.redPacket.published'))
    await loadActivities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.redPacket.publishFailed')))
  } finally {
    publishingId.value = null
  }
}

function requestCancel(activity: RedPacketActivity): void {
  cancelingActivity.value = activity
  showCancelDialog.value = true
}

function closeCancel(): void {
  showCancelDialog.value = false
  cancelingActivity.value = null
}

async function confirmCancel(): Promise<void> {
  if (!cancelingActivity.value) return
  try {
    await adminAPI.redPackets.cancel(cancelingActivity.value.id)
    appStore.showSuccess(t('admin.redPacket.canceled'))
    closeCancel()
    await loadActivities()
  } catch (error) {
    appStore.showError(t('admin.redPacket.cancelFailed'))
  }
}

async function handleExport(activity: RedPacketActivity): Promise<void> {
  try {
    const blob = await adminAPI.redPackets.export(activity.id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `red-packet-period-${activity.period_no}.csv`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    appStore.showError(t('admin.redPacket.exportFailed'))
  }
}

function handlePageChange(page: number): void {
  pagination.page = page
  loadActivities()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  loadActivities()
}

onMounted(loadActivities)
</script>
