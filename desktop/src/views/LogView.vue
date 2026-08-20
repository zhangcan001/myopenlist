<template>
  <div
    class="relative flex h-full w-full items-center justify-center"
    :class="{ fullscreen: isFullscreen, compact: isCompactMode }"
  >
    <div class="relative z-1 flex h-full w-full flex-col items-center justify-start gap-2 rounded-xl border-none p-2">
      <div class="flex w-full items-center justify-between gap-4 rounded-2xl border border-border-secondary px-2 py-1">
        <div class="flex flex-wrap items-center gap-1">
          <CustomButton
            :icon="isPaused ? Play : Pause"
            :title="isPaused ? t('logs.toolbar.resume') : t('logs.toolbar.pause')"
            :type="isPaused ? 'primary' : 'secondary'"
            :class="'border-none'"
            text=""
            @click="togglePause"
          />
          <CustomButton
            :icon="RotateCcw"
            :title="t('logs.toolbar.refresh')"
            type="secondary"
            :class="'border-none'"
            text=""
            @click="refreshLogs"
          />

          <CustomButton
            :icon="Filter"
            :title="t('logs.toolbar.showFilters')"
            :type="showFilters ? 'primary' : 'secondary'"
            :class="'border-none'"
            text=""
            @click="showFilters = !showFilters"
          />

          <CustomButton
            :icon="Settings"
            :title="t('logs.toolbar.settings')"
            :type="showSettings ? 'primary' : 'secondary'"
            :class="'border-none'"
            text=""
            @click="showSettings = !showSettings"
          />
        </div>

        <div class="flex flex-1 items-center justify-center">
          <div class="relative flex max-w-100 w-full items-center">
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="t('logs.search.placeholder')"
              class="w-full rounded-lg border border-border-secondary bg-bg-secondary px-8 py-1 text-sm text-main placeholder:text-secondary focus:border-accent focus:outline-none placeholder:text-xs"
              @keydown.escape="searchQuery = ''"
            />
          </div>
        </div>

        <div class="flex items-center justify-end gap-0.5">
          <div
            class="flex items-center gap-3 text-xs text-secondary font-['SF_Mono,monospace'] border-r border-r-border-secondary pr-3"
          >
            <span class="">{{
              t('logs.stats.logsCount', { filtered: filteredLogs.length, total: appStore.logs.length })
            }}</span>
            <span v-if="selectedEntries.size > 0" class="bg-accent text-white px-2 py-1 rounded-md">
              {{ t('logs.stats.selected', { count: selectedEntries.size }) }}
            </span>
          </div>

          <CustomButton
            :icon="Copy"
            :title="t('logs.toolbar.copyToClipboard')"
            :type="'secondary'"
            :class="'border-none'"
            :disabled="filteredLogs.length === 0"
            text=""
            @click="copyLogsToClipboard"
          />

          <CustomButton
            :icon="Download"
            :title="t('logs.toolbar.exportLogs')"
            :type="'secondary'"
            :class="'border-none'"
            :disabled="filteredLogs.length === 0"
            text=""
            @click="exportLogs"
          />

          <CustomButton
            :icon="Trash2"
            :title="t('logs.toolbar.clearLogs')"
            :type="'danger'"
            :class="'border-none'"
            :disabled="filteredLogs.length === 0 || filterSource === 'all'"
            text=""
            @click="clearLogs"
          />

          <CustomButton
            :icon="FolderOpen"
            :title="t('logs.toolbar.openLogsDirectory')"
            :type="'secondary'"
            :class="'border-none'"
            text=""
            @click="openLogsDirectory"
          />

          <CustomButton
            :icon="isFullscreen ? Minimize2 : Maximize2"
            :title="t('logs.toolbar.toggleFullscreen')"
            :type="'secondary'"
            :class="'border-none'"
            text=""
            @click="toggleFullscreen"
          />
        </div>
      </div>

      <div
        v-if="showFilters"
        class="flex w-full items-center justify-between gap-4 rounded-2xl border border-border-secondary px-4 py-2"
      >
        <div class="flex flex-1 items-center gap-2">
          <SingleSelect
            v-model="filterLevel"
            :key-list="logLevelList.map(item => item.key)"
            title=""
            :fronticon="false"
            :placeholder="logLevelList.find(level => level.key === filterLevel)?.label || filterLevel"
          >
            <template #item="{ item }">
              {{ logLevelList.find(level => level.key === item)?.label || item }}
            </template>
          </SingleSelect>
          <SingleSelect
            v-model="filterSource"
            :key-list="filterSourceOptions.map(item => item.key)"
            title=""
            :fronticon="false"
            :placeholder="filterSourceOptions.find(source => source.key === filterSource)?.label || filterSource"
          >
            <template #item="{ item }">
              {{ filterSourceOptions.find(source => source.key === item)?.label || item }}
            </template>
          </SingleSelect>
        </div>
        <div class="flex items-center gap-3">
          <button
            class="py-1 px-3 border border-border rounded-md bg-bg text-main text-xs cursor-pointer not-disabled:hover:bg-accent/50 not-disabled:hover:text-white disabled:opacity-50 disabled:cursor-not-allowed"
            :disabled="filteredLogs.length === 0"
            @click="selectAllVisible"
          >
            {{ t('logs.filters.actions.selectAll') }}
          </button>

          <button
            class="py-1 px-3 border border-border rounded-md bg-bg text-main text-xs cursor-pointer not-disabled:hover:bg-accent/50 not-disabled:hover:text-white disabled:opacity-50 disabled:cursor-not-allowed"
            :disabled="selectedEntries.size === 0"
            @click="clearSelection"
          >
            {{ t('logs.filters.actions.clearSelection') }}
          </button>

          <label class="flex items-center gap-1.5 cursor-pointer text-xs text-secondary">
            <input
              v-model="autoScroll"
              type="checkbox"
              class="w-3.5 h-3.5 border border-border rounded-xs bg-bg checked:bg-accent checked:border-accent"
            />
            {{ t('logs.filters.actions.autoScroll') }}
          </label>
        </div>
      </div>

      <div
        v-if="showSettings"
        class="flex w-full items-center justify-between gap-4 rounded-2xl border border-border-secondary px-4 py-2"
      >
        <div class="flex items-center gap-2 text-xs">
          <label class="font-medium text-secondary whitespace-nowrap">{{ t('logs.settings.fontSize') }}</label>
          <input v-model="fontSize" type="range" min="10" max="20" class="w-25" />
          <span class="font-['SF_Mono',monospace] text-xs text-tertiary min-w-7.5">{{ fontSize }}px</span>
        </div>

        <div class="flex items-center gap-2 text-xs">
          <label class="font-medium text-secondary whitespace-nowrap">{{ t('logs.settings.maxLines') }}</label>
          <input
            v-model="maxLines"
            type="number"
            min="100"
            max="10000"
            step="100"
            class="py-1 px-2 border border-border rounded-sm bg-bg text-main text-xs"
          />
        </div>
        <div class="flex items-center gap-2 text-xs">
          <label class="flex items-center gap-1.5 cursor-pointer text-xs text-secondary">
            <input
              v-model="isCompactMode"
              type="checkbox"
              class="w-3.5 h-3.5 border border-border rounded-xs bg-bg checked:bg-accent checked:border-accent"
            />
            {{ t('logs.settings.compactMode') }}
          </label>

          <label class="flex items-center gap-1.5 cursor-pointer text-xs text-secondary">
            <input
              v-model="stripAnsiColors"
              type="checkbox"
              class="w-3.5 h-3.5 border border-border rounded-xs bg-bg checked:bg-accent checked:border-accent"
            />
            {{ t('logs.settings.stripAnsiColors') }}
          </label>
        </div>
      </div>
      <div
        class="flex-1 flex flex-col border w-full border-border-secondary shadow-sm rounded-sm overflow-hidden bg-bg"
      >
        <div
          class="grid grid-cols-[120px_60px_80px_1fr_80px] gap-3 py-2 px-4 bg-surface border-b-2 border-b-border text-xs font-semibold text-secondary uppercase tracking-wider items-center"
        >
          <div
            class="overflow-hidden text-ellipsis border-r border-r-border whitespace-nowrap font-['SF_Mono',monospace] text-center"
          >
            {{ t('logs.headers.timestamp') }}
          </div>
          <div class="overflow-hidden text-ellipsis border-r border-r-border whitespace-nowrap text-center">
            {{ t('logs.headers.level') }}
          </div>
          <div class="overflow-hidden text-ellipsis border-r border-r-border whitespace-nowrap text-center">
            {{ t('logs.headers.source') }}
          </div>
          <div
            class="overflow-hidden text-ellipsis border-r border-r-border whitespace-nowrap font-semibold text-center"
          >
            {{ t('logs.headers.message') }}
          </div>
          <div class="overflow-hidden text-ellipsis whitespace-nowrap flex gap-1 justify-center">
            <button
              class="flex items-center justify-center w-5.5 h-5.5 border-none rounded-sm text-secondary cursor-pointer hover:bg-accent/30 hover:text-white"
              :title="t('logs.toolbar.scrollToTop')"
              @click="scrollToTop"
            >
              <ArrowUp :size="14" />
            </button>
            <button
              class="flex items-center justify-center w-5.5 h-5.5 border-none rounded-sm text-secondary cursor-pointer hover:bg-accent/30 hover:text-white"
              :title="t('logs.toolbar.scrollToBottom')"
              @click="scrollToBottom"
            >
              <ArrowDown :size="14" />
            </button>
          </div>
        </div>

        <div
          ref="logContainer"
          class="flex-1 w-full overflow-y-auto font-['SF_Mono',monospace] leading-[1.4]"
          :style="{ fontSize: fontSize + 'px' }"
        >
          <div
            v-if="filteredLogs.length === 0"
            class="flex flex-col items-center justify-center h-full text-tertiary text-center p-10"
          >
            <div class="text-[48px] opacity-50 mb-4">📄</div>
            <h3 class="text-lg font-semibold text-secondary">{{ t('logs.viewer.noLogsFound') }}</h3>
            <p v-if="searchQuery" class="text-lg text-secondary">{{ t('logs.viewer.noLogsMatch') }}</p>
            <p v-else class="text-lg text-secondary">{{ t('logs.viewer.logsWillAppear') }}</p>
          </div>
          <div
            v-for="(log, index) in filteredLogs"
            :key="index"
            class="grid grid-cols-[120px_60px_80px_1fr] gap-3 py-2 px-4 border-b border-b-border-secondary cursor-pointer hover:bg-accent/10"
            :class="{
              'bg-accent/10! border-l-3 border-l-accent pl-4': selectedEntries.has(index),
              'py-1 px-4 text-[11px]': isCompactMode,
              'bg-warning/5': log.level === 'warn',
              'bg-danger/5': log.level === 'error',
            }"
            @click="toggleSelectEntry(index)"
          >
            <div
              class="text-[10px] text-tertiary text-left font-['SF_Mono',monospace] whitespace-nowrap overflow-hidden text-ellipsis"
            >
              <div class="text-center">{{ log.timestamp || '--:--:--' }}</div>
            </div>
            <div class="overflow-hidden text-ellipsis whitespace-nowrap text-center">
              <span
                class="inline-block py-0.5 px-1.5 rounded-sm text-[9px] font-semibold text-center min-w-12.5 border border-transparent [.debug]:bg-warning/20 [.debug]:text-warning [.debug]:border-warning [.warn]:bg-warning/20 [.warn]:text-warning [.warn]:border-warning [.info]:bg-accent/20 [.info]:text-accent [.info]:border-accent [.error]:bg-danger/20 [.error]:text-danger [.error]:border-danger"
                :class="log.level"
              >
                {{ log.level.toUpperCase() }}
              </span>
            </div>
            <div
              class="flex items-center justify-center text-[9px] text-secondary text-center font-semibold uppercase tracking-wider"
              :data-source="log.source"
            >
              {{ log.source }}
            </div>
            <div
              class="flex items-center wrap-break-word whitespace-pre-wrap text-main font-['SF_Mono',Consolas,Monaco,'Courier New',monospace] text-[10px] leading-[1.4]"
            >
              {{ log.message }}
            </div>
          </div>
        </div>
      </div>

      <div
        class="flex rounded-md justify-between items-center w-full py-2 px-2 bg-surface border-t border-t-border text-[11px] text-secondary shrink-0"
      >
        <div class="flex items-center gap-4">
          <span class="flex items-center gap-1">
            {{ t('logs.status.autoScroll') }} {{ autoScroll ? t('logs.status.on') : t('logs.status.off') }}
          </span>
          <span class="flex items-center gap-1">
            {{ t('logs.status.updates') }} {{ isPaused ? t('logs.status.paused') : t('logs.status.live') }}
          </span>
        </div>

        <div class="flex items-center">
          <span class="flex items-center gap-1">
            {{ t('logs.status.showing', { filtered: filteredLogs.length, total: appStore.logs.length }) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core'
import * as chrono from 'chrono-node'
import {
  ArrowDown,
  ArrowUp,
  Copy,
  Download,
  Filter,
  FolderOpen,
  Maximize2,
  Minimize2,
  Pause,
  Play,
  RotateCcw,
  Settings,
  Trash2,
} from 'lucide-vue-next'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import CustomButton from '@/components/common/CustomButton.vue'
import SingleSelect from '@/components/common/SingleSelect.vue'
import useConfirm from '@/hooks/useConfirm'
import useMessage from '@/hooks/useMessage'

import { useTranslation } from '../composables/useI18n'
import { useAppStore } from '../stores/app'

type filterSourceType = 'openlist' | 'rclone' | 'app' | 'all'

const appStore = useAppStore()
const message = useMessage()
const confirm = useConfirm()
const { t } = useTranslation()
const logContainer = ref<HTMLElement>()
const searchInputRef = ref<HTMLInputElement>()
const autoScroll = useLocalStorage('logViewerAutoScroll', true)
const isPaused = ref(false)
const filterLevel = ref<string>(appStore.settings.app.log_filter_level || 'all')
const filterSource = ref<string>(appStore.settings.app.log_filter_source || 'openlist')
const searchQuery = ref('')
const selectedEntries = ref<Set<number>>(new Set())
const showFilters = useLocalStorage('logViewerShowFilters', true)
const showSettings = useLocalStorage('logViewerShowSettings', false)
const fontSize = useLocalStorage('logViewerFontSize', 13)
const maxLines = useLocalStorage('logViewerMaxLines', 1000)
const isCompactMode = useLocalStorage('logViewerCompactMode', false)
const isFullscreen = ref(false)
const stripAnsiColors = useLocalStorage('logViewerStripAnsiColors', true)

let logRefreshInterval: NodeJS.Timeout | null = null
const logLevelList = [
  { key: 'all', label: t('logs.filters.levels.all') },
  { key: 'debug', label: t('logs.filters.levels.debug') },
  { key: 'info', label: t('logs.filters.levels.info') },
  { key: 'warn', label: t('logs.filters.levels.warn') },
  { key: 'error', label: t('logs.filters.levels.error') },
]
const filterSourceOptions = [
  { key: 'all', label: t('logs.filters.sources.all') },
  { key: 'openlist', label: t('logs.filters.sources.openlist') },
  { key: 'rclone', label: t('logs.filters.sources.rclone') },
  { key: 'app', label: t('logs.filters.app') },
]

watch(filterLevel, async newValue => {
  appStore.settings.app.log_filter_level = newValue
  await appStore.saveSettings()
})

watch(filterSource, async newValue => {
  appStore.settings.app.log_filter_source = newValue
  await appStore.saveSettings()
  await appStore.loadLogs(newValue as filterSourceType)
  await scrollToBottom()
})

const openLogsDirectory = async () => {
  try {
    await appStore.openLogsDirectory()
    message.success(t('logs.notifications.openDirectorySuccess'))
  } catch (error) {
    console.error('Failed to open logs directory:', error)
    message.error(t('logs.notifications.openDirectoryFailed'))
  }
}

const stripAnsiCodes = (text: string): string => {
  return text.replace(/\u001b\[[0-9;]*[mGKHF]/g, '')
}

const parseLogEntry = (logText: string) => {
  const cleanText = stripAnsiColors.value ? stripAnsiCodes(logText).trim() : logText.trim()
  const originalText = logText.trim()

  let level = 'info'
  let message = cleanText

  const levelMatch = cleanText.match(/^(WARN|ERROR|INFO|DEBUG|info|debug|warn|error)/i)
  if (levelMatch) {
    level = levelMatch[1].toLowerCase()
  }

  const timestampMatch = cleanText.match(/(\d{4}[-/]\d{2}[-/]\d{2}[T\s-]*\d{2}:\d{2}:\d{2})/)
  const timestamp = timestampMatch
    ? timestampMatch[1]
    : chrono.parseDate(cleanText)?.toISOString().replace('T', ' ').substring(0, 19) || ''
  const source = filterSource.value

  message = message
    .replace(/^(WARN|ERROR|INFO|DEBUG)\s*/i, '')
    .replace(/^\[\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}\]\s*/, '')
    .replace(/^\d{4}\/\d{2}\/\d{2}\s*-\s*\d{2}:\d{2}:\d{2}\s*\|\s*/, '')
    .trim()

  const timeMatch = timestamp.match(/(\d{2}:\d{2}:\d{2})/)
  const displayTime = timeMatch ? timeMatch[1] : timestamp

  return {
    timestamp: displayTime,
    level,
    source,
    message: message || cleanText,
    original: cleanText,
    rawMessage: stripAnsiColors.value ? message : originalText,
    fullTimestamp: timestamp,
  }
}

const filteredLogs = computed(() => {
  let logs = appStore.logs
    .slice(-maxLines.value)
    .filter((log: string | string[]) => !log.includes('/ping'))
    .map(parseLogEntry)

  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase()
    logs = logs.filter(
      (log: any) =>
        log.message.toLowerCase().includes(query) ||
        log.source.toLowerCase().includes(query) ||
        log.level.toLowerCase().includes(query),
    )
  }

  return logs
})

const scrollToBottom = async () => {
  if (autoScroll.value && !isPaused.value && logContainer.value) {
    await nextTick()
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

const scrollToTop = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = 0
  }
}

const clearLogs = async () => {
  const result = await confirm.confirm({
    message: t('logs.messages.confirmClear'),
    title: t('logs.messages.confirmTitle'),
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning',
  })
  if (!result) {
    return
  }
  try {
    await appStore.clearLogs(
      (filterSource.value !== 'all' && filterSource.value !== 'gin'
        ? filterSource.value
        : 'openlist') as filterSourceType,
    )
    selectedEntries.value.clear()
    message.success(t('logs.notifications.clearSuccess'))
  } catch (error) {
    console.error('Failed to clear logs:', error)
    message.error(t('logs.notifications.clearFailed'))
  }
}

const copyLogsToClipboard = async () => {
  let logsToExport = filteredLogs.value

  if (selectedEntries.value.size > 0) {
    logsToExport = filteredLogs.value.filter((_, index) => selectedEntries.value.has(index))
  }

  const logsText = logsToExport
    .map((log: any) => `[${log.timestamp || 'N/A'}] [${log.level.toUpperCase()}] [${log.source}] ${log.message}`)
    .join('\n')

  try {
    await navigator.clipboard.writeText(logsText)
    const count = selectedEntries.value.size > 0 ? selectedEntries.value.size : filteredLogs.value.length
    message.success(t('logs.notifications.copySuccess', { count }))
  } catch (error) {
    console.error('Failed to copy logs:', error)
    message.error(t('logs.notifications.copyFailed'))
  }
}

const exportLogs = () => {
  let logsToExport = filteredLogs.value

  if (selectedEntries.value.size > 0) {
    logsToExport = filteredLogs.value.filter((_, index) => selectedEntries.value.has(index))
  }

  const logsText = logsToExport
    .map((log: any) => `[${log.timestamp || 'N/A'}] [${log.level.toUpperCase()}] [${log.source}] ${log.message}`)
    .join('\n')

  const blob = new Blob([logsText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `openlist-logs-${new Date().toISOString().split('T')[0]}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  const count = selectedEntries.value.size > 0 ? selectedEntries.value.size : filteredLogs.value.length
  message.success(t('logs.notifications.exportSuccess', { count }))
}

const toggleSelectEntry = (index: number) => {
  if (selectedEntries.value.has(index)) {
    selectedEntries.value.delete(index)
  } else {
    selectedEntries.value.add(index)
  }
}

const selectAllVisible = () => {
  filteredLogs.value.forEach((_: any, index: number) => {
    selectedEntries.value.add(index)
  })
}

const clearSelection = () => {
  selectedEntries.value.clear()
}

const togglePause = () => {
  isPaused.value = !isPaused.value
}

const refreshLogs = async () => {
  await appStore.loadLogs(
    (filterSource.value !== 'all' && filterSource.value !== 'gin'
      ? filterSource.value
      : 'openlist') as filterSourceType,
  )
  await scrollToBottom()
  if (isPaused.value) {
    isPaused.value = false
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  if (isFullscreen.value) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

const handleKeydown = (event: KeyboardEvent) => {
  const ctrl = event.ctrlKey
  const key = event.key.toLowerCase()

  if (ctrl) {
    switch (key) {
      case 'f':
        event.preventDefault()
        searchInputRef.value?.focus()
        break
      case 'a':
        event.preventDefault()
        selectAllVisible()
        break
      case 'c':
        if (selectedEntries.value.size > 0) {
          event.preventDefault()
          copyLogsToClipboard()
        }
        break
      case 'r':
        event.preventDefault()
        refreshLogs()
        break
      case 'delete':
        event.preventDefault()
        clearLogs()
        break
    }
  } else {
    switch (key) {
      case 'escape':
        clearSelection()
        searchQuery.value = ''
        break
      case 'home':
        event.preventDefault()
        scrollToTop()
        break
      case 'end':
        event.preventDefault()
        scrollToBottom()
        break
      case 'f11':
        event.preventDefault()
        toggleFullscreen()
        break
      case ' ':
        if (event.target === document.body) {
          event.preventDefault()
          togglePause()
        }
        break
    }
  }
}

onMounted(async () => {
  appStore.loadLogs((filterSource.value !== 'gin' ? filterSource.value : 'openlist') as filterSourceType).then(() => {
    scrollToBottom()
  })

  document.addEventListener('keydown', handleKeydown)

  logRefreshInterval = setInterval(async () => {
    if (!isPaused.value) {
      const oldLength = appStore.logs.length
      await appStore.loadLogs((filterSource.value !== 'gin' ? filterSource.value : 'openlist') as filterSourceType)

      if (appStore.logs.length > oldLength) {
        await scrollToBottom()
      }
    }
  }, 30 * 1000)
})

onUnmounted(() => {
  if (logRefreshInterval) {
    clearInterval(logRefreshInterval)
  }
  document.removeEventListener('keydown', handleKeydown)
})

const unwatchLogs = appStore.$subscribe(mutation => {
  if (mutation.storeId === 'app') {
    const events = Array.isArray(mutation.events) ? mutation.events : [mutation.events]
    if (events.some((event: any) => event?.key === 'logs')) {
      scrollToBottom()
    }
  }
})

onUnmounted(() => {
  unwatchLogs()
})
</script>
