import { defineStore } from 'pinia'
import { ref } from 'vue'

import { TauriAPI } from '../api/tauri'

export const useRcloneStore = defineStore('rclone', () => {
  const loading = ref(false)
  const error = ref<string | undefined>()
  const rcloneAvailable = ref(true)

  const setError = (msg?: string) => (error.value = msg)

  const clearError = () => setError()

  const checkRcloneAvailable = async () => {
    const available = await TauriAPI.rclone.isAvailable().catch(() => false)
    rcloneAvailable.value = available
    return available
  }

  const init = () => {
    checkRcloneAvailable()
  }

  return {
    loading,
    error,
    rcloneAvailable,
    clearError,
    checkRcloneAvailable,
    init,
  }
})
