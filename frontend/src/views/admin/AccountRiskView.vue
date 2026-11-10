<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 页头 -->
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading || saving" @click="loadConfig">
            <Icon name="refresh" size="sm" class="mr-1.5" :class="loading ? 'animate-spin' : ''" />
            {{ t('admin.accountRisk.reload') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="saving || loading" @click="saveConfig">
            <Icon name="check" size="sm" class="mr-1.5" />
            {{ saving ? t('admin.accountRisk.saving') : t('admin.accountRisk.save') }}
          </button>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <template v-else>
        <!-- 总开关 -->
        <div class="card p-4 sm:p-6">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.accountRisk.master.title') }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.master.hint') }}</p>
            </div>
            <label class="inline-flex cursor-pointer items-center gap-2">
              <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
              <span class="text-sm text-gray-700 dark:text-gray-200">
                {{ form.enabled ? t('admin.accountRisk.master.on') : t('admin.accountRisk.master.off') }}
              </span>
            </label>
          </div>
          <p class="mt-3 text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.accountRisk.master.rules', { rules: rules.length ? rules.join('、') : '—' }) }}
          </p>
        </div>

        <!-- 四条规则 -->
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <!-- 注册即探测 -->
          <div class="card p-4 sm:p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.probe.title') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.probe.hint') }}</p>
              </div>
              <input v-model="form.register_probe.enabled" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600" />
            </div>
            <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="input-label">{{ t('admin.accountRisk.probe.window') }}</label>
                <input v-model.number="form.register_probe.window_minutes" type="number" min="1" max="1440" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.fields.action') }}</label>
                <Select v-model="form.register_probe.action" :options="actionOptionsProbe" />
              </div>
            </div>
          </div>

          <!-- 同 IP 批量注册 -->
          <div class="card p-4 sm:p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.multi.title') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.multi.hint') }}</p>
              </div>
              <input v-model="form.multi_account.enabled" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600" />
            </div>
            <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label class="input-label">{{ t('admin.accountRisk.multi.max') }}</label>
                <input v-model.number="form.multi_account.max_accounts_per_ip" type="number" min="1" max="1000" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.fields.windowHours') }}</label>
                <input v-model.number="form.multi_account.window_hours" type="number" min="1" max="720" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.fields.action') }}</label>
                <Select v-model="form.multi_account.action" :options="actionOptionsMulti" />
              </div>
            </div>
          </div>

          <!-- 会话指纹异常 -->
          <div class="card p-4 sm:p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.session.title') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.session.hint') }}</p>
              </div>
              <input v-model="form.session_abuse.enabled" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600" />
            </div>
            <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label class="input-label">{{ t('admin.accountRisk.session.max') }}</label>
                <input v-model.number="form.session_abuse.max_mismatches" type="number" min="1" max="10000" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.fields.windowHours') }}</label>
                <input v-model.number="form.session_abuse.window_hours" type="number" min="1" max="720" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.fields.action') }}</label>
                <Select v-model="form.session_abuse.action" :options="actionOptionsSession" />
              </div>
            </div>
          </div>

          <!-- 登录防护 -->
          <div class="card p-4 sm:p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.login.title') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.login.hint') }}</p>
              </div>
              <input v-model="form.login_guard.enabled" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600" />
            </div>
            <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
              <div>
                <label class="input-label">{{ t('admin.accountRisk.login.maxIp') }}</label>
                <input v-model.number="form.login_guard.max_ip_failures" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.login.ipWindow') }}</label>
                <input v-model.number="form.login_guard.ip_window_minutes" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.login.lock') }}</label>
                <input v-model.number="form.login_guard.lock_minutes" type="number" min="1" max="1440" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.login.maxAccount') }}</label>
                <input v-model.number="form.login_guard.max_account_failures" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.accountRisk.login.accountWindow') }}</label>
                <input v-model.number="form.login_guard.account_window_minutes" type="number" min="1" class="input" />
              </div>
            </div>
          </div>
        </div>

        <!-- 命中记录 -->
        <div class="card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
            <div>
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accountRisk.events.title') }}</p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.events.hint') }}</p>
            </div>
            <div class="flex items-center gap-2">
              <Select v-model="eventFilter" :options="eventFilterOptions" @change="onFilterChange" />
              <button type="button" class="btn btn-secondary" :disabled="eventsLoading" @click="loadEvents">
                <Icon name="refresh" size="sm" :class="eventsLoading ? 'animate-spin' : ''" />
              </button>
            </div>
          </div>

          <DataTable :columns="eventColumns" :data="events" :loading="eventsLoading" row-key="id">
            <template #cell-created_at="{ value }">
              <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatTime(value) }}</span>
            </template>

            <template #cell-actor_email="{ row }">
              <div class="min-w-0 max-w-[220px]">
                <div class="truncate text-gray-900 dark:text-white" :title="row.actor_email">
                  {{ row.actor_email || '—' }}
                </div>
                <div class="mt-0.5 truncate font-mono text-xs text-gray-400">#{{ row.actor_user_id ?? '—' }}</div>
              </div>
            </template>

            <template #cell-client_ip="{ value }">
              <span class="whitespace-nowrap font-mono text-gray-600 dark:text-gray-300">{{ value || '—' }}</span>
            </template>

            <template #cell-action="{ value }">
              <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="actionBadgeClass(value)">
                {{ eventActionLabel(value) }}
              </span>
            </template>

            <template #cell-rule="{ row }">
              <span class="font-mono text-xs text-gray-700 dark:text-gray-300">{{ ruleLabel(row) }}</span>
            </template>

            <template #cell-reason="{ row }">
              <span class="line-clamp-2 text-xs text-gray-600 dark:text-gray-400" :title="reasonLabel(row)">
                {{ reasonLabel(row) || '—' }}
              </span>
            </template>

            <template #empty>
              <div class="flex flex-col items-center py-8">
                <Icon name="shield" size="xl" class="mb-4 h-12 w-12 text-gray-300 dark:text-dark-600" />
                <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.accountRisk.events.empty') }}</p>
              </div>
            </template>
          </DataTable>

          <div class="border-t border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-6">
            <Pagination
              v-if="eventsTotal > 0"
              :total="eventsTotal"
              :page="eventsPage"
              :page-size="eventsPageSize"
              @update:page="onPageChange"
              @update:pageSize="onPageSizeChange"
            />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AccountRiskConfig } from '@/api/admin'
import type { AuditLog } from '@/api/admin/audit'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const rules = ref<string[]>([])

// 编辑中的配置副本：只有点"保存"才会提交，避免误触开关直接生效。
const form = reactive<AccountRiskConfig>({
  enabled: true,
  register_probe: { enabled: true, window_minutes: 120, action: 'ban' },
  multi_account: { enabled: true, max_accounts_per_ip: 3, window_hours: 24, action: 'block' },
  session_abuse: { enabled: true, max_mismatches: 12, window_hours: 24, action: 'flag' },
  login_guard: {
    enabled: true,
    max_ip_failures: 30,
    ip_window_minutes: 10,
    max_account_failures: 8,
    account_window_minutes: 15,
    lock_minutes: 15
  },
  allowlisted_user_ids: []
})

function applyConfig(config: AccountRiskConfig) {
  form.enabled = config.enabled
  form.allowlisted_user_ids = [...(config.allowlisted_user_ids ?? [])]

  form.register_probe.enabled = config.register_probe.enabled
  form.register_probe.window_minutes = config.register_probe.window_minutes
  form.register_probe.action = config.register_probe.action

  form.multi_account.enabled = config.multi_account.enabled
  form.multi_account.max_accounts_per_ip = config.multi_account.max_accounts_per_ip
  form.multi_account.window_hours = config.multi_account.window_hours
  form.multi_account.action = config.multi_account.action

  form.session_abuse.enabled = config.session_abuse.enabled
  form.session_abuse.max_mismatches = config.session_abuse.max_mismatches
  form.session_abuse.window_hours = config.session_abuse.window_hours
  form.session_abuse.action = config.session_abuse.action

  form.login_guard.enabled = config.login_guard.enabled
  form.login_guard.max_ip_failures = config.login_guard.max_ip_failures
  form.login_guard.ip_window_minutes = config.login_guard.ip_window_minutes
  form.login_guard.max_account_failures = config.login_guard.max_account_failures
  form.login_guard.account_window_minutes = config.login_guard.account_window_minutes
  form.login_guard.lock_minutes = config.login_guard.lock_minutes
}

async function loadConfig() {
  loading.value = true
  try {
    const res = await adminAPI.accountRisk.getConfig()
    applyConfig(res.config)
    rules.value = res.rules ?? []
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || t('admin.accountRisk.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const res = await adminAPI.accountRisk.updateConfig({ ...form })
    // 后端会做区间归一化，回显落库后的真实值。
    applyConfig(res.config)
    appStore.showSuccess(t('admin.accountRisk.saved'))
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || t('admin.accountRisk.saveFailed'))
  } finally {
    saving.value = false
  }
}

// ---------------------------------------------------------------------------
// 命中记录
// ---------------------------------------------------------------------------

const events = ref<AuditLog[]>([])
const eventsTotal = ref(0)
const eventsPage = ref(1)
const eventsPageSize = ref(20)
const eventsLoading = ref(false)
const eventFilter = ref('')

const eventColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.accountRisk.events.columns.time') },
  { key: 'actor_email', label: t('admin.accountRisk.events.columns.user') },
  { key: 'client_ip', label: t('admin.accountRisk.events.columns.ip') },
  { key: 'action', label: t('admin.accountRisk.events.columns.action') },
  { key: 'rule', label: t('admin.accountRisk.events.columns.rule') },
  { key: 'reason', label: t('admin.accountRisk.events.columns.reason') }
])

const eventFilterOptions = computed(() => [
  { value: '', label: t('admin.accountRisk.events.filters.all') },
  { value: 'security.risk.account_banned', label: t('admin.accountRisk.events.filters.banned') },
  { value: 'security.risk.registration_blocked', label: t('admin.accountRisk.events.filters.blocked') },
  { value: 'security.risk.login_locked', label: t('admin.accountRisk.events.filters.loginLocked') },
  { value: 'security.risk.triggered', label: t('admin.accountRisk.events.filters.flagged') }
])

const actionOptionsProbe = computed(() => [
  { value: 'ban', label: t('admin.accountRisk.actions.ban') },
  { value: 'flag', label: t('admin.accountRisk.actions.flag') }
])
const actionOptionsMulti = computed(() => [
  { value: 'block', label: t('admin.accountRisk.actions.block') },
  { value: 'ban', label: t('admin.accountRisk.actions.ban') },
  { value: 'flag', label: t('admin.accountRisk.actions.flag') }
])
const actionOptionsSession = computed(() => [
  { value: 'flag', label: t('admin.accountRisk.actions.flag') },
  { value: 'ban', label: t('admin.accountRisk.actions.ban') }
])

async function loadEvents() {
  eventsLoading.value = true
  try {
    const res = await adminAPI.accountRisk.listEvents({
      page: eventsPage.value,
      page_size: eventsPageSize.value,
      action: eventFilter.value || undefined
    })
    events.value = res.items ?? []
    eventsTotal.value = res.total ?? 0
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || t('admin.accountRisk.events.loadFailed'))
  } finally {
    eventsLoading.value = false
  }
}

function onFilterChange() {
  eventsPage.value = 1
  loadEvents()
}

function onPageChange(next: number) {
  eventsPage.value = next
  loadEvents()
}

function onPageSizeChange(next: number) {
  eventsPageSize.value = next
  eventsPage.value = 1
  loadEvents()
}

function formatTime(value: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function eventActionLabel(action: string): string {
  switch (action) {
    case 'security.risk.account_banned':
      return t('admin.accountRisk.events.actions.banned')
    case 'security.risk.registration_blocked':
      return t('admin.accountRisk.events.actions.blocked')
    case 'security.risk.login_locked':
      return t('admin.accountRisk.events.actions.loginLocked')
    default:
      return t('admin.accountRisk.events.actions.flagged')
  }
}

function actionBadgeClass(action: string): string {
  switch (action) {
    case 'security.risk.account_banned':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'security.risk.registration_blocked':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'security.risk.login_locked':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

function ruleLabel(row: AuditLog): string {
  const extra = (row.extra ?? {}) as Record<string, any>
  if (typeof extra.rules === 'string' && extra.rules) return extra.rules
  return '—'
}

function reasonLabel(row: AuditLog): string {
  const extra = (row.extra ?? {}) as Record<string, any>
  if (typeof extra.reason === 'string' && extra.reason) return extra.reason
  return ''
}

onMounted(async () => {
  await loadConfig()
  await loadEvents()
})
</script>
