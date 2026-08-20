import type { MediaDriveDiagnostic, MediaDriveHealthSnapshot } from '../utils/mediaDrive'

type AdminPasswordProvider = () => Promise<string>

interface ApiResponse<T> {
  code: number
  data: T
}

export interface MediaDriveStatus {
  state: string
  running: boolean
  diagnostic?: MediaDriveDiagnostic
}

export interface MediaDriveMountProfile {
  drive_letter: string
  webdav_url: string
  enabled: boolean
  auto_reconnect: boolean
}

export interface MediaDrive115Capabilities {
  pkce_available: boolean
  token_import_available: boolean
  client_configured: boolean
}

export interface MediaDrive115AuthSession {
  session_id: string
  state: string
  qr_code?: string
  expires_at: string
}

export interface MediaDrive115StorageResult {
  storage_id: number
  mount_path: string
  connected: boolean
  state: string
}

export class MediaDriveError extends Error {
  readonly code: string

  constructor(code: string) {
    super(code)
    this.name = 'MediaDriveError'
    this.code = code
  }
}

function stableCode(value: unknown): string | undefined {
  return typeof value === 'string' && /^[A-Z0-9_]+$/.test(value) ? value : undefined
}

function errorCode(response: Response, body: Partial<ApiResponse<unknown>>): string {
  const data = body.data as { diagnostic?: MediaDriveDiagnostic } | undefined
  return (
    stableCode(data?.diagnostic?.code) ||
    stableCode((body as { message?: unknown }).message) ||
    (response.status === 401 ? 'ADMIN_AUTH_REQUIRED' : 'API_UNAVAILABLE')
  )
}

export class MediaDriveClient {
  private token: string | null = null
  private readonly adminPasswordProvider: AdminPasswordProvider
  private readonly baseUrl: string

  constructor(baseUrl: string, adminPasswordProvider?: AdminPasswordProvider) {
    this.baseUrl = baseUrl
    this.adminPasswordProvider =
      adminPasswordProvider ||
      (async () => {
        const { TauriAPI } = await import('./tauri')
        return TauriAPI.logs.adminPassword()
      })
  }

  private endpoint(path: string): string {
    return `${this.baseUrl.replace(/\/$/, '')}/api/${path.replace(/^\//, '')}`
  }

  private async request<T>(path: string, init: RequestInit = {}, authenticated = true): Promise<T> {
    if (authenticated && !this.token) await this.authenticate()

    const headers = new Headers(init.headers)
    headers.set('Accept', 'application/json')
    if (init.body) headers.set('Content-Type', 'application/json')
    if (authenticated && this.token) headers.set('Authorization', `Bearer ${this.token}`)

    let response: Response
    try {
      response = await fetch(this.endpoint(path), { ...init, credentials: 'omit', headers })
    } catch (_error) {
      throw new MediaDriveError('API_UNAVAILABLE')
    }

    const body = (await response.json().catch(() => {
      throw new MediaDriveError('API_UNAVAILABLE')
    })) as Partial<ApiResponse<T>>

    if (response.status === 401 || body.code === 401) {
      if (authenticated && this.token) {
        this.token = null
        await this.authenticate()
        return this.request<T>(path, init, true)
      }
      throw new MediaDriveError('ADMIN_AUTH_REQUIRED')
    }
    if (!response.ok || body.code !== 200) throw new MediaDriveError(errorCode(response, body))
    return body.data as T
  }

  private async authenticate(): Promise<void> {
    let password: string
    try {
      password = await this.adminPasswordProvider()
    } catch (_error) {
      throw new MediaDriveError('ADMIN_AUTH_REQUIRED')
    }
    if (!password) throw new MediaDriveError('ADMIN_AUTH_REQUIRED')

    const data = await this.request<{ token: string }>(
      'auth/login',
      {
        method: 'POST',
        body: JSON.stringify({ username: 'admin', password }),
      },
      false,
    )
    if (!data.token) throw new MediaDriveError('ADMIN_AUTH_REQUIRED')
    this.token = data.token
  }

  health(): Promise<MediaDriveHealthSnapshot> {
    return this.request<MediaDriveHealthSnapshot>('admin/media-drive/health')
  }

  status(): Promise<MediaDriveStatus> {
    return this.request<MediaDriveStatus>('admin/media-drive/status')
  }

  start(webdavPassword: string): Promise<MediaDriveStatus> {
    return this.request<MediaDriveStatus>('admin/media-drive/start', {
      method: 'POST',
      body: JSON.stringify({ webdav_password: webdavPassword }),
    })
  }

  stop(): Promise<MediaDriveStatus> {
    return this.request<MediaDriveStatus>('admin/media-drive/stop', { method: 'POST' })
  }

  authCapabilities(): Promise<MediaDrive115Capabilities> {
    return this.request<MediaDrive115Capabilities>('admin/media-drive/115/auth/capabilities')
  }

  start115Auth(): Promise<MediaDrive115AuthSession> {
    return this.request<MediaDrive115AuthSession>('admin/media-drive/115/auth/start', {
      method: 'POST',
      body: '{}',
    })
  }

  complete115Auth(sessionId: string): Promise<MediaDrive115StorageResult> {
    return this.request<MediaDrive115StorageResult>('admin/media-drive/115/auth/complete', {
      method: 'POST',
      body: JSON.stringify({ session_id: sessionId }),
    })
  }

  mountProfile(): Promise<MediaDriveMountProfile> {
    return this.request<MediaDriveMountProfile>('admin/media-drive/mount/profile')
  }

  updateMountProfile(driveLetter: string): Promise<MediaDriveMountProfile> {
    return this.request<MediaDriveMountProfile>('admin/media-drive/mount/profile', {
      method: 'POST',
      body: JSON.stringify({ drive_letter: driveLetter, enabled: true, auto_reconnect: true }),
    })
  }
}
