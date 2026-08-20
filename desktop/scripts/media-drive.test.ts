import assert from 'node:assert/strict'
import test from 'node:test'

import { MediaDriveClient } from '../src/api/mediaDrive.ts'
import { mapHealthToDesktopState, translateMediaDriveCode } from '../src/utils/mediaDrive.ts'

const readyHealth = {
  state: 'RUNNING',
  healthy: true,
  auth: { state: 'READY', ready: true },
  webdav: { state: 'RUNNING', ready: true },
  mount: { state: 'MOUNTED', ready: true },
}

function response(data: unknown, status = 200): Response {
  return new Response(JSON.stringify({ code: status === 200 ? 200 : status, data }), {
    status,
    headers: { 'content-type': 'application/json' },
  })
}

test('TestDesktopStateMapping', () => {
  assert.equal(mapHealthToDesktopState(undefined, true), 'READY_TO_START')
  assert.equal(mapHealthToDesktopState(readyHealth, true), 'RUNNING')
  assert.equal(mapHealthToDesktopState({ ...readyHealth, state: 'DEGRADED', healthy: false }, true), 'DEGRADED')
  assert.equal(
    mapHealthToDesktopState({ ...readyHealth, auth: { state: 'NOT_READY', ready: false }, healthy: false }, true),
    'WAITING_AUTH',
  )
})

test('TestStartButtonWorkflow', async () => {
  const calls: Request[] = []
  const originalFetch = globalThis.fetch
  globalThis.fetch = async (input, init) => {
    const request = new Request(input, init)
    calls.push(request)
    if (request.url.endsWith('/api/auth/login')) return response({ token: 'admin-token' })
    if (request.url.endsWith('/api/admin/media-drive/start')) return response({ state: 'STARTING', running: false })
    return response(readyHealth)
  }
  try {
    const client = new MediaDriveClient('http://127.0.0.1:5244', async () => 'admin-password')
    await client.start('webdav-password')
    assert.deepEqual(
      calls.map(request => new URL(request.url).pathname),
      ['/api/auth/login', '/api/admin/media-drive/start'],
    )
    assert.equal(await calls[1].clone().text(), JSON.stringify({ webdav_password: 'webdav-password' }))
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('TestStopButtonWorkflow', async () => {
  const calls: Request[] = []
  const originalFetch = globalThis.fetch
  globalThis.fetch = async (input, init) => {
    const request = new Request(input, init)
    calls.push(request)
    if (request.url.endsWith('/api/auth/login')) return response({ token: 'admin-token' })
    return response({ state: 'READY', running: false })
  }
  try {
    const client = new MediaDriveClient('http://127.0.0.1:5244', async () => 'admin-password')
    await client.stop()
    assert.equal(new URL(calls[1].url).pathname, '/api/admin/media-drive/stop')
    assert.equal(calls[1].method, 'POST')
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('Test115AuthorizationWorkflow', async () => {
  const calls: Request[] = []
  const originalFetch = globalThis.fetch
  globalThis.fetch = async (input, init) => {
    const request = new Request(input, init)
    calls.push(request)
    if (request.url.endsWith('/api/auth/login')) return response({ token: 'admin-token' })
    if (request.url.endsWith('/api/admin/media-drive/115/auth/capabilities')) {
      return response({ pkce_available: true, token_import_available: true, client_configured: true })
    }
    if (request.url.endsWith('/api/admin/media-drive/115/auth/start')) {
      return response({
        session_id: 'session-id',
        state: 'WAITING',
        qr_code: 'https://115.example/qr',
        expires_at: '2030-01-01T00:00:00Z',
      })
    }
    return response({ storage_id: 1, mount_path: '/115', connected: true, state: 'READY' })
  }
  try {
    const client = new MediaDriveClient('http://127.0.0.1:5244', async () => 'admin-password')
    const capabilities = await client.authCapabilities()
    const session = await client.start115Auth()
    const storage = await client.complete115Auth(session.session_id)
    assert.equal(capabilities.client_configured, true)
    assert.equal(session.state, 'WAITING')
    assert.equal(storage.mount_path, '/115')
    assert.deepEqual(
      calls.map(request => new URL(request.url).pathname),
      [
        '/api/auth/login',
        '/api/admin/media-drive/115/auth/capabilities',
        '/api/admin/media-drive/115/auth/start',
        '/api/admin/media-drive/115/auth/complete',
      ],
    )
    assert.equal(await calls[3].clone().text(), JSON.stringify({ session_id: 'session-id' }))
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('TestErrorTranslation', () => {
  const diagnostic = translateMediaDriveCode('WINFSP_UNAVAILABLE')
  assert.match(diagnostic.message, /WinFsp/)
  assert.match(diagnostic.suggestion, /安装|install/i)
})

test('TestNoSecretDisplay', () => {
  const diagnostic = translateMediaDriveCode('TOKEN_PERSISTENCE_FAILED')
  const rendered = JSON.stringify(diagnostic)
  assert.equal(rendered.includes('access_token'), false)
  assert.equal(rendered.includes('refresh_token'), false)
  assert.equal(rendered.includes('admin-password'), false)
})
