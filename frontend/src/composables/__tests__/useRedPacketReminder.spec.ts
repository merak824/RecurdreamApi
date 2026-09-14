import { defineComponent, h, nextTick, type PropType, toRef } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'

const { getCurrent } = vi.hoisted(() => ({
  getCurrent: vi.fn()
}))

vi.mock('@/api/redPackets', () => ({
  redPacketsAPI: { getCurrent }
}))

import { useRedPacketReminder } from '@/composables/useRedPacketReminder'

const Host = defineComponent({
  props: {
    userId: { type: Number as PropType<number | null>, default: null },
    path: { type: String, required: true }
  },
  setup(props) {
    const reminder = useRedPacketReminder(toRef(props, 'userId'), toRef(props, 'path'))
    return reminder
  },
  render() {
    return h('div')
  }
})

function mountHost(userId = 7, path = '/dashboard'): VueWrapper<InstanceType<typeof Host>> {
  return mount(Host, { props: { userId, path } }) as VueWrapper<InstanceType<typeof Host>>
}

describe('useRedPacketReminder', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    getCurrent.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('reports an active activity as unread when this user has not seen it', async () => {
    getCurrent.mockResolvedValue({ id: 41, status: 'active' })
    const wrapper = mountHost()

    await flushPromises()

    expect(wrapper.vm.hasUnread).toBe(true)
    wrapper.unmount()
  })

  it('marks the current activity seen when the route is opened', async () => {
    getCurrent.mockResolvedValue({ id: 41, status: 'active' })
    const wrapper = mountHost(7, '/red-packets')

    await flushPromises()

    expect(localStorage.getItem('red-packet:last-seen-activity:7')).toBe('41')
    expect(wrapper.vm.hasUnread).toBe(false)
    wrapper.unmount()
  })

  it('shows a later activity as unread again', async () => {
    localStorage.setItem('red-packet:last-seen-activity:7', '41')
    getCurrent.mockResolvedValue({ id: 42, status: 'active' })
    const wrapper = mountHost()

    await flushPromises()

    expect(wrapper.vm.hasUnread).toBe(true)
    wrapper.unmount()
  })

  it('does not report inactive activities as unread', async () => {
    getCurrent.mockResolvedValue({ id: 41, status: 'completed' })
    const wrapper = mountHost()

    await flushPromises()

    expect(wrapper.vm.hasUnread).toBe(false)
    wrapper.unmount()
  })

  it('isolates seen values by user ID', async () => {
    localStorage.setItem('red-packet:last-seen-activity:7', '41')
    getCurrent.mockResolvedValue({ id: 41, status: 'active' })
    const wrapper = mountHost(8)

    await flushPromises()

    expect(wrapper.vm.hasUnread).toBe(true)
    wrapper.unmount()
  })

  it('hides the reminder for no activity or a failed request', async () => {
    getCurrent.mockResolvedValueOnce(null).mockRejectedValueOnce(new Error('offline'))
    const wrapper = mountHost()

    await flushPromises()
    expect(wrapper.vm.hasUnread).toBe(false)

    await wrapper.vm.refresh()
    expect(wrapper.vm.hasUnread).toBe(false)
    wrapper.unmount()
  })

  it('refreshes on the interval and when the window regains focus', async () => {
    getCurrent.mockResolvedValue({ id: 41, status: 'active' })
    const wrapper = mountHost()

    await flushPromises()
    expect(getCurrent).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(60_000)
    await flushPromises()
    window.dispatchEvent(new Event('focus'))
    await flushPromises()
    expect(getCurrent).toHaveBeenCalledTimes(3)

    wrapper.unmount()
    vi.advanceTimersByTime(60_000)
    expect(getCurrent).toHaveBeenCalledTimes(3)
  })

  it('updates when another tab stores the current activity as seen', async () => {
    getCurrent.mockResolvedValue({ id: 41, status: 'active' })
    const wrapper = mountHost()

    await flushPromises()
    expect(wrapper.vm.hasUnread).toBe(true)

    localStorage.setItem('red-packet:last-seen-activity:7', '41')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'red-packet:last-seen-activity:7',
      newValue: '41'
    }))
    await nextTick()

    expect(wrapper.vm.hasUnread).toBe(false)
    wrapper.unmount()
  })
})
