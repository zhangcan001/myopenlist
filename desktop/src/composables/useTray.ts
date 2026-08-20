import { TauriAPI } from '../api/tauri'

export const useTray = () => {
  const updateTrayMenu = async (serviceRunning: boolean) => {
    try {
      await TauriAPI.tray.update(serviceRunning)
    } catch (error) {
      console.error('Failed to update tray menu:', error)
    }
  }

  return {
    updateTrayMenu,
  }
}
