<template>
  <AppLayout>
    <div data-test="red-packet-workspace" class="mx-auto w-full max-w-[96rem] space-y-5">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">{{ t('redPacket.title') }}</h1>
          <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ t('redPacket.subtitle') }}</p>
        </div>
        <div v-if="currentActivity && !loading" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <span class="h-2 w-2 rounded-full bg-primary-500" aria-hidden="true"></span>
          <span>{{ t('redPacket.period', { period: currentActivity.period_no }) }}</span>
        </div>
      </header>

      <div v-if="loading" class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_22rem]" aria-busy="true">
        <div class="h-[32rem] animate-pulse rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"></div>
        <div class="space-y-5">
          <div class="h-72 animate-pulse rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"></div>
          <div class="h-64 animate-pulse rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"></div>
        </div>
      </div>

      <div v-else class="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <section
          data-test="current-activity-card"
          class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900"
        >
          <template v-if="currentActivity">
            <div class="border-b border-gray-200 bg-gray-50/70 px-5 py-5 dark:border-dark-700 dark:bg-dark-800/50">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
                      <Icon name="gift" size="md" aria-hidden="true" />
                    </span>
                    <span class="rounded bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">
                      {{ packetTypeLabel(currentActivity.packet_type) }}
                    </span>
                    <span :class="statusClass(currentActivity.status)">{{ statusLabel(currentActivity.status) }}</span>
                  </div>
                  <h2 data-test="current-activity-name" class="mt-3 text-xl font-semibold text-gray-950 dark:text-white sm:text-2xl">
                    {{ currentActivity.name }}
                  </h2>
                  <p data-test="current-activity-message" class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ currentActivity.message }}
                  </p>
                </div>
                <span class="shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">#RP{{ String(currentActivity.period_no).padStart(4, '0') }}</span>
              </div>
            </div>

            <dl class="grid grid-cols-1 border-b border-gray-200 dark:border-dark-700 sm:grid-cols-3">
              <div data-test="activity-stat" class="px-5 py-5 sm:border-r sm:border-gray-200 sm:px-7 dark:sm:border-dark-700">
                <dt class="text-sm text-gray-500 dark:text-gray-400">{{ t('redPacket.totalAmount') }}</dt>
                <dd class="mt-1 text-2xl font-semibold tabular-nums text-red-600 dark:text-red-400">{{ formatMoney(currentActivity.total_amount_cents) }}</dd>
              </div>
              <div data-test="activity-stat" class="border-t border-gray-200 px-5 py-5 sm:border-r sm:border-t-0 sm:px-7 dark:border-dark-700 dark:sm:border-dark-700">
                <dt class="text-sm text-gray-500 dark:text-gray-400">{{ t('redPacket.winnerCount') }}</dt>
                <dd class="mt-1 text-2xl font-semibold tabular-nums text-gray-950 dark:text-white">{{ currentActivity.winner_count }} <span class="text-base font-medium">{{ t('redPacket.peopleUnit') }}</span></dd>
              </div>
              <div data-test="activity-stat" class="border-t border-gray-200 px-5 py-5 sm:border-t-0 sm:px-7 dark:border-dark-700">
                <dt class="text-sm text-gray-500 dark:text-gray-400">{{ t('redPacket.activityPeriod') }}</dt>
                <dd class="mt-1 text-2xl font-semibold tabular-nums text-gray-950 dark:text-white">{{ t('redPacket.period', { period: currentActivity.period_no }) }}</dd>
              </div>
            </dl>

            <div class="px-5 py-5 sm:px-7 sm:py-6">
              <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
                <span class="font-medium text-gray-900 dark:text-gray-100">{{ t('redPacket.joinedCount', { count: currentActivity.participant_count }) }}</span>
                <span class="text-gray-500 dark:text-gray-400">{{ t('redPacket.remainingToDraw', { count: remainingParticipants }) }}</span>
              </div>
              <div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700" role="progressbar" :aria-label="t('redPacket.progress')" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
                <div class="h-full rounded-full bg-primary-600 transition-[width] duration-200 motion-reduce:transition-none" :style="{ width: `${progress}%` }"></div>
              </div>
              <div class="mt-5 flex gap-3 rounded-lg border border-primary-200 bg-primary-50/60 px-4 py-3 text-sm leading-6 text-primary-900 dark:border-primary-900/70 dark:bg-primary-950/30 dark:text-primary-100">
                <Icon name="clock" size="sm" class="mt-0.5 shrink-0" aria-hidden="true" />
                <span>{{ t('redPacket.drawDescription', { winners: currentActivity.winner_count }) }}</span>
              </div>
            </div>

            <div class="flex flex-col gap-4 border-t border-gray-200 bg-gray-50/50 px-5 py-4 dark:border-dark-700 dark:bg-dark-800/30 sm:flex-row sm:items-center sm:justify-between sm:px-7">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-950 dark:text-white">
                  {{ currentActivity.has_participated ? t('redPacket.alreadyParticipated') : t('redPacket.currentQualification') }}
                </p>
                <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
                  <template v-if="currentActivity.has_participated">{{ t('redPacket.waitForDraw') }}</template>
                  <template v-else-if="eligibility?.preferred_qualification === 'recharge'">{{ t('redPacket.useRechargeNoPoints') }}</template>
                  <template v-else-if="eligibility?.points_qualified">{{ t('redPacket.pointsCost') }}</template>
                  <template v-else>{{ t('redPacket.notEligible') }}</template>
                </p>
              </div>
              <button
                type="button"
                class="btn btn-primary min-h-11 w-full shrink-0 justify-center sm:w-auto"
                :disabled="!canParticipate || participating"
                data-test="participate-button"
                @click="handleParticipate"
              >
                <Icon v-if="participating" name="refresh" size="sm" class="animate-spin motion-reduce:animate-none" aria-hidden="true" />
                <Icon v-else-if="currentActivity.has_participated" name="check" size="sm" aria-hidden="true" />
                <Icon v-else name="gift" size="sm" aria-hidden="true" />
                {{ currentActivity.has_participated ? t('redPacket.joined') : participating ? t('redPacket.participating') : t('redPacket.participate') }}
              </button>
            </div>
          </template>

          <div v-else class="flex min-h-96 flex-col items-center justify-center px-6 py-12 text-center">
            <span class="flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-gray-400">
              <Icon name="gift" size="lg" aria-hidden="true" />
            </span>
            <p class="mt-4 font-medium text-gray-900 dark:text-white">{{ t('redPacket.noActive') }}</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('redPacket.noActiveHint') }}</p>
          </div>
        </section>

        <aside class="grid gap-5 md:grid-cols-2 xl:grid-cols-1">
          <section data-test="eligibility-panel" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <div class="flex items-center gap-3">
                <Icon name="shield" size="md" class="text-primary-600 dark:text-primary-400" aria-hidden="true" />
                <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('redPacket.eligibility') }}</h2>
              </div>
              <span class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">{{ t('redPacket.eitherMet') }}</span>
            </div>
            <div class="px-5 py-1">
              <div class="flex items-start gap-3 border-b border-gray-100 py-4 dark:border-dark-800">
                <span :class="eligibility?.recharge_qualified ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-gray-100 text-gray-400 dark:bg-dark-800'" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
                  <Icon name="creditCard" size="md" aria-hidden="true" />
                </span>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-3">
                    <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('redPacket.recharge') }}</h3>
                    <span :class="eligibility?.recharge_qualified ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400'" class="text-xs font-medium">
                      {{ eligibility?.recharge_qualified ? t('redPacket.met') : t('redPacket.unmet') }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ eligibility?.recharge_qualified ? t('redPacket.rechargeQualified') : t('redPacket.rechargeRequired') }}</p>
                </div>
              </div>

              <div class="flex items-start gap-3 py-4">
                <span :class="eligibility?.points_qualified ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-gray-100 text-gray-400 dark:bg-dark-800'" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
                  <Icon name="users" size="md" aria-hidden="true" />
                </span>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-3">
                    <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('redPacket.invitationPoints') }}</h3>
                    <span :class="eligibility?.points_qualified ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400'" class="text-xs font-medium">
                      {{ eligibility?.points_qualified ? t('redPacket.available') : t('redPacket.unmet') }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('redPacket.pointsBalanceSummary', { current: eligibility?.lottery_points ?? 0, required: eligibility?.invitation_points_required ?? 2 }) }}</p>
                  <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                    <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('redPacket.pointsRemaining', { count: eligibility?.lottery_points ?? 0 }) }}</span>
                    <span>{{ t('redPacket.pointsCost') }}</span>
                  </div>
                </div>
              </div>
            </div>
            <p class="border-t border-gray-200 bg-primary-50/60 px-5 py-3 text-xs leading-5 text-primary-800 dark:border-dark-700 dark:bg-primary-950/30 dark:text-primary-200">{{ t('redPacket.qualificationEither') }}</p>
          </section>

          <section data-test="rules-panel" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <Icon name="document" size="md" class="text-primary-600 dark:text-primary-400" aria-hidden="true" />
              <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('redPacket.rules') }}</h2>
            </div>
            <ol class="divide-y divide-gray-100 px-5 dark:divide-dark-800">
              <li v-for="(rule, index) in [t('redPacket.ruleEligibility'), t('redPacket.rulePoints'), t('redPacket.ruleDraw'), t('redPacket.ruleCredit')]" :key="index" class="flex gap-3 py-3.5 text-sm leading-6 text-gray-600 dark:text-gray-400">
                <span class="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded bg-primary-50 text-xs font-semibold tabular-nums text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">{{ index + 1 }}</span>
                <span>{{ rule }}</span>
              </li>
            </ol>
          </section>
        </aside>
      </div>

      <section data-test="history-section" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700 sm:px-6">
          <div>
            <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('redPacket.history') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.latestFive') }}</p>
          </div>
          <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ recentActivities.length }} / 5</span>
        </div>

        <div v-if="recentActivities.length" class="overflow-x-auto">
          <table data-test="history-table" class="w-full min-w-full text-left text-sm sm:min-w-[58rem]">
            <thead class="bg-gray-50/80 text-xs font-medium text-gray-500 dark:bg-dark-800/60 dark:text-gray-400">
              <tr>
                <th scope="col" class="px-4 py-3.5 sm:px-6">{{ t('redPacket.activity') }}</th>
                <th scope="col" class="px-2 py-3.5 sm:px-4">{{ t('redPacket.packetType') }}</th>
                <th scope="col" class="px-2 py-3.5 sm:px-4">{{ t('redPacket.totalAmount') }}</th>
                <th scope="col" class="hidden px-4 py-3.5 sm:table-cell">{{ t('redPacket.participantsAndWinners') }}</th>
                <th scope="col" class="hidden px-4 py-3.5 sm:table-cell">{{ t('redPacket.result') }}</th>
                <th scope="col" class="hidden px-4 py-3.5 sm:table-cell">{{ t('redPacket.statusLabel') }}</th>
                <th scope="col" class="px-3 py-3.5 text-right sm:px-6">{{ t('redPacket.action') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
              <tr v-for="item in recentActivities" :key="item.id" data-test="history-row" class="transition-colors duration-150 hover:bg-gray-50/70 dark:hover:bg-dark-800/40">
                <td class="px-4 py-4 sm:px-6">
                  <p data-test="history-activity-name" class="font-medium text-gray-950 dark:text-white">{{ item.name }}</p>
                  <p class="mt-1 hidden text-xs text-gray-500 dark:text-gray-400 sm:block">#RP{{ String(item.period_no).padStart(4, '0') }}<span v-if="item.created_at"> · {{ formatDate(item.created_at) }}</span></p>
                </td>
                <td class="px-2 py-4 text-gray-700 dark:text-gray-300 sm:px-4">{{ packetTypeLabel(item.packet_type) }}</td>
                <td class="px-2 py-4 font-medium tabular-nums text-gray-950 dark:text-white sm:px-4">{{ formatMoney(item.total_amount_cents) }}</td>
                <td class="hidden px-4 py-4 tabular-nums text-gray-700 dark:text-gray-300 sm:table-cell">{{ item.participant_count }} / {{ item.winner_count }}</td>
                <td class="hidden px-4 py-4 sm:table-cell">
                  <span v-if="item.my_reward_cents != null" class="font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">+{{ formatMoney(item.my_reward_cents) }}</span>
                  <span v-else class="text-gray-500 dark:text-gray-400">{{ historyResult(item) }}</span>
                </td>
                <td class="hidden px-4 py-4 sm:table-cell"><span :class="statusClass(item.status)">{{ statusLabel(item.status) }}</span></td>
                <td class="px-3 py-4 text-right sm:px-6">
                  <button type="button" class="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-2 text-sm font-medium text-primary-700 transition-colors hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-primary-300 dark:hover:bg-primary-950/40 sm:px-3" :data-test="`history-detail-${item.id}`" @click="openDetail(item)">
                    <span class="hidden sm:inline">{{ t('redPacket.viewRecords') }}</span>
                    <span class="sm:hidden">{{ t('redPacket.details') }}</span>
                    <Icon name="chevronRight" size="xs" class="hidden sm:block" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="px-6 py-12 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('redPacket.noHistory') }}</div>
      </section>
    </div>

    <BaseDialog :show="showDetail" :title="t('redPacket.records')" width="wide" @close="closeDetail">
      <div v-if="detailLoading" class="flex min-h-64 items-center justify-center" aria-busy="true">
        <Icon name="refresh" size="lg" class="animate-spin text-primary-600 motion-reduce:animate-none" aria-hidden="true" />
      </div>
      <div v-else-if="selectedDetail" class="space-y-5">
        <div class="flex flex-col gap-4 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300"><Icon name="gift" size="md" aria-hidden="true" /></span>
            <div class="min-w-0">
              <p data-test="detail-activity-name" class="truncate font-semibold text-gray-950 dark:text-white">{{ selectedDetail.activity.name }}</p>
              <p data-test="detail-activity-message" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ selectedDetail.activity.message }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.period', { period: selectedDetail.activity.period_no }) }} · {{ packetTypeLabel(selectedDetail.activity.packet_type) }}</p>
            </div>
          </div>
          <span :class="statusClass(selectedDetail.activity.status)">{{ statusLabel(selectedDetail.activity.status) }}</span>
        </div>

        <dl data-test="detail-stats" class="grid grid-cols-2 border-y border-gray-200 dark:border-dark-700 sm:grid-cols-4">
          <div class="min-w-0 py-3 pr-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.totalAmount') }}</dt><dd class="mt-1 font-semibold tabular-nums text-gray-950 dark:text-white">{{ formatMoney(selectedDetail.activity.total_amount_cents) }}</dd></div>
          <div class="min-w-0 border-l border-gray-200 px-4 py-3 dark:border-dark-700"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.targetParticipants') }}</dt><dd class="mt-1 font-semibold tabular-nums text-gray-950 dark:text-white">{{ selectedDetail.activity.target_participants }}</dd></div>
          <div class="min-w-0 border-t border-gray-200 py-3 pr-4 dark:border-dark-700 sm:border-l sm:border-t-0 sm:px-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.winnerCount') }}</dt><dd class="mt-1 font-semibold tabular-nums text-gray-950 dark:text-white">{{ selectedDetail.activity.winner_count }}</dd></div>
          <div class="min-w-0 border-l border-t border-gray-200 px-4 py-3 dark:border-dark-700 sm:border-t-0"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('redPacket.packetType') }}</dt><dd class="mt-1 truncate font-semibold text-gray-950 dark:text-white">{{ packetTypeLabel(selectedDetail.activity.packet_type) }}</dd></div>
        </dl>

        <div v-if="selectedDetail.winners.length" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
          <div class="grid grid-cols-[3rem_minmax(0,1fr)_auto] bg-gray-50 px-4 py-3 text-xs font-medium text-gray-500 dark:bg-dark-800 dark:text-gray-400">
            <span>{{ t('redPacket.rank') }}</span><span>{{ t('redPacket.winner') }}</span><span>{{ t('redPacket.rewardAmount') }}</span>
          </div>
          <div v-for="(winner, index) in selectedDetail.winners" :key="`${winner.masked_username}-${index}`" class="grid grid-cols-[3rem_minmax(0,1fr)_auto] items-center border-t border-gray-100 px-4 py-3 dark:border-dark-800">
            <span class="text-xs tabular-nums text-gray-400">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="min-w-0">
              <p class="flex min-w-0 items-center gap-2">
                <span class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{{ winner.masked_username }}</span>
                <span v-if="winner.is_current_user" data-test="current-user-badge" class="inline-flex shrink-0 items-center rounded bg-primary-50 px-1.5 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-950/60 dark:text-primary-300">{{ t('redPacket.me') }}</span>
              </p>
              <span v-if="winner.is_luckiest" class="mt-1 inline-flex rounded bg-amber-50 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">{{ t('redPacket.luckiest') }}</span>
            </div>
            <span class="shrink-0 text-sm font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">{{ formatMoney(winner.amount_cents) }}</span>
          </div>
        </div>
        <p v-else class="rounded-lg bg-gray-50 py-10 text-center text-sm text-gray-500 dark:bg-dark-800 dark:text-gray-400">{{ t('redPacket.noWinners') }}</p>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary min-h-11" @click="closeDetail">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { redPacketsAPI } from '@/api/redPackets'
import type {
  RedPacketActivity,
  RedPacketActivityDetail,
  RedPacketEligibility,
  RedPacketReward,
  RedPacketStatus,
  RedPacketType
} from '@/api/redPackets'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const currentActivity = ref<RedPacketActivity | null>(null)
const eligibility = ref<RedPacketEligibility | null>(null)
const recentActivities = ref<RedPacketActivity[]>([])
const rewards = ref<RedPacketReward[]>([])
const loading = ref(true)
const participating = ref(false)
const showDetail = ref(false)
const detailLoading = ref(false)
const selectedDetail = ref<RedPacketActivityDetail | null>(null)

const DRAWING_POLL_INTERVAL_MS = 2000
let drawingPollTimer: number | null = null
let drawingPollInFlight = false
let drawingTriggeredLocally = false
let isUnmounted = false

const progress = computed(() => {
  const activity = currentActivity.value
  if (!activity || activity.target_participants <= 0) return 0
  return Math.min(100, Math.max(0, Math.round(activity.participant_count / activity.target_participants * 100)))
})

const remainingParticipants = computed(() => {
  const activity = currentActivity.value
  if (!activity) return 0
  return Math.max(0, activity.target_participants - activity.participant_count)
})

const canParticipate = computed(() => {
  const activity = currentActivity.value
  const qualification = eligibility.value
  return Boolean(
    activity &&
    activity.status === 'active' &&
    !activity.has_participated &&
    qualification &&
    (qualification.recharge_qualified || qualification.points_qualified)
  )
})

function formatMoney(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function packetTypeLabel(type: RedPacketType): string {
  return t(type === 'fixed' ? 'redPacket.fixed' : 'redPacket.lucky')
}

function statusLabel(status: RedPacketStatus): string {
  return t(`redPacket.status.${status}`)
}

function statusClass(status: RedPacketStatus): string {
  const base = 'inline-flex rounded px-2 py-1 text-xs font-medium'
  if (status === 'active') return `${base} bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300`
  if (status === 'drawing') return `${base} bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300`
  if (status === 'completed') return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
  return `${base} bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300`
}

function historyResult(item: RedPacketActivity): string {
  const reward = rewards.value.find(entry => entry.activity_id === item.id)
  if (reward) return `+${formatMoney(reward.amount_cents)}`
  if (!item.has_participated) return t('redPacket.notParticipated')
  if (item.status === 'drawing') return statusLabel(item.status)
  return t('redPacket.notWon')
}

async function loadPage(): Promise<void> {
  loading.value = true
  try {
    const [current, qualification, recent, rewardList] = await Promise.all([
      redPacketsAPI.getCurrent(),
      redPacketsAPI.getEligibility(),
      redPacketsAPI.getRecent(5),
      redPacketsAPI.getRewards(5)
    ])
    currentActivity.value = current
    eligibility.value = qualification
    recentActivities.value = recent.slice(0, 5)
    rewards.value = rewardList
  } catch (error) {
    appStore.showError(t('redPacket.loadFailed'))
  } finally {
    loading.value = false
  }
}

function shouldPollActivity(): boolean {
  const activity = currentActivity.value
  return drawingTriggeredLocally && activity?.status === 'drawing'
}

function stopActivityPolling(): void {
  if (drawingPollTimer !== null) {
    window.clearTimeout(drawingPollTimer)
    drawingPollTimer = null
  }
}

async function refreshActivityState(): Promise<void> {
  const activity = currentActivity.value
  if (!activity || drawingPollInFlight) return

  drawingPollInFlight = true
  try {
    const detail = await redPacketsAPI.getActivity(activity.id)
    const statusChanged = activity.status !== detail.activity.status
    currentActivity.value = detail.activity

    if (statusChanged || detail.activity.status === 'completed' || detail.activity.status === 'canceled') {
      const [recent, rewardList] = await Promise.all([
        redPacketsAPI.getRecent(5),
        redPacketsAPI.getRewards(5)
      ])
      recentActivities.value = recent.slice(0, 5)
      rewards.value = rewardList
    }
    if (detail.activity.status === 'completed' || detail.activity.status === 'canceled') {
      drawingTriggeredLocally = false
    }
  } catch {
    // Background polling retries on the next cycle without interrupting the page.
  } finally {
    drawingPollInFlight = false
  }
}

function scheduleActivityPolling(): void {
  if (isUnmounted || drawingPollTimer !== null || !shouldPollActivity()) return

  drawingPollTimer = window.setTimeout(async () => {
    drawingPollTimer = null
    await refreshActivityState()
    if (shouldPollActivity()) {
      scheduleActivityPolling()
    }
  }, DRAWING_POLL_INTERVAL_MS)
}

async function handleParticipate(): Promise<void> {
  if (!currentActivity.value || !canParticipate.value || participating.value) return
  participating.value = true
  try {
    const result = await redPacketsAPI.participate(currentActivity.value.id)
    currentActivity.value = result.activity
    if (eligibility.value) {
      eligibility.value.lottery_points = result.lottery_points
      eligibility.value.points_qualified = result.lottery_points >= eligibility.value.invitation_points_required
      eligibility.value.preferred_qualification = result.qualification_type
    }
    appStore.showSuccess(t(result.triggered_drawing ? 'redPacket.drawingTriggered' : 'redPacket.participated'))
    recentActivities.value = (await redPacketsAPI.getRecent(5)).slice(0, 5)
    drawingTriggeredLocally = result.triggered_drawing
    scheduleActivityPolling()
  } catch (error) {
    appStore.showError(t('redPacket.participateFailed'))
  } finally {
    participating.value = false
  }
}

async function openDetail(activity: RedPacketActivity): Promise<void> {
  showDetail.value = true
  detailLoading.value = true
  selectedDetail.value = null
  try {
    selectedDetail.value = await redPacketsAPI.getActivity(activity.id)
  } catch (error) {
    appStore.showError(t('redPacket.detailFailed'))
    showDetail.value = false
  } finally {
    detailLoading.value = false
  }
}

function closeDetail(): void {
  showDetail.value = false
  selectedDetail.value = null
}

onMounted(async () => {
  await loadPage()
})

onBeforeUnmount(() => {
  isUnmounted = true
  stopActivityPolling()
})
</script>
