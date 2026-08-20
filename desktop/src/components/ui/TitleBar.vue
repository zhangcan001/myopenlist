<template>
  <div
    class="fixed top-0 right-0 left-0 z-1000 h-8 border-b border-b-border/40 bg-bg-secondary drag-region"
    data-tauri-drag-region
    @dblclick="handleDoubleClick"
  >
    <div data-tauri-drag-region class="flex h-full items-center justify-end px-4 py-0">
      <div class="flex-none" @dblclick.stop>
        <WindowControls @minimize="handleMinimize" @maximize="handleMaximize" @close="handleClose" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getCurrentWindow } from '@tauri-apps/api/window'
import { onMounted, onUnmounted, ref } from 'vue'

import WindowControls from './WindowControls.vue'

const isMaximized = ref(false)
let appWindow: any = null
let unlistenResize: (() => void) | null = null

const handleMinimize = async () => {
  try {
    const appWindow = getCurrentWindow()
    await appWindow.minimize()
  } catch (error) {
    console.error('Error minimizing window:', error)
  }
}

const handleMaximize = async () => {
  try {
    const appWindow = getCurrentWindow()
    const currentMaximized = await appWindow.isMaximized()
    if (currentMaximized) {
      await appWindow.unmaximize()
    } else {
      await appWindow.maximize()
    }
    isMaximized.value = await appWindow.isMaximized()
  } catch (error) {
    console.error('Error toggling window maximize:', error)
  }
}

const handleClose = async () => {
  try {
    const appWindow = getCurrentWindow()
    await appWindow.close()
  } catch (error) {
    console.error('Error closing window:', error)
  }
}

const handleDoubleClick = async () => {
  await handleMaximize()
}

onMounted(async () => {
  appWindow = getCurrentWindow()

  try {
    unlistenResize = await appWindow.listen('tauri://resize', async () => {
      isMaximized.value = await appWindow.isMaximized()
    })

    isMaximized.value = await appWindow.isMaximized()
  } catch (error) {
    console.error('Error setting up window listeners:', error)
  }
})

onUnmounted(() => {
  if (unlistenResize) {
    unlistenResize()
  }
})
</script>
