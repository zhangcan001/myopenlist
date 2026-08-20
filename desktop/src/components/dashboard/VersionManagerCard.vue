<template>
  <div class="flex flex-col gap-4 w-full justify-center p-4">
    <div class="flex gap-2 justify-start items-center">
      <ImportIcon class="text-accent" />
      <h4 class="font-semibold text-main">{{ t('dashboard.versionManager.title') }}</h4>
    </div>
    <div class="flex flex-1 items-center justify-center min-h-0">
      <div class="flex flex-wrap gap-3 w-full">
        <div class="flex flex-col bg-surface rounded-md border border-border-secondary p-4 shadow-sm flex-1">
          <div class="flex justify-between items-start mb-3 gap-2">
            <div class="flex flex-col gap-1.5 flex-1 min-w-0">
              <h4 class="text-sm font-semibold text-main leading-[1.3]">
                {{ t('dashboard.versionManager.openlist') }}
              </h4>
              <span class="text-xs text-secondary font-medium bg-accent/10 p-1 rounded-md w-fit whitespace-nowrap">{{
                currentVersions.openlist
              }}</span>
            </div>
            <button
              :disabled="refreshingOpenList"
              class="flex items-center justify-center w-8 h-8 bg-transparent text-secondary border border-border-secondary rounded-md cursor-pointer shrink-0 not-disabled:hover:bg-accent/20 not-disabled:hover:border-accent/20 disabled:opacity-60 disabled:cursor-not-allowed"
              @click="refreshVersions(true, 'openlist')"
            >
              <component
                :is="RefreshCw"
                :size="16"
                :class="{ 'animate-spin': refreshingOpenList && !loading.openlist }"
              />
            </button>
          </div>
          <div class="flex flex-col gap-2 flex-1">
            <SingleSelect
              v-model="selectedVersions.openlist"
              :key-list="availableVersions.openlist"
              title=""
              :fronticon="false"
              :placeholder="t('dashboard.versionManager.selectVersion')"
            />
            <CustomButton
              type="primary"
              :disabled="
                !selectedVersions.openlist || loading.openlist || selectedVersions.openlist === currentVersions.openlist
              "
              :text="loading.openlist ? t('dashboard.versionManager.updating') : t('dashboard.versionManager.update')"
              :icon="loading.openlist ? Loader : Download"
              @click="updateVersion('openlist')"
            />
          </div>
        </div>

        <div class="flex flex-col bg-surface rounded-md border border-border-secondary p-4 shadow-sm flex-1">
          <div class="flex justify-between items-start mb-3 gap-2">
            <div class="flex flex-col gap-1.5 flex-1 min-w-0">
              <div class="flex flex-col gap-1.5 flex-1 min-w-0">
                <h4 class="text-sm font-semibold text-main leading-[1.3]">
                  {{ t('dashboard.versionManager.rclone') }}
                </h4>
                <span class="text-xs text-secondary font-medium bg-accent/10 p-1 rounded-md w-fit whitespace-nowrap">{{
                  currentVersions.rclone
                }}</span>
              </div>
            </div>
            <button
              :disabled="refreshingRclone"
              class="flex items-center justify-center w-8 h-8 bg-transparent text-secondary border border-border-secondary rounded-md cursor-pointer shrink-0 not-disabled:hover:bg-accent/20 not-disabled:hover:border-accent/20 disabled:opacity-60 disabled:cursor-not-allowed"
              @click="refreshVersions(true, 'rclone')"
            >
              <component :is="RefreshCw" :size="16" :class="{ 'animate-spin': refreshingRclone && !loading.rclone }" />
            </button>
          </div>
          <div class="flex flex-col gap-2 flex-1">
            <SingleSelect
              v-model="selectedVersions.rclone"
              :key-list="availableVersions.rclone"
              title=""
              :fronticon="false"
              :disabled="loading.rclone"
              :placeholder="t('dashboard.versionManager.selectVersion')"
            />
            <CustomButton
              type="primary"
              :disabled="
                !selectedVersions.rclone || loading.rclone || selectedVersions.rclone === currentVersions.rclone
              "
              :text="loading.rclone ? t('dashboard.versionManager.updating') : t('dashboard.versionManager.update')"
              :icon="loading.rclone ? Loader : Download"
              @click="updateVersion('rclone')"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Download, ImportIcon, Loader, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'

import useMessage from '@/hooks/useMessage'

import { TauriAPI } from '../../api/tauri'
import { useTranslation } from '../../composables/useI18n'
import CustomButton from '../common/CustomButton.vue'
import SingleSelect from '../common/SingleSelect.vue'

const { t } = useTranslation()
const message = useMessage()

const currentVersions = ref({
  openlist: 'unknown',
  rclone: 'unknown',
})

const selectedVersions = ref({
  openlist: '',
  rclone: '',
})

const availableVersions = ref({
  openlist: [] as string[],
  rclone: [] as string[],
})

const loading = ref({
  openlist: false,
  rclone: false,
})

const refreshingOpenList = ref(false)
const refreshingRclone = ref(false)

const getCurrentVersions = async () => {
  try {
    const [openlistVersion, rcloneVersion] = await Promise.all([
      TauriAPI.bin.version('openlist'),
      TauriAPI.bin.version('rclone'),
    ])
    currentVersions.value.openlist = openlistVersion || 'unknown'
    currentVersions.value.rclone = rcloneVersion || 'unknown'
  } catch (error) {
    console.error('Failed to fetch current versions:', error)
  }
}

const fetchOpenListVersions = async (force: boolean) => {
  try {
    return await TauriAPI.bin.availableVersions('openlist', force)
  } catch (error) {
    console.error('Failed to fetch OpenList versions:', error)
    return []
  }
}

const fetchRcloneVersions = async (force: boolean) => {
  try {
    const versions = await TauriAPI.bin.availableVersions('rclone', force)
    return versions
  } catch (_error) {
    return []
  }
}

const refreshVersions = async (force: boolean, tool: 'openlist' | 'rclone' | 'all') => {
  await getCurrentVersions()
  try {
    if (tool === 'openlist') {
      refreshingOpenList.value = true
      const openlistVersions = await fetchOpenListVersions(force)
      availableVersions.value.openlist = openlistVersions
      refreshingOpenList.value = false
      return
    }
    if (tool === 'rclone') {
      refreshingRclone.value = true
      const rcloneVersions = await fetchRcloneVersions(force)
      availableVersions.value.rclone = rcloneVersions
      refreshingRclone.value = false
      return
    }
    refreshingOpenList.value = true
    refreshingRclone.value = true
    const [openlistVersions, rcloneVersions] = await Promise.all([
      fetchOpenListVersions(force),
      fetchRcloneVersions(force),
    ])
    availableVersions.value.openlist = openlistVersions
    availableVersions.value.rclone = rcloneVersions
    refreshingOpenList.value = false
    refreshingRclone.value = false
  } catch (error) {
    console.error('Failed to refresh versions:', error)
  }
}

const updateVersion = async (type: 'openlist' | 'rclone') => {
  loading.value[type] = true

  try {
    const result = await TauriAPI.bin.updateVersion(type, selectedVersions.value[type])

    currentVersions.value[type] = selectedVersions.value[type]
    selectedVersions.value[type] = ''
    message.success(t('dashboard.versionManager.updateSuccess', { type: type.charAt(0).toUpperCase() + type.slice(1) }))

    console.log(`Updated ${type}:`, result)
  } catch (error) {
    console.error(`Failed to update ${type}:`, error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    message.error(
      t('dashboard.versionManager.updateError', {
        type: type.charAt(0).toUpperCase() + type.slice(1),
        error: errorMessage,
      }),
    )
  } finally {
    loading.value[type] = false
  }
}

onMounted(() => {
  refreshVersions(false, 'all')
})
</script>
