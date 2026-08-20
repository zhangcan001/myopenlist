<!-- eslint-disable vue/no-v-html -->
<template>
  <div :title="t('update.title')" class="w-full h-full overflow-auto flex items-center justify-center">
    <div class="w-full h-full flex flex-col gap-6">
      <div
        v-if="error"
        class="flex items-center overflow-x-hidden gap-2 justify-start p-4 bg-error/10 rounded-md shadow-md"
      >
        <div>
          <AlertCircle :size="16" class="text-danger" />
        </div>
        <div class="flex min-w-0">
          <span class="text-secondary text-sm font-medium block break-all whitespace-normal">{{ error }}</span>
        </div>
      </div>

      <div
        v-if="!updateCheck?.hasUpdate && lastChecked && !checking && !error"
        class="flex items-center gap-4 p-3 bg-success/10 rounded-xl shadow-md"
      >
        <CheckCircle :size="18" class="text-success" />
        <div class="flex flex-col gap-0.5">
          <h4 class="text-main font-semibold text-sm">{{ t('update.upToDate') }}</h4>
          <p class="text-xs text-secondary font-medium">{{ t('update.lastChecked') }}: {{ formatDate(lastChecked) }}</p>
        </div>
      </div>

      <div
        v-if="backgroundUpdateAvailable"
        class="flex items-center justify-between gap-2 p-3 bg-accent/10 rounded-xl shadow-md"
      >
        <Info :size="18" class="text-accent" />
        <div class="flex items-center gap-2 flex-1">
          <span class="text-sm text-secondary font-semibold">{{ t('update.backgroundUpdateAvailable') }}</span>
        </div>
        <CustomButton type="primary" :text="t('update.showUpdate')" @click="showBackgroundUpdate" />
      </div>

      <SettingSection :icon="RefreshCw" :title="t('update.title')">
        <CustomNavCard noarrow :icon="RefreshCw" :title="t('update.currentVersion')">
          <template #description>
            <div class="flex items-center gap-2">
              <span class="rounded-md bg-accent/30 px-2 py-0.5 text-sm font-semibold text-white"
                >v{{ currentVersion }}</span
              >
            </div>
          </template>
          <template #extra>
            <CustomButton
              :text="checking ? t('update.checking') : t('update.checkForUpdates')"
              type="secondary"
              :disabled="checking || downloading || installing"
              @click="checkForUpdates"
            />
          </template>
        </CustomNavCard>

        <SettingCard p1 class="flex items-center">
          <CustomSwitch
            v-model="autoCheckEnabled"
            no-border
            small
            class="w-full!"
            :title="t('update.autoCheck')"
            @change="toggleAutoCheck"
          />
        </SettingCard>
      </SettingSection>

      <div
        v-if="updateCheck?.hasUpdate && !installing"
        class="flex overflow-hidden flex-1 border border-border-secondary rounded-xl p-2 gap-4"
      >
        <div class="flex flex-col flex-1 overflow-auto no-scrollbar gap-2 rounded-md">
          <div class="flex flex-col gap-2 p-2">
            <div class="flex justify-between items-center">
              <div class="flex gap-2 items-center">
                <Download :size="18" class="text-success" />
                <h4 class="text-secondary text-sm font-semibold">{{ t('update.updateAvailable') }}</h4>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-secondary text-sm font-semibold">v{{ updateCheck.currentVersion }}</span>
                <ArrowRight :size="12" />
                <span class="text-success text-sm font-semibold">{{ updateCheck.latestVersion }}</span>
              </div>
            </div>
            <div class="flex items-center">
              <div class="text-xs text-secondary">
                {{ t('update.releaseDate') }}: {{ formatDate(updateCheck.releaseDate) }}
              </div>
            </div>
          </div>

          <div v-if="updateCheck.releaseNotes" class="w-full border rounded-md border-border-secondary p-1">
            <div class="notes-body" v-html="renderedReleaseNotes"></div>
          </div>

          <div v-if="downloading" class="flex flex-col p-2 border border-border-secondary rounded-md">
            <div class="flex justify-between items-center mb-2">
              <span class="text-sm text-secondary font-semibold">{{ t('update.downloading') }}...</span>
              <span class="text-sm text-secondary font-semibold"
                >{{ Math.round(downloadProgress?.percentage || 0) }}%</span
              >
            </div>
            <div class="w-full h-2 bg-surface rounded-sm overflow-hidden mb-2">
              <div
                class="h-full bg-[linear-gradient(90deg,var(--color-accent)_0%,var(--color-primary)_50%)] transition-[width] duration-medium ease-standard"
                :style="{ width: `${downloadProgress?.percentage || 0}%` }"
              ></div>
            </div>
            <div class="flex justify-between text-xs text-secondary font-medium">
              <span class="text-xs font-medium text-secondary">{{ formatSpeed(downloadProgress?.speed || 0) }}</span>
              <span class="text-xs font-medium text-secondary">
                {{ formatBytes(downloadProgress?.downloaded || 0) }} / {{ formatBytes(downloadProgress?.total || 0) }}
              </span>
            </div>
          </div>

          <div v-if="!downloading" class="flex justify-end">
            <CustomButton
              type="primary"
              :icon="Download"
              :text="t('update.downloadAndInstall')"
              :disabled="!updateCheck?.hasUpdate || checking || downloading || installing"
              @click="downloadAndInstall"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AlertCircle, ArrowRight, CheckCircle, Download, Info, RefreshCw } from 'lucide-vue-next'
import { marked } from 'marked'
import { computed, onMounted, onUnmounted, ref } from 'vue'

import useMessage from '@/hooks/useMessage'
import { formatBytes } from '@/utils/formatters'

import { TauriAPI } from '../../api/tauri'
import { useTranslation } from '../../composables/useI18n'
import { useAppStore } from '../../stores/app'
import CustomButton from '../common/CustomButton.vue'
import CustomNavCard from '../common/CustomNavCard.vue'
import CustomSwitch from '../common/CustomSwitch.vue'
import SettingCard from '../common/SettingCard.vue'
import SettingSection from '../common/SettingSection.vue'

const { t } = useTranslation()
const appStore = useAppStore()
const message = useMessage()

const currentVersion = ref('')
const updateCheck = ref<UpdateCheck | null>(null)
const backgroundUpdateCheck = ref<UpdateCheck | null>(null)
const checking = ref(false)
const downloading = ref(false)
const installing = ref(false)
const downloadProgress = ref<DownloadProgress | null>(null)
const lastChecked = ref<string | null>(null)
const error = ref<string | null>(null)
const autoCheckEnabled = ref(true)
const settingsLoading = ref(false)

const backgroundUpdateAvailable = computed(() => backgroundUpdateCheck.value && !updateCheck.value?.hasUpdate)

let backgroundUpdateUnlisten: (() => void) | null = null

const checkForUpdates = async () => {
  if (checking.value || downloading.value || installing.value) return

  try {
    checking.value = true
    error.value = null

    const result = await TauriAPI.updater.check()
    updateCheck.value = result

    lastChecked.value = new Date().toISOString()

    if (!result.hasUpdate) {
      message.info(t('update.noUpdatesFound'))
    }
  } catch (err: any) {
    console.error('Failed to check for updates:', err)
    error.value = t('update.checkError') + String(err ? `: ${err}` : '')
  } finally {
    checking.value = false
  }
}

const downloadAndInstall = async () => {
  if (!TauriAPI.updater.hasPendingUpdate() || downloading.value || installing.value) return

  try {
    downloading.value = true
    message.info(t('update.startingDownload'))

    await TauriAPI.updater.downloadAndInstall(progress => {
      downloadProgress.value = progress
    })

    downloading.value = false
    installing.value = true
    message.info(t('update.installingUpdate'))

    // Relaunch the application after update is installed
    await TauriAPI.updater.relaunch()
  } catch (err: any) {
    console.error('Failed to download/install update:', err)
    downloading.value = false
    installing.value = false
    error.value = err.message || t('update.installError')
    message.error(t('update.installError'))
  }
}

const toggleAutoCheck = async () => {
  if (settingsLoading.value) return

  try {
    settingsLoading.value = true
    await TauriAPI.updater.setAutoCheck(autoCheckEnabled.value)
  } catch (err: any) {
    console.error('Failed to update auto-check setting:', err)
    autoCheckEnabled.value = !autoCheckEnabled.value
  } finally {
    settingsLoading.value = false
  }
}

const showBackgroundUpdate = async () => {
  if (backgroundUpdateCheck.value) {
    // Re-check to get the actual Update object stored in TauriAPI
    try {
      const result = await TauriAPI.updater.check()
      updateCheck.value = result
      backgroundUpdateCheck.value = null
    } catch (err) {
      console.error('Failed to fetch update info:', err)
    }
  }
}

const formatDate = (dateString: string) => {
  try {
    const date = new Date(dateString)
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
  } catch {
    return dateString
  }
}

const renderedReleaseNotes = computed(() => {
  return marked(updateCheck.value?.releaseNotes || '', { breaks: true, gfm: true })
})

const formatSpeed = (bytesPerSecond: number) => {
  if (bytesPerSecond === 0) return '0 B/s'
  const k = 1024
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k))
  return parseFloat((bytesPerSecond / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(async () => {
  try {
    if (appStore.updateAvailable && appStore.updateCheck) {
      updateCheck.value = appStore.updateCheck
      try {
        await TauriAPI.updater.check()
      } catch (err) {
        console.warn('Failed to fetch update object:', err)
      }
    }
    appStore.clearUpdateStatus()
    currentVersion.value = await TauriAPI.updater.currentVersion()
    autoCheckEnabled.value = await TauriAPI.updater.isAutoCheckEnabled()
    try {
      backgroundUpdateUnlisten = await TauriAPI.updater.onBackgroundUpdate(updateInfo => {
        backgroundUpdateCheck.value = updateInfo
      })
    } catch (err) {
      console.warn('Background update listener not available:', err)
      backgroundUpdateUnlisten = null
    }

    if (autoCheckEnabled.value) {
      await checkForUpdates()
    }
  } catch (err) {
    console.error('Failed to initialize update manager:', err)
  }
})

onUnmounted(() => {
  try {
    backgroundUpdateUnlisten?.()
  } catch (err) {
    console.warn('Error unregistering background update listener:', err)
  }
})
</script>

<style scoped>
@import 'tailwindcss' reference;
@import '../../assets/css/index.css' reference;

.notes-body {
  @apply overflow-y-auto rounded-lg p-5 max-h-50 text-base leading-normal bg-bg-tertiary text-secondary;
}

.notes-body :deep(h1),
.notes-body :deep(h2),
.notes-body :deep(h3) {
  @apply font-bold text-main mt-5 mb-2;
}

.notes-body :deep(h1:first-child),
.notes-body :deep(h2:first-child),
.notes-body :deep(h3:first-child) {
  @apply mt-0;
}

.notes-body :deep(p) {
  @apply mb-2;
}

.notes-body :deep(ul),
.notes-body :deep(ol) {
  @apply list-inside my-3.5 pl-6;
}

.notes-body :deep(li) {
  @apply mb-1.5;
}

.notes-body :deep(a) {
  @apply text-accent underline;
}

.notes-body :deep(a:hover) {
  @apply text-accent-hover;
}

.notes-body :deep(img) {
  @apply max-w-full rounded-md;
}
</style>
