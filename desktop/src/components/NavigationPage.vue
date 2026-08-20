<template>
  <nav
    class="group no-scrollbar flex h-screen w-37.5 flex-col overflow-hidden border-r border-r-border-secondary/50 bg-bg-secondary transition-all duration-medium ease-apple gap-2"
  >
    <div class="relative flex items-center justify-center bg-bg-secondary px-4 mt-5">
      <div class="flex flex-col items-center gap-1">
        <span class="font-bold tracking-tight text-main">{{ t('app.title') }}</span>
      </div>
    </div>

    <div class="flex items-center justify-center py-1">
      <ThemeSwitcher />
    </div>

    <div class="flex-1 overflow-y-auto no-scrollbar min-h-9 py-2">
      <router-link
        v-for="item in navigationItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center justify-center py-3 px-4 gap-3 text-sm font-medium no-underline cursor-pointer transition-all duration-fast ease-apple hover:text-accent hover:bg-surface [.router-link-active]:border-accent [.router-link-active]:bg-surface [.router-link-active]:text-accent [.router-link-active]:border-r-4 [.has-notification]:text-success/90"
        :class="{ 'has-notification': item.hasNotification }"
        :title="`${item.name} (${item.shortcut})`"
      >
        <div class="relative flex items-center justify-center">
          <component :is="item.icon" :size="18" />
          <div v-if="item.hasNotification" class="absolute -top-1 -right-1 w-1.5 h-1.5 bg-success rounded-full"></div>
        </div>
        <span>{{ item.name }}</span>
      </router-link>
    </div>

    <div class="absolute bottom-0 p-2 flex justify-center items-center mt-auto">
      <a
        class="flex items-center justify-center p-1 text-main/80 no-underline rounded-full hover:bg-surface hover:text-accent transition-all duration-fast ease-apple"
        title="View on GitHub"
        @click.prevent="openLink('https://github.com/OpenListTeam/openlist-desktop')"
      >
        <BaseSvg name="GitHub" :size="18" />
      </a>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { Download, DownloadCloud, FileText, HardDrive, Home, Settings } from 'lucide-vue-next'
import { computed } from 'vue'

import { TauriAPI } from '@/api/tauri'
import BaseSvg from '@/assets/svg/BaseSvg.vue'
import { createNewWindow } from '@/utils/common'

import { useTranslation } from '../composables/useI18n'
import { useAppStore } from '../stores/app'
import ThemeSwitcher from './ui/ThemeSwitcher.vue'

const { t } = useTranslation()
const appStore = useAppStore()

const navigationItems = computed(() => [
  { name: t('navigation.dashboard'), path: '/', icon: Home, shortcut: 'Ctrl+H' },
  { name: t('navigation.mount'), path: '/mount', icon: HardDrive, shortcut: 'Ctrl+M' },
  { name: t('navigation.logs'), path: '/logs', icon: FileText, shortcut: 'Ctrl+L' },
  { name: t('navigation.settings'), path: '/settings', icon: Settings, shortcut: 'Ctrl+,' },
  {
    name: t('navigation.update'),
    path: '/update',
    icon: appStore.updateAvailable ? DownloadCloud : Download,
    shortcut: 'Ctrl+U',
    hasNotification: appStore.updateAvailable,
  },
])

const isMacOs = computed(() => {
  return typeof OS_PLATFORM !== 'undefined' && OS_PLATFORM === 'darwin'
})

const openLink = async (url: string) => {
  if (appStore.settings.app.open_links_in_browser || isMacOs) {
    try {
      await TauriAPI.files.urlInBrowser(url)
    } catch (error) {
      console.error('Failed to open link:', error)
    }
    return
  }
  createNewWindow(url, `webview-${Date.now()}`, 'External Link')
}
</script>
