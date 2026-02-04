import { ApiRequest as WailsApiRequest } from '../../wailsjs/go/main/App'

export async function apiRequest(path, options = {}) {
  const token = localStorage.getItem('adminToken')
  const method = (options.method || 'GET').toUpperCase()
  const body = options.body || ''

  // 通过 Wails 绑定调用 Go 后端代理，避免跨域问题
  let raw
  try {
    raw = await WailsApiRequest(method, path, body, token || '')
  } catch (err) {
    // Wails 错误可能是字符串或对象
    const msg = typeof err === 'string' ? err : (err?.message || JSON.stringify(err))
    console.error('ApiRequest error:', err)
    const error = new Error(msg)
    error.data = err
    throw error
  }

  let data = null
  try {
    data = JSON.parse(raw)
  } catch {
    data = raw
  }

  return { data, status: 200 }
}

export function postJson(path, body) {
  return apiRequest(path, {
    method: 'POST',
    body: JSON.stringify(body ?? {}),
  })
}
