import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const currentDir = dirname(fileURLToPath(import.meta.url))
const componentSource = readFileSync(resolve(currentDir, '../UserBalanceHistoryModal.vue'), 'utf8')
const zhLocaleSource = readFileSync(resolve(currentDir, '../../../../i18n/locales/zh/admin/overview.ts'), 'utf8')

describe('UserBalanceHistoryModal red packet records', () => {
  it('offers a red packet reward filter and renders the source with a unique key', () => {
    expect(componentSource).toContain("value: 'red_packet_reward'")
    expect(componentSource).toContain("case 'red_packet_reward':")
    expect(componentSource).toContain(':key="`${item.type}:${item.id}`"')
    expect(zhLocaleSource).toContain("typeRedPacketReward: '红包奖励'")
  })

  it('uses balance-history wording instead of recharge-only wording', () => {
    expect(zhLocaleSource).toContain("balanceHistory: '余额明细'")
    expect(zhLocaleSource).toContain("balanceHistoryTitle: '用户余额变动明细'")
  })
})
