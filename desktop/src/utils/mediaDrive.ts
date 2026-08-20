export type DesktopState =
  'APP_STARTING' | 'CHECKING_ENV' | 'WAITING_AUTH' | 'READY_TO_START' | 'STARTING' | 'RUNNING' | 'DEGRADED' | 'ERROR'

export interface HealthComponent {
  state: string
  ready: boolean
}

export interface MediaDriveHealthSnapshot {
  state: string
  healthy: boolean
  auth: HealthComponent
  webdav: HealthComponent
  mount: HealthComponent
  diagnostic?: MediaDriveDiagnostic
}

export interface MediaDriveDiagnostic {
  module?: string
  code?: string
  reason?: string
  suggestion?: string
}

export interface UserDiagnostic {
  message: string
  suggestion: string
}

const safeDiagnostics: Record<string, UserDiagnostic> = {
  API_UNAVAILABLE: {
    message: 'OpenList 核心服务暂时不可用。',
    suggestion: '确认核心服务已启动，然后重试。',
  },
  ADMIN_AUTH_REQUIRED: {
    message: '桌面端无法连接 OpenList 管理接口，请检查核心服务状态。',
    suggestion: '确认 OpenList 核心服务正在运行后重试。',
  },
  AUTH_REQUIRED: {
    message: '115 登录状态异常，请重新授权。',
    suggestion: '打开 OpenList 网页完成 115 授权，然后重试。',
  },
  TOKEN_PERSISTENCE_FAILED: {
    message: '115 登录状态异常，请重新授权。',
    suggestion: '重新完成 115 授权并等待令牌保存完成。',
  },
  WINFSP_UNAVAILABLE: {
    message: '请安装 WinFsp 组件。',
    suggestion: '安装与系统架构匹配的 WinFsp 后重启程序。',
  },
  MOUNT_CREDENTIALS_REQUIRED: {
    message: '请输入 WebDAV 密码。',
    suggestion: '首次启动或密码未保存时，需要重新输入 WebDAV 密码。',
  },
  WEBDAV_PASSWORD_REQUIRED: {
    message: '请输入 WebDAV 密码。',
    suggestion: 'WebDAV 密码只在本次启动期间使用，不会保存到桌面配置。',
  },
  WEBDAV_PORT_CONFLICT: {
    message: 'WebDAV 端口被占用。',
    suggestion: '关闭占用本地 WebDAV 端口的程序后重试。',
  },
  WEBDAV_PROFILE_DISABLED: {
    message: '媒体服务未启用。',
    suggestion: '启用本地 WebDAV 配置后重试。',
  },
  WEBDAV_NOT_RUNNING: {
    message: '本地 WebDAV 服务未运行。',
    suggestion: '重新启动媒体盘流程。',
  },
  WEBDAV_START_FAILED: {
    message: '本地 WebDAV 服务启动失败。',
    suggestion: '检查 WebDAV 端口和本地配置后重试。',
  },
  INVALID_DRIVE_LETTER: {
    message: '盘符不可用，请选择 A-Z 之间的盘符。',
    suggestion: '选择一个未被其他程序使用的盘符后重试。',
  },
  PUBLIC_WEBDAV_FORBIDDEN: {
    message: '媒体服务只能绑定本机地址。',
    suggestion: '使用 127.0.0.1 的本地 WebDAV 配置。',
  },
  INVALID_LOCALHOST_WEBDAV: {
    message: '媒体服务地址不是本机地址。',
    suggestion: '使用绑定到 127.0.0.1 的本地 WebDAV 配置。',
  },
  MOUNT_NOT_MOUNTED: {
    message: 'R 盘没有成功挂载。',
    suggestion: '检查 WinFsp 和盘符占用情况后重试。',
  },
  MOUNT_FAILED: {
    message: 'Windows 媒体盘挂载失败。',
    suggestion: '检查 WinFsp、盘符和 WebDAV 服务后重试。',
  },
  VLC_UNAVAILABLE: {
    message: '找不到 VLC 播放器。',
    suggestion: '安装 VLC 后再点击“打开 VLC”。',
  },
}

const defaultDiagnostic: UserDiagnostic = {
  message: '媒体盘暂时无法启动，请检查配置。',
  suggestion: '检查 OpenList、115 授权和 WinFsp 状态后重试。',
}

export function mapHealthToDesktopState(
  health: MediaDriveHealthSnapshot | undefined,
  environmentReady: boolean,
): DesktopState {
  if (!environmentReady) return 'ERROR'
  if (!health) return 'READY_TO_START'
  if (health.state === 'RUNNING' && health.healthy) return 'RUNNING'
  if (health.state === 'DEGRADED') return 'DEGRADED'
  if (!health.auth.ready) return 'WAITING_AUTH'
  if (health.state === 'FAILED') return 'ERROR'
  return 'READY_TO_START'
}

export function translateMediaDriveCode(code: string | undefined): UserDiagnostic {
  if (!code) return defaultDiagnostic
  return safeDiagnostics[code] || defaultDiagnostic
}

export function normalizeDriveLetter(value: string): string | null {
  const normalized = value.trim().toUpperCase()
  if (!/^[A-Z]:$/.test(normalized)) return null
  return normalized
}
