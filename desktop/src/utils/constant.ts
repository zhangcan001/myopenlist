export const DEFAULT_CONFIG = {
  openlistCore: {
    port: 5244,
    data_dir: '',
    auto_launch: false,
    ssl_enabled: false,
    binary_path: '',
  },
  rclone: {
    mount_config: {},
    binary_path: '',
    rclone_conf_path: '',
  },
  app: {
    theme: 'light',
    auto_update_enabled: true,
    gh_proxy: '',
    gh_proxy_api: false,
    open_links_in_browser: false,
    show_window_on_startup: true,
    hide_dock_icon: false,
    admin_password: '',
  },
}
export const isWindows = typeof OS_PLATFORM !== 'undefined' && OS_PLATFORM === 'win32'
export const isLinux = typeof OS_PLATFORM !== 'undefined' && OS_PLATFORM === 'linux'
export const isMacOs = typeof OS_PLATFORM !== 'undefined' && OS_PLATFORM === 'darwin'
