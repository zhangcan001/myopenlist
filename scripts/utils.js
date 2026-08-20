import fs from 'fs-extra'
import { HttpsProxyAgent } from 'https-proxy-agent'
import fetch from 'node-fetch'

export const platformIdentifierMap = {
  'x86_64-pc-windows-msvc': 'win32-x64',
  'i686-pc-windows-msvc': 'win32-ia32',
  'aarch64-pc-windows-msvc': 'win32-arm64',
  'x86_64-apple-darwin': 'darwin-x64',
  'aarch64-apple-darwin': 'darwin-arm64',
  'x86_64-unknown-linux-gnu': 'linux-x64',
  'i686-unknown-linux-gnu': 'linux-ia32',
  'aarch64-unknown-linux-gnu': 'linux-arm64',
  'armv7-unknown-linux-gnueabihf': 'linux-arm',
}

const archMaps = {
  'win32-x64': { openlist: 'windows-amd64', rclone: 'windows-amd64' },
  'win32-ia32': { openlist: 'windows-386', rclone: 'windows-386' },
  'win32-arm64': { openlist: 'windows-arm64', rclone: 'windows-arm64' },
  'darwin-x64': { openlist: 'darwin-amd64', rclone: 'osx-amd64' },
  'darwin-arm64': { openlist: 'darwin-arm64', rclone: 'osx-arm64' },
  'linux-x64': { openlist: 'linux-amd64', rclone: 'linux-amd64' },
  'linux-ia32': { openlist: 'linux-386', rclone: 'linux-386' },
  'linux-arm64': { openlist: 'linux-arm64', rclone: 'linux-arm64' },
  'linux-arm': { openlist: 'linux-arm-7', rclone: 'linux-arm-v7' },
}

export const getOpenlistArchMap = Object.fromEntries(
  Object.entries(archMaps).map(([key, { openlist }]) => [
    key,
    `openlist-${openlist}.${key.startsWith('darwin') || key.startsWith('linux') ? 'tar.gz' : 'zip'}`,
  ]),
)

export const getRcloneArchMap = version =>
  Object.fromEntries(Object.entries(archMaps).map(([key, { rclone }]) => [key, `rclone-${version}-${rclone}.zip`]))

export async function downloadFile(url, path) {
  const response = await fetch(url, {
    ...getFetchOptions(),
    headers: { 'Content-Type': 'application/octet-stream' },
  })
  await fs.writeFile(path, new Uint8Array(await response.arrayBuffer()))
  console.log(`download finished: ${url}`)
}

export const getFetchOptions = () => {
  const proxy = process.env.HTTP_PROXY || process.env.http_proxy || process.env.HTTPS_PROXY || process.env.https_proxy
  return proxy ? { agent: new HttpsProxyAgent(proxy) } : {}
}
