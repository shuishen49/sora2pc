import { apiRequest, postJson } from './client'
import { ApiRequestBlob } from '../../wailsjs/go/main/App'

// Token Management
export const fetchTokens = (page = 1, limit = 20) =>
  apiRequest(`/api/tokens?page=${page}&limit=${limit}`)

export const addToken = (data) => postJson('/api/tokens', data)

export const updateToken = (id, data) => postJson(`/api/tokens/${id}`, data)

export const deleteToken = (id) => apiRequest(`/api/tokens/${id}`, { method: 'DELETE' })

// Status & Testing
export const checkToken = (id) => postJson(`/api/tokens/${id}/test`)
export const enableToken = (id) => postJson(`/api/tokens/${id}/enable`)
export const disableToken = (id) => postJson(`/api/tokens/${id}/disable`)
export const updateTokenStatus = (id, isActive) => postJson(`/api/tokens/${id}/status`, { is_active: isActive }, 'PUT')

// Conversion
export const convertST2AT = (st) => postJson('/api/tokens/st2at', { st })
export const convertRT2AT = (rt, clientId) => postJson('/api/tokens/rt2at', { rt, client_id: clientId })

// Batch Operations
// Backend expects { token_ids: [...] }
export const batchTestUpdateTokens = (tokenIds) => postJson('/api/tokens/batch/test-update', { token_ids: tokenIds })
export const batchDeleteTokens = (tokenIds) => postJson('/api/tokens/batch/delete-disabled', { token_ids: tokenIds })
export const batchEnableTokens = (tokenIds) => postJson('/api/tokens/batch/enable-all', { token_ids: tokenIds })
export const batchDisableTokens = (tokenIds) => postJson('/api/tokens/batch/disable-selected', { token_ids: tokenIds })
export const batchUpdateProxy = (tokenIds, proxyUrl) => postJson('/api/tokens/batch/update-proxy', { token_ids: tokenIds, proxy_url: proxyUrl })

// Import/Export
export const importTokens = (tokens, mode) => postJson('/api/tokens/import', { tokens, mode })

// Auto Refresh
export const fetchATAutoRefreshConfig = () => apiRequest('/api/token-refresh/config')
export const toggleATAutoRefresh = (enabled) => postJson('/api/token-refresh/enabled', { enabled })

// System Settings
export const fetchSettings = () => apiRequest('/api/admin/config') // Backend path is /api/admin/config
export const updateSettings = (data) => postJson('/api/admin/config', data)
export const updateAdminPassword = (username, oldPassword, newPassword) =>
  postJson('/api/admin/password', { username, old_password: oldPassword, new_password: newPassword })
export const updateAPIKey = (newAPIKey) => postJson('/api/admin/apikey', { new_api_key: newAPIKey })
export const updateDebugConfig = (enabled) => postJson('/api/admin/debug', { enabled })

// Logs
export const fetchLogs = (page = 1, limit = 50) =>
  apiRequest(`/api/logs?page=${page}&limit=${limit}`)

export const clearLogs = () => apiRequest('/api/logs', { method: 'DELETE' })

// Proxy
export const fetchProxyConfig = () => apiRequest('/api/proxy/config')
export const updateProxyConfig = (enabled, url) => postJson('/api/proxy/config', { proxy_enabled: enabled, proxy_url: url })

// Watermark-free
export const fetchWatermarkConfig = () => apiRequest('/api/watermark-free/config')
export const updateWatermarkConfig = (data) => postJson('/api/watermark-free/config', data)

// Cache
export const fetchCacheConfig = () => apiRequest('/api/cache/config')
export const updateCacheEnabled = (enabled) => postJson('/api/cache/enabled', { enabled })
export const updateCacheTimeout = (timeout) => postJson('/api/cache/config', { timeout })
export const updateCacheBaseUrl = (baseUrl) => postJson('/api/cache/base-url', { base_url: baseUrl })

// Generation Timeouts
export const fetchGenerationTimeout = () => apiRequest('/api/generation/timeout')
export const updateGenerationTimeout = (imageTimeout, videoTimeout) => postJson('/api/generation/timeout', { image_timeout: imageTimeout, video_timeout: videoTimeout })



export const cancelTask = (taskId) => postJson(`/api/tasks/${taskId}/cancel`)

// Log Download (This returns a blob, so we might need a custom fetch in the store or here)
// However, apiRequest expects JSON usually. Let's export a specialized function if needed,
// but for download we usually just use window.open or a direct fetch in the component/store to get a blob.
// But keeping it consistent:
export const downloadLogs = async () => {
    const token = localStorage.getItem('adminToken')
    const dataUrl = await ApiRequestBlob('GET', '/api/admin/logs/download', token || '')
    // dataUrl is "data:application/octet-stream;base64,..."
    const base64 = dataUrl.split(',')[1]
    const binary = atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
    return new Blob([bytes])
}

// Stats
export const fetchStats = () => apiRequest('/api/stats')
