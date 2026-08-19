<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <!-- Row 1: Core Stats -->
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- Total API Keys -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
                <Icon name="key" size="md" class="text-blue-600 dark:text-blue-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.apiKeys') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ stats.total_api_keys }}
                </p>
                <p class="text-xs text-green-600 dark:text-green-400">
                  {{ stats.active_api_keys }} {{ t('common.active') }}
                </p>
              </div>
            </div>
          </div>

          <!-- Service Accounts -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
                <Icon name="server" size="md" class="text-purple-600 dark:text-purple-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.accounts') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ stats.total_accounts }}
                </p>
                <p class="text-xs">
                  <span class="text-green-600 dark:text-green-400"
                    >{{ stats.normal_accounts }} {{ t('common.active') }}</span
                  >
                  <span v-if="stats.error_accounts > 0" class="ml-1 text-red-500"
                    >{{ stats.error_accounts }} {{ t('common.error') }}</span
                  >
                </p>
              </div>
            </div>
          </div>

          <!-- Today Requests -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                <Icon name="chart" size="md" class="text-green-600 dark:text-green-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.todayRequests') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ stats.today_requests }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}
                </p>
              </div>
            </div>
          </div>

          <!-- New Users Today -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
                <Icon name="userPlus" size="md" class="text-emerald-600 dark:text-emerald-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.users') }}
                </p>
                <p class="text-xl font-bold text-emerald-600 dark:text-emerald-400">
                  +{{ stats.today_new_users }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('common.total') }}: {{ formatNumber(stats.total_users) }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Row 2: Token Stats -->
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- Today Tokens -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
                <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.todayTokens') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(stats.today_tokens) }}
                </p>
                <p class="text-xs">
                  <span
                    class="text-green-600 dark:text-green-400"
                    :title="t('admin.dashboard.actual')"
                    >${{ formatCost(stats.today_actual_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-orange-500 dark:text-orange-400"
                    :title="t('admin.dashboard.accountCost')"
                    >${{ formatCost(stats.today_account_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-gray-400 dark:text-gray-500"
                    :title="t('admin.dashboard.standard')"
                    >${{ formatCost(stats.today_cost) }}</span
                  >
                </p>
              </div>
            </div>
          </div>

          <!-- Total Tokens -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30">
                <Icon name="database" size="md" class="text-indigo-600 dark:text-indigo-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.totalTokens') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(stats.total_tokens) }}
                </p>
                <p class="text-xs">
                  <span
                    class="text-green-600 dark:text-green-400"
                    :title="t('admin.dashboard.actual')"
                    >${{ formatCost(stats.total_actual_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-orange-500 dark:text-orange-400"
                    :title="t('admin.dashboard.accountCost')"
                    >${{ formatCost(stats.total_account_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-gray-400 dark:text-gray-500"
                    :title="t('admin.dashboard.standard')"
                    >${{ formatCost(stats.total_cost) }}</span
                  >
                </p>
              </div>
            </div>
          </div>

          <!-- Performance (RPM/TPM) -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30">
                <Icon name="bolt" size="md" class="text-violet-600 dark:text-violet-400" :stroke-width="2" />
              </div>
              <div class="flex-1">
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.performance') }}
                </p>
                <div class="flex items-baseline gap-2">
                  <p class="text-xl font-bold text-gray-900 dark:text-white">
                    {{ formatTokens(stats.rpm) }}
                  </p>
                  <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
                </div>
                <div class="flex items-baseline gap-2">
                  <p class="text-sm font-semibold text-violet-600 dark:text-violet-400">
                    {{ formatTokens(stats.tpm) }}
                  </p>
                  <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Avg Response Time -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-rose-100 p-2 dark:bg-rose-900/30">
                <Icon name="clock" size="md" class="text-rose-600 dark:text-rose-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.avgResponse') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatDuration(stats.average_duration_ms) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Profit monitor is part of the dashboard, using the same date range. -->
        <section v-if="profitMonitor" class="space-y-4" data-testid="profit-monitor">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('admin.dashboard.profitMonitor') }}
              </h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.profitMonitorHint') }}
              </p>
            </div>
            <div class="text-right">
              <span
                class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset"
                :class="profitReconciliationClass(profitMonitor.summary.reconciliation_status)"
              >
                <Icon name="shield" size="xs" :stroke-width="2" />
                {{ formatProfitReconciliationStatus(profitMonitor.summary.reconciliation_status) }}
              </span>
              <p v-if="profitMonitor.last_sample_at || profitMonitor.next_sample_at" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                <span v-if="profitMonitor.last_sample_at">{{ t('admin.dashboard.profitLastSample') }} {{ formatSampleTime(profitMonitor.last_sample_at) }}</span>
                <span v-if="profitMonitor.last_sample_at && profitMonitor.next_sample_at"> · </span>
                <span v-if="profitMonitor.next_sample_at">{{ t('admin.dashboard.profitNextSample') }} {{ formatSampleTime(profitMonitor.next_sample_at) }}</span>
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div class="card border-l-4 border-emerald-500 p-4">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profitSales') }}</p>
              <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ formatCost(profitMonitor.summary.sales) }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatNumber(profitMonitor.summary.requests) }} {{ t('admin.dashboard.requests') }}</p>
            </div>
            <div class="card border-l-4 border-amber-500 p-4">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profitCost') }}</p>
              <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ formatCost(profitMonitor.summary.cost) }}</p>
              <p class="mt-1 text-xs text-amber-600 dark:text-amber-400">{{ formatProfitSource(profitMonitor.summary.cost_source) }}</p>
            </div>
            <div class="card border-l-4 p-4" :class="profitMonitor.summary.profit < 0 ? 'border-red-500' : 'border-blue-500'">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profit') }}</p>
              <p class="mt-1 text-xl font-bold" :class="profitMonitor.summary.profit < 0 ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400'">
                ${{ formatCost(profitMonitor.summary.profit) }}
              </p>
              <p v-if="profitMonitor.summary.profit < 0" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ t('admin.dashboard.profitNegative') }}</p>
            </div>
            <div class="card border-l-4 border-violet-500 p-4">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profitMargin') }}</p>
              <p class="mt-1 text-xl font-bold text-violet-600 dark:text-violet-400">{{ formatMargin(profitMonitor.summary.margin_percent) }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profitPendingCostCount') }}: {{ formatNumber(profitMonitor.summary.unknown_cost_count) }}</p>
            </div>
          </div>

          <div class="card p-4">
            <div class="mb-3 flex items-center justify-between">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.profitTrend') }}</h3>
              <div v-if="profitMonitor.summary.upstream_actual_cost !== undefined && profitMonitor.summary.upstream_actual_cost !== null" class="text-right text-xs text-gray-500 dark:text-gray-400">
                <span>{{ t('admin.dashboard.profitUpstreamActualCost') }}: ${{ formatCost(profitMonitor.summary.upstream_actual_cost) }}</span>
                <span v-if="profitMonitor.summary.reconciliation_difference !== undefined && profitMonitor.summary.reconciliation_difference !== null" class="ml-3" :class="profitMonitor.summary.reconciliation_status === 'difference' ? 'text-red-600 dark:text-red-400' : ''">
                  {{ t('admin.dashboard.profitReconciliationDifferenceAmount') }}: {{ formatSignedCost(profitMonitor.summary.reconciliation_difference) }}
                </span>
              </div>
            </div>
            <div class="h-64">
              <Line v-if="profitTrendChartData" :data="profitTrendChartData" :options="profitLineOptions" />
              <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.profitNoData') }}
              </div>
            </div>
          </div>

          <div class="card overflow-hidden">
            <div class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
              <button
                v-for="tab in profitTabs"
                :key="tab.value"
                type="button"
                class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
                :class="profitDimension === tab.value ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-800'"
                @click="profitDimension = tab.value"
              >
                {{ tab.label }}
              </button>
            </div>
            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-200 text-left text-xs dark:divide-dark-700">
                <thead class="bg-gray-50 text-gray-500 dark:bg-dark-800/50 dark:text-gray-400">
                  <tr>
                    <th class="px-4 py-2 font-medium">{{ t('admin.dashboard.profitDimension') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profitRequests') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profitTokens') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profitSales') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profitCost') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profit') }}</th>
                    <th class="px-4 py-2 text-right font-medium">{{ t('admin.dashboard.profitMargin') }}</th>
                    <th class="px-4 py-2 font-medium">{{ t('admin.dashboard.profitSource') }}</th>
                    <th v-if="profitDimension === 'accounts'" class="px-4 py-2 font-medium">{{ t('admin.dashboard.profitReconciliationStatus') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
                  <tr v-for="row in profitRows" :key="`${profitDimension}-${row.id ?? row.name}`" class="text-gray-700 dark:text-gray-300">
                    <td class="max-w-[220px] truncate px-4 py-2 font-medium text-gray-900 dark:text-white" :title="row.name">{{ row.name }}</td>
                    <td class="px-4 py-2 text-right">{{ formatNumber(row.requests) }}</td>
                    <td class="px-4 py-2 text-right">{{ formatTokens(row.tokens) }}</td>
                    <td class="px-4 py-2 text-right">${{ formatCost(row.sales) }}</td>
                    <td class="px-4 py-2 text-right">${{ formatCost(row.cost) }}</td>
                    <td class="px-4 py-2 text-right font-semibold" :class="row.profit < 0 ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'">${{ formatCost(row.profit) }}</td>
                    <td class="px-4 py-2 text-right">{{ formatMargin(row.margin_percent) }}</td>
                    <td
                      class="whitespace-nowrap px-4 py-2"
                      :class="row.cost_source === 'upstream_probe' || row.cost_source === 'official_upstream' ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'"
                    >
                      {{ formatProfitSource(row.cost_source) }}
                    </td>
                    <td v-if="profitDimension === 'accounts'" class="min-w-[180px] px-4 py-2">
                      <span
                        class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset"
                        :class="profitReconciliationClass(row.reconciliation_status)"
                      >
                        {{ formatProfitReconciliationStatus(row.reconciliation_status) }}
                      </span>
                      <div v-if="row.upstream_actual_cost !== undefined && row.upstream_actual_cost !== null" class="mt-1 space-y-0.5 text-[11px] text-gray-500 dark:text-gray-400">
                        <p>{{ t('admin.dashboard.profitUpstreamActualCost') }} ${{ formatCost(row.upstream_actual_cost) }}</p>
                        <p
                          v-if="row.reconciliation_difference !== undefined && row.reconciliation_difference !== null"
                          :class="row.reconciliation_status === 'difference' ? 'font-medium text-red-600 dark:text-red-400' : ''"
                        >
                          {{ t('admin.dashboard.profitReconciliationDifferenceAmount') }} {{ formatSignedCost(row.reconciliation_difference) }}<template v-if="row.reconciliation_difference_percent !== undefined && row.reconciliation_difference_percent !== null"> ({{ formatSignedPercent(row.reconciliation_difference_percent) }})</template>
                        </p>
                      </div>
                      <p v-if="row.last_sample_at || row.next_sample_at" class="mt-1 whitespace-nowrap text-[11px] text-gray-500 dark:text-gray-400">
                        <span v-if="row.last_sample_at">{{ t('admin.dashboard.profitLastSample') }} {{ formatSampleTime(row.last_sample_at) }}</span>
                        <span v-if="row.last_sample_at && row.next_sample_at"> · </span>
                        <span v-if="row.next_sample_at">{{ t('admin.dashboard.profitNextSample') }} {{ formatSampleTime(row.next_sample_at) }}</span>
                      </p>
                    </td>
                  </tr>
                  <tr v-if="profitRows.length === 0">
                    <td :colspan="profitDimension === 'accounts' ? 9 : 8" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.profitNoData') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <!-- Quick Actions -->
        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dashboard.quickActions') }}
            </h2>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <button
              v-if="canUseBatchImage"
              type="button"
              class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-sky-50 dark:bg-dark-800/50 dark:hover:bg-sky-900/20"
              @click="router.push('/batch-image')"
            >
              <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400">
                <Icon name="sparkles" size="md" :stroke-width="2" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.dashboard.batchImage') }}
                </span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.batchImageDesc') }}
                </span>
              </span>
              <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-sky-500" />
            </button>
            <button
              type="button"
              class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-emerald-50 dark:bg-dark-800/50 dark:hover:bg-emerald-900/20"
              @click="router.push('/admin/groups')"
            >
              <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400">
                <Icon name="grid" size="md" :stroke-width="2" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.dashboard.groupPricing') }}
                </span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.groupPricingDesc') }}
                </span>
              </span>
              <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-emerald-500" />
            </button>
          </div>
        </div>

        <!-- Charts Section -->
        <div class="space-y-6">
          <!-- Date Range Filter -->
          <div class="card p-4">
            <div class="flex flex-wrap items-center gap-4">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >{{ t('admin.dashboard.timeRange') }}:</span
                >
                <DateRangePicker
                  v-model:start-date="startDate"
                  v-model:end-date="endDate"
                  @change="onDateRangeChange"
                />
              </div>
              <button @click="loadDashboardStats" :disabled="chartsLoading" class="btn btn-secondary">
                {{ t('common.refresh') }}
              </button>
              <div class="ml-auto flex items-center gap-2">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >{{ t('admin.dashboard.granularity') }}:</span
                >
                <div class="w-28">
                  <Select
                    v-model="granularity"
                    :options="granularityOptions"
                    @change="loadChartData"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Charts Grid -->
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <ModelDistributionChart
              :model-stats="modelStats"
              :enable-ranking-view="true"
              :ranking-items="rankingItems"
              :ranking-total-actual-cost="rankingTotalActualCost"
              :ranking-total-requests="rankingTotalRequests"
              :ranking-total-tokens="rankingTotalTokens"
              :loading="chartsLoading"
              :ranking-loading="rankingLoading"
              :ranking-error="rankingError"
              :start-date="startDate"
              :end-date="endDate"
              @ranking-click="goToUserUsage"
            />
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
          </div>

          <!-- User Usage Trend (Full Width) -->
          <div class="card p-4">
            <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dashboard.recentUsage') }} (Top 12)
            </h3>
            <div class="h-64">
              <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                <LoadingSpinner size="md" />
              </div>
              <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
              <div
                v-else
                class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
              >
                {{ t('admin.dashboard.noDataAvailable') }}
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
import { adminAPI } from '@/api/admin'
import type { ProfitMonitorDimensionStat, ProfitMonitorResponse } from '@/api/admin/dashboard'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
const profitMonitor = ref<ProfitMonitorResponse | null>(null)
const profitDimension = ref<'groups' | 'models' | 'accounts'>('groups')
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
let profitRefreshTimer: ReturnType<typeof setTimeout> | undefined
const rankingLimit = 12

const profitTabs = computed(() => [
  { value: 'groups' as const, label: t('admin.dashboard.profitGroups') },
  { value: 'models' as const, label: t('admin.dashboard.profitModels') },
  { value: 'accounts' as const, label: t('admin.dashboard.profitAccounts') }
])

const profitRows = computed<ProfitMonitorDimensionStat[]>(() => {
  if (!profitMonitor.value) return []
  return profitMonitor.value[profitDimension.value] || []
})

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

const profitLineOptions = computed(() => ({
  ...lineOptions.value,
  plugins: {
    ...lineOptions.value.plugins,
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.dataset.label}: $${formatCost(Number(context.raw))}`
      }
    }
  },
  scales: {
    ...lineOptions.value.scales,
    y: {
      ...lineOptions.value.scales.y,
      ticks: {
        ...lineOptions.value.scales.y.ticks,
        callback: (value: string | number) => `$${formatCost(Number(value))}`
      }
    }
  }
}))

const profitTrendChartData = computed(() => {
  const trend = profitMonitor.value?.trend || []
  if (!trend.length) return null
  return {
    labels: trend.map((point) => point.date),
    datasets: [
      {
        label: t('admin.dashboard.profitSales'),
        data: trend.map((point) => point.sales),
        borderColor: '#10b981',
        backgroundColor: '#10b98120',
        tension: 0.3,
        fill: false
      },
      {
        label: t('admin.dashboard.profitCost'),
        data: trend.map((point) => point.cost),
        borderColor: '#f59e0b',
        backgroundColor: '#f59e0b20',
        tension: 0.3,
        fill: false
      },
      {
        label: t('admin.dashboard.profit'),
        data: trend.map((point) => point.profit),
        borderColor: '#3b82f6',
        backgroundColor: '#3b82f620',
        tension: 0.3,
        fill: false
      }
    ]
  }
})

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#3b82f6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatMargin = (value: number | null | undefined): string => {
  if (value === undefined || value === null || !Number.isFinite(Number(value))) return '-'
  return `${Number(value).toFixed(1)}%`
}

const formatSampleTime = (value: string): string => {
  const sampleTime = new Date(value)
  if (Number.isNaN(sampleTime.getTime())) return '-'
  return sampleTime.toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

const formatProfitSource = (value: string): string => {
  switch (value) {
    case 'upstream_probe':
      return t('admin.dashboard.profitUpstreamProbe')
    case 'group_break_even_estimate':
      return t('admin.dashboard.profitGroupEstimate')
    case 'official_upstream':
      return t('admin.dashboard.profitOfficialUpstream')
    case 'unknown':
      return t('admin.dashboard.profitCostUnknown')
    case 'usage_record_upstream_rate':
      return t('admin.dashboard.profitLocalEstimate')
    case 'channel_pricing':
      return t('admin.dashboard.profitLocalEstimate')
    case 'mixed':
      return t('admin.dashboard.profitMixed')
    case 'legacy_formula':
      return t('admin.dashboard.profitLegacyFormula')
    default:
      return value || t('admin.dashboard.profitCostUnknown')
  }
}

const formatProfitReconciliationStatus = (value: string | undefined): string => {
  switch (value) {
    case 'matched':
      return t('admin.dashboard.profitReconciliationMatched')
    case 'difference':
      return t('admin.dashboard.profitReconciliationDifference')
    case 'pending':
      return t('admin.dashboard.profitReconciliationPending')
    case 'missing_start':
      return t('admin.dashboard.profitReconciliationMissingStart')
    case 'sample_unauthorized':
      return t('admin.dashboard.profitReconciliationSampleUnauthorized')
    case 'sample_failed':
      return t('admin.dashboard.profitReconciliationSampleFailed')
    case 'sample_unsupported':
      return t('admin.dashboard.profitReconciliationSampleUnsupported')
    case 'estimated':
      return t('admin.dashboard.profitReconciliationEstimated')
    case 'official_zero':
      return t('admin.dashboard.profitReconciliationOfficialZero')
    case 'unavailable':
    default:
      return t('admin.dashboard.profitReconciliationUnavailable')
  }
}

const profitReconciliationClass = (value: string | undefined): string => {
  switch (value) {
    case 'matched':
      return 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300 dark:ring-emerald-800'
    case 'difference':
      return 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-900/30 dark:text-red-300 dark:ring-red-800'
    case 'pending':
      return 'bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:ring-amber-800'
    case 'missing_start':
      return 'bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:ring-amber-800'
    case 'sample_unauthorized':
    case 'sample_failed':
      return 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-900/30 dark:text-red-300 dark:ring-red-800'
    case 'sample_unsupported':
      return 'bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-700'
    case 'estimated':
      return 'bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-900/30 dark:text-sky-300 dark:ring-sky-800'
    case 'official_zero':
      return 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300 dark:ring-emerald-800'
    case 'unavailable':
    default:
      return 'bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-700'
  }
}

const formatSignedCost = (value: number | null | undefined): string => {
  if (value === undefined || value === null || !Number.isFinite(Number(value))) return '-'
  const numericValue = Number(value)
  const sign = numericValue >= 0 ? '+' : '-'
  return `${sign}$${formatCost(Math.abs(numericValue))}`
}

const formatSignedPercent = (value: number | null | undefined): string => {
  if (value === undefined || value === null || !Number.isFinite(Number(value))) return '-'
  const numericValue = Number(value)
  const sign = numericValue >= 0 ? '+' : '-'
  return `${sign}${Math.abs(numericValue).toFixed(1)}%`
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

const scheduleProfitRefresh = (nextSampleAt?: string) => {
  if (profitRefreshTimer !== undefined) {
    clearTimeout(profitRefreshTimer)
    profitRefreshTimer = undefined
  }
  if (!nextSampleAt) return

  const target = new Date(nextSampleAt).getTime() + 20_000
  const delay = target - Date.now()
  if (!Number.isFinite(target) || delay <= 0) return

  profitRefreshTimer = setTimeout(() => {
    profitRefreshTimer = undefined
    void loadDashboardSnapshot(false)
  }, delay)
}

// Load data
async function loadDashboardSnapshot(includeStats: boolean) {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false,
      include_profit: true
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
    profitMonitor.value = response.profit || null
    scheduleProfitRefresh(response.profit?.next_sample_at)
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
})

onUnmounted(() => {
  if (profitRefreshTimer !== undefined) {
    clearTimeout(profitRefreshTimer)
  }
})
</script>

<style scoped>
</style>
