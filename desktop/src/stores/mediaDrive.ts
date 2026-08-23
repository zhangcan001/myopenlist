import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { MediaDriveClient, MediaDriveError } from '../api/mediaDrive'
import { TauriAPI } from '../api/tauri'
import {
  type DesktopState,
  mapHealthToDesktopState,
  type MediaDriveHealthSnapshot,
  normalizeDriveLetter,
  translateMediaDriveCode,
  type UserDiagnostic,
} from '../utils/mediaDrive'
import { useAppStore } from './app'

interface SavedMediaDriveConfig {
  driveLetter: string
  autoStart: boolean
  configured: boolean
}

const storageKey = 'openlist.media-drive.desktop'
const defaultConfig: SavedMediaDriveConfig = {
  driveLetter: 'R:',
  autoStart: true,
  configured: false,
}

function loadConfig(): SavedMediaDriveConfig {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey) || '{}') as Partial<SavedMediaDriveConfig>
    const driveLetter = saved.driveLetter ? normalizeDriveLetter(saved.driveLetter) : null
    return {
      driveLetter: driveLetter || defaultConfig.driveLetter,
      autoStart: saved.autoStart ?? defaultConfig.autoStart,
      configured: saved.configured ?? defaultConfig.configured,
    }
  } catch (_error) {
    return { ...defaultConfig }
  }
}

export const useMediaDriveStore = defineStore('mediaDrive', () => {
  const appStore = useAppStore()
  const state = ref<DesktopState>('APP_STARTING')
  const environment = ref<MediaDriveEnvironment>({ windows: false, winfsp_installed: false })
  const health = ref<MediaDriveHealthSnapshot>()
  const error = ref<UserDiagnostic>()
  const config = ref<SavedMediaDriveConfig>(loadConfig())
  const stopped = ref(false)
  const stopping = ref(false)
  const initializing = ref(false)
  const authSessionId = ref<string>()
  const authorizing = ref(false)
  const authPrompted = ref(false)

  const driveLetter = computed(() => config.value.driveLetter)
  const isReady = computed(() => state.value === 'RUNNING' && health.value?.healthy === true)
  const isBusy = computed(() => stopping.value || ['APP_STARTING', 'CHECKING_ENV', 'STARTING'].includes(state.value))
  const authorizationPending = computed(() => Boolean(authSessionId.value))

  function saveConfig(): void {
    localStorage.setItem(storageKey, JSON.stringify(config.value))
  }

  function setError(code: string | undefined): void {
    error.value = translateMediaDriveCode(code)
  }

  function errorCode(caught: unknown): string {
    return caught instanceof MediaDriveError ? caught.code : 'API_UNAVAILABLE'
  }

  function setHealth(nextHealth: MediaDriveHealthSnapshot): void {
    health.value = nextHealth
    state.value = mapHealthToDesktopState(nextHealth, environment.value.winfsp_installed)
    if (nextHealth.diagnostic?.code && state.value !== 'RUNNING') setError(nextHealth.diagnostic.code)
    else if (state.value === 'RUNNING') error.value = undefined
  }

  async function checkEnvironment(): Promise<boolean> {
    environment.value = await TauriAPI.mediaDrive.environment()
    if (!environment.value.winfsp_installed) {
      state.value = 'ERROR'
      setError('WINFSP_UNAVAILABLE')
      return false
    }
    return true
  }

  async function refreshHealth(): Promise<void> {
    if (!appStore.isCoreRunning) {
      state.value = 'READY_TO_START'
      return
    }
    try {
      const client = new MediaDriveClient(appStore.openListCoreUrl)
      setHealth(await client.health())
    } catch (caught) {
      state.value = 'READY_TO_START'
      setError(errorCode(caught))
    }
  }

  async function initialize(): Promise<void> {
    if (initializing.value) return
    initializing.value = true
    state.value = 'CHECKING_ENV'
    error.value = undefined
    stopped.value = false
    try {
      if (await checkEnvironment()) await refreshHealth()
    } catch (caught) {
      state.value = 'ERROR'
      setError(errorCode(caught))
    } finally {
      initializing.value = false
    }
  }

  async function ensureCore(): Promise<void> {
    if (!appStore.isCoreRunning) await appStore.startOpenListCore()
    const deadline = Date.now() + 15_000
    while (!appStore.isCoreRunning && Date.now() < deadline) {
      await new Promise(resolve => setTimeout(resolve, 250))
      await appStore.refreshOpenListCoreStatus()
    }
    if (!appStore.isCoreRunning) throw new MediaDriveError('API_UNAVAILABLE')

    while (Date.now() < deadline) {
      try {
        await new MediaDriveClient(appStore.openListCoreUrl).status()
        return
      } catch (_error) {
        await new Promise(resolve => setTimeout(resolve, 250))
      }
    }
    throw new MediaDriveError('API_UNAVAILABLE')
  }

  async function waitForHealthy(client: MediaDriveClient): Promise<void> {
    const deadline = Date.now() + 30_000
    let lastCode: string | undefined
    while (Date.now() < deadline) {
      try {
        const nextHealth = await client.health()
        setHealth(nextHealth)
        if (nextHealth.healthy) return
        lastCode = nextHealth.diagnostic?.code
        if (nextHealth.state === 'FAILED') break
      } catch (caught) {
        lastCode = errorCode(caught)
      }
      await new Promise(resolve => setTimeout(resolve, 500))
    }
    throw new MediaDriveError(lastCode || 'API_UNAVAILABLE')
  }

  async function start(webdavPassword: string): Promise<void> {
    state.value = 'STARTING'
    error.value = undefined
    stopped.value = false
    const normalized = normalizeDriveLetter(config.value.driveLetter)
    if (!normalized) {
      state.value = 'ERROR'
      setError('INVALID_DRIVE_LETTER')
      return
    }
    if (!webdavPassword) {
      state.value = 'WAITING_AUTH'
      setError('WEBDAV_PASSWORD_REQUIRED')
      return
    }
    try {
      if (!(await checkEnvironment())) return
      await ensureCore()
      const client = new MediaDriveClient(appStore.openListCoreUrl)
      await client.updateMountProfile(normalized)
      await client.start(webdavPassword)
      await waitForHealthy(client)
      config.value.configured = true
      saveConfig()
      state.value = 'RUNNING'
      error.value = undefined
    } catch (caught) {
      state.value = 'ERROR'
      const diagnostic = health.value?.diagnostic
      setError(diagnostic?.code || errorCode(caught))
    }
  }

  async function stop(): Promise<void> {
    stopping.value = true
    error.value = undefined
    try {
      await ensureCore()
      const client = new MediaDriveClient(appStore.openListCoreUrl)
      await client.stop()
      stopped.value = true
      health.value = undefined
      state.value = 'READY_TO_START'
    } catch (caught) {
      state.value = 'ERROR'
      setError(errorCode(caught))
    } finally {
      stopping.value = false
    }
  }

  async function authorize115(): Promise<void> {
    authPrompted.value = true
    authorizing.value = true
    error.value = undefined
    try {
      await ensureCore()
      const client = new MediaDriveClient(appStore.openListCoreUrl)
      const capabilities = await client.authCapabilities()
      if (!capabilities.client_configured) {
        if (!capabilities.token_import_available) {
          setError('CONFIG_REQUIRED')
          return
        }
        let tokens
        try {
          tokens = await TauriAPI.mediaDrive.authorize115Hosted()
        } catch (_error) {
          throw new MediaDriveError('QR_UNAVAILABLE')
        }
        if (!tokens) return
        await client.import115Tokens(tokens.access_token, tokens.refresh_token)
        await refreshHealth()
        return
      }
      const session = await client.start115Auth()
      if (!session.qr_code) throw new MediaDriveError('QR_UNAVAILABLE')
      authSessionId.value = session.session_id
      await TauriAPI.files.urlInBrowser(session.qr_code)
    } catch (caught) {
      setError(errorCode(caught))
    } finally {
      authorizing.value = false
    }
  }

  async function complete115Authorization(): Promise<void> {
    if (!authSessionId.value) return
    authorizing.value = true
    error.value = undefined
    try {
      const client = new MediaDriveClient(appStore.openListCoreUrl)
      await client.complete115Auth(authSessionId.value)
      authSessionId.value = undefined
      await refreshHealth()
    } catch (caught) {
      const code = errorCode(caught)
      if (['SESSION_EXPIRED', 'EXCHANGE_FAILED', 'PERSISTENCE_FAILED', 'STATE_CONFLICT'].includes(code)) {
        authSessionId.value = undefined
      }
      setError(code)
    } finally {
      authorizing.value = false
    }
  }

  function saveUserConfig(drive: string, autoStart: boolean): boolean {
    const normalized = normalizeDriveLetter(drive)
    if (!normalized) {
      setError('INVALID_DRIVE_LETTER')
      return false
    }
    config.value = { driveLetter: normalized, autoStart, configured: true }
    saveConfig()
    return true
  }

  async function openVlc(): Promise<void> {
    try {
      await TauriAPI.mediaDrive.openVlc(driveLetter.value)
    } catch (caught) {
      const code =
        caught instanceof Error ? caught.message : caught === 'VLC_UNAVAILABLE' ? 'VLC_UNAVAILABLE' : undefined
      setError(code === 'VLC_UNAVAILABLE' ? code : 'API_UNAVAILABLE')
    }
  }

  return {
    state,
    environment,
    health,
    error,
    config,
    driveLetter,
    isReady,
    isBusy,
    authorizationPending,
    authorizing,
    authPrompted,
    stopped,
    stopping,
    initialize,
    refreshHealth,
    start,
    stop,
    authorize115,
    complete115Authorization,
    saveUserConfig,
    openVlc,
  }
})
