<template>
  <div
    v-if="isLoading"
    class="fixed inset-0 flex items-center justify-center z-9999 overflow-hidden bg-[linear-gradient(135deg,#f57dff_0%,#88adff_30%,#feffe3_100%)]"
  >
    <div class="z-1 text-center">
      <div class="mb-8 flex justify-center">
        <div class="relative w-20 h-20 flex items-center justify-center">
          <div class="relative z-2">
            <img src="./assets/logo.svg" alt="App Logo" class="w-26 h-26" />
          </div>
          <div
            class="absolute -inset-5 bg-[conic-gradient(from_0deg,transparent,rgba(255,255,255,0.2),transparent)] rounded-full"
          ></div>
        </div>
      </div>
      <h1 class="mb-2 text-4xl font-light text-white tracking-wide flex flex-col items-center gap-1">
        <span class="font-semibold bg-linear-to-br from-[#ffffff] to-[#e0e0e0] bg-clip-text text-transparent">{{
          t('app.title')
        }}</span>
      </h1>
      <p class="text-white/80 text-sm mb-8 font-normal">{{ t('app.loading') }}</p>
      <div class="w-50 mx-auto">
        <div class="w-full h-1 bg-white/20 rounded-xs overflow-hidden relative">
          <div class="absolute inset-0 bg-white/30"></div>
          <div
            class="absolute inset-0 bg-linear-to-r from-transparent via-accent/40 to-transparent -translate-x-full animate-[shimmer_3s_infinite]"
          ></div>
        </div>
      </div>
    </div>
  </div>
  <div v-else id="app" class="relative h-full min-h-screen w-full select-none bg-transparent">
    <div
      class="relative flex h-screen overflow-hidden bg-bg"
      :class="{
        'pt-8': !isMacOs,
      }"
    >
      <TitleBar v-if="!isMacOs" />
      <div
        class="pointer-events-none absolute inset-0 -z-1 bg-custom bg-cover bg-fixed bg-center bg-no-repeat opacity-custom blur-custom"
      />
      <Navigation />

      <main class="relative z-1 no-scrollbar h-full flex-1 overflow-scroll bg-bg-secondary">
        <router-view v-slot="{ Component, route }">
          <transition name="page" mode="out-in">
            <component :is="Component" :key="route.path" />
          </transition>
        </router-view>
      </main>
    </div>
    <UIServiceProvider />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

import UIServiceProvider from '@/components/ui/UIServiceProvider.vue'

import { TauriAPI } from './api/tauri'
import Navigation from './components/NavigationPage.vue'
import TitleBar from './components/ui/TitleBar.vue'
import { useTranslation } from './composables/useI18n'
import { useTray } from './composables/useTray'
import { useAppStore } from './stores/app'
import { isMacOs } from './utils/constant'

const appStore = useAppStore()
const { t } = useTranslation()
const { updateTrayMenu } = useTray()
const isLoading = ref(true)

let updateUnlisten: (() => void) | null = null

onMounted(async () => {
  try {
    await appStore.init()
    appStore.applyTheme(appStore.settings.app.theme || 'light')
    await updateTrayMenu(appStore.openlistCoreStatus.running)
    updateUnlisten = await TauriAPI.updater.onBackgroundUpdate(updateInfo => {
      appStore.setUpdateAvailable(true, updateInfo)
    })
  } finally {
    isLoading.value = false
  }
})

onUnmounted(() => {
  try {
    updateUnlisten?.()
  } catch (err) {
    console.warn('Error cleaning up global update listener:', err)
  }
})
</script>
