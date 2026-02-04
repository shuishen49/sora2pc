import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getEmailFromJwt, getExpFromJwt } from '../utils/jwtUtils'
import {
  fetchTokens,
  addToken,
  updateToken,
  deleteToken,
  checkToken,
  enableToken,
  disableToken,
  fetchSettings,
  updateSettings,
  fetchLogs,
  fetchStats,
  batchTestUpdateTokens,
  batchDeleteTokens,
  batchEnableTokens,
  batchDisableTokens,
  batchUpdateProxy,
  importTokens,
  convertST2AT,
  convertRT2AT,
  fetchATAutoRefreshConfig,
  toggleATAutoRefresh,
  clearLogs,
  updateAdminPassword,
  updateAPIKey,
  fetchProxyConfig,
  updateProxyConfig,
  fetchWatermarkConfig,
  updateWatermarkConfig,
  fetchCacheConfig,
  updateCacheEnabled,
  updateCacheTimeout,
  updateCacheBaseUrl,
  fetchGenerationTimeout,
  updateGenerationTimeout,
  downloadLogs,
  cancelTask
} from '../api/admin'

export const useAdminStore = defineStore('admin', () => {
  // State
  const tokens = ref([])
  const totalTokens = ref(0)
  const currentPage = ref(1)
  const loadingTokens = ref(false)
  const pageSize = ref(20)

  const settings = ref({})
  const loadingSettings = ref(false)

  const logs = ref([])
  const loadingLogs = ref(false)

  const stats = ref({
    total: 0,
    active: 0,
    images: 0,
    videos: 0,
    errors: 0,
    todayImages: 0,
    totalImages: 0,
    todayVideos: 0,
    totalVideos: 0,
    todayErrors: 0,
    totalErrors: 0
  })

  // Helpers：从后端映射；邮箱/过期时间缺省时从 JWT 解析；剩余次数/恢复秒数来自 account/status
  const mapTokenFromBackend = (t) => {
    const tokenStr = t.token || t.accessToken || ''
    const emailFromJwt = getEmailFromJwt(tokenStr)
    const expFromJwt = getExpFromJwt(tokenStr)
    return {
      id: t.id,
      email: t.email || emailFromJwt || null,
      accessToken: tokenStr || t.token,
      sessionToken: t.st,
      refreshToken: t.rt,
      clientId: t.client_id,
      expireTime: t.expiry_time ?? expFromJwt ?? null,
      valid: t.is_active && !t.is_expired,
      isActive: t.is_active,
      isExpired: t.is_expired,
      planType: t.plan_type,
      isPlus: ['chatgpt_team', 'chatgpt_plus', 'chatgpt_pro'].includes(t.plan_type),
      usage: t.sora2_remaining_count ?? t.estimated_num_videos_remaining ?? 0,
      limit: t.sora2_total_count ?? null,
      accessResetsInSeconds: t.access_resets_in_seconds ?? null,
      imageCount: t.image_count,
      videoCount: t.video_count,
      errorCount: t.error_count,
      remark: t.remark,
      proxyUrl: t.proxy_url,
      imageEnabled: t.image_enabled,
      videoEnabled: t.video_enabled,
      imageConcurrency: t.image_concurrency,
      videoConcurrency: t.video_concurrency,
    }
  }

  const mapTokenToBackend = (t) => ({
    token: t.accessToken,
    st: t.sessionToken || null,
    rt: t.refreshToken || null,
    client_id: t.clientId || null,
    proxy_url: t.proxyUrl || '',
    remark: t.remark || null,
    image_enabled: t.imageEnabled,
    video_enabled: t.videoEnabled,
    image_concurrency: t.imageConcurrency,
    video_concurrency: t.videoConcurrency,
  })

  // Actions - Tokens（apiRequest 返回 { data, status }，列表在 data.result / data.total）
  const loadTokens = async (page = 1) => {
    loadingTokens.value = true
    try {
      const response = await fetchTokens(page, pageSize.value)
      const payload = response?.data != null ? response.data : response
      if (Array.isArray(payload)) {
          const start = (page - 1) * pageSize.value
          const end = start + pageSize.value
          tokens.value = payload.slice(start, end).map(mapTokenFromBackend)
          totalTokens.value = payload.length
      } else if (payload && Array.isArray(payload.result)) {
          tokens.value = payload.result.map(mapTokenFromBackend)
          totalTokens.value = payload.total ?? 0
      } else {
        tokens.value = []
        totalTokens.value = 0
      }
      currentPage.value = page
    } catch (e) {
      console.error('Failed to load tokens', e)
    } finally {
      loadingTokens.value = false
    }
  }

  const createToken = async (tokenData, statusResponse = null) => {
    const payload = mapTokenToBackend(tokenData)
    if (statusResponse != null && statusResponse !== '') {
      payload.status_response = statusResponse
    }
    await addToken(payload)
    await loadTokens(currentPage.value)
  }

  const editToken = async (id, tokenData) => {
    await updateToken(id, mapTokenToBackend(tokenData))
    await loadTokens(currentPage.value)
  }

  const removeToken = async (id) => {
    await deleteToken(id)
    await loadTokens(currentPage.value)
  }

  const verifyToken = async (id) => {
    await checkToken(id)
    await loadTokens(currentPage.value)
  }

  const toggleToken = async (id, isActive) => {
    if (isActive) {
      await disableToken(id)
    } else {
      await enableToken(id)
    }
    await loadTokens(currentPage.value)
  }

  // Batch
  const batchCheckTokens = async (ids) => {
    await batchTestUpdateTokens(ids)
    await loadTokens(currentPage.value)
  }

  const handleBatchEnable = async (ids) => {
    await batchEnableTokens(ids)
    await loadTokens(currentPage.value)
  }

  const handleBatchDisable = async (ids) => {
    await batchDisableTokens(ids)
    await loadTokens(currentPage.value)
  }

  const handleBatchDelete = async (ids) => {
    await batchDeleteTokens(ids)
    await loadTokens(currentPage.value)
  }

  const handleBatchProxy = async (ids, proxyUrl) => {
      await batchUpdateProxy(ids, proxyUrl)
      await loadTokens(currentPage.value)
  }

  // Import/Convert
  const handleImportTokens = async (data, mode) => {
      return await importTokens(data, mode)
      // We often reload tokens after this in the component
  }

  const convertST = async (st) => {
      return await convertST2AT(st)
  }

  const convertRT = async (rt, clientId) => {
      return await convertRT2AT(rt, clientId)
  }


  // Helpers - Settings
  const mapSettingsFromBackend = (s) => ({
      adminUsername: s.admin_username,
      // password not returned usually
      proxyEnabled: s.proxy_enabled,
      proxyUrl: s.proxy_url,
      errorBanThreshold: s.error_ban_threshold,
      cacheEnabled: s.cache_enabled,
      cacheTimeout: s.cache_timeout,
      cacheBaseUrl: s.cache_base_url,
      imageTimeout: s.image_timeout,
      videoTimeout: s.video_timeout,
      debugEnabled: s.debug_enabled,
      watermarkEnabled: s.watermark_enabled !== false, // Default true if missing? or s.watermark_enabled
      // API Key usually not returned or masked
  })

  const mapSettingsToBackend = (s) => ({
      // We only map fields that are present in the payload
      ...(s.proxyEnabled !== undefined && { proxy_enabled: s.proxyEnabled }),
      ...(s.proxyUrl !== undefined && { proxy_url: s.proxyUrl }),
      ...(s.errorBanThreshold !== undefined && { error_ban_threshold: s.errorBanThreshold }),
      ...(s.cacheEnabled !== undefined && { cache_enabled: s.cacheEnabled }),
      ...(s.cacheTimeout !== undefined && { cache_timeout: s.cacheTimeout }),
      ...(s.cacheBaseUrl !== undefined && { cache_base_url: s.cacheBaseUrl }),
      ...(s.imageTimeout !== undefined && { image_timeout: s.imageTimeout }),
      ...(s.videoTimeout !== undefined && { video_timeout: s.videoTimeout }),
      ...(s.debugEnabled !== undefined && { debug_enabled: s.debugEnabled }),
      ...(s.watermarkEnabled !== undefined && { watermark_enabled: s.watermarkEnabled }),

      // Special cases for password/apikey if passed here, though usually separate endpoints
      ...(s.apiKey && { new_api_key: s.apiKey }),
      ...(s.passwordUpdate && {
          old_password: s.oldPassword,
          new_password: s.newPassword
      })
  })

  // Actions - Settings（apiRequest 返回 { data, status }，需解包 data）
  const loadSettings = async () => {
    loadingSettings.value = true
    try {
      const [
        generalRes,
        proxyRes,
        watermarkRes,
        cacheRes,
        timeoutRes,
        atRefreshRes
      ] = await Promise.all([
        fetchSettings(),
        fetchProxyConfig(),
        fetchWatermarkConfig(),
        fetchCacheConfig(),
        fetchGenerationTimeout(),
        fetchATAutoRefreshConfig()
      ])
      const unwrap = (r) => (r?.data != null ? r.data : r)
      const general = unwrap(generalRes)
      const proxy = unwrap(proxyRes)
      const watermark = unwrap(watermarkRes)
      const cache = unwrap(cacheRes)
      const timeout = unwrap(timeoutRes)
      const atRefresh = unwrap(atRefreshRes)

      settings.value = {
        ...mapSettingsFromBackend(general || {}),
        proxyEnabled: proxy?.proxy_enabled ?? false,
        proxyUrl: proxy?.proxy_url ?? '',
        watermarkEnabled: watermark?.watermark_free_enabled ?? false,
        watermarkParseMethod: watermark?.parse_method ?? 'third_party',
        watermarkCustomUrl: watermark?.custom_parse_url ?? '',
        watermarkCustomToken: watermark?.custom_parse_token ?? '',
        cacheEnabled: cache?.config?.enabled ?? true,
        cacheTimeout: cache?.config?.timeout ?? 7200,
        cacheBaseUrl: cache?.config?.base_url ?? '',
        cacheEffectiveUrl: cache?.config?.effective_base_url ?? '',
        imageTimeout: timeout?.config?.image_timeout ?? 300,
        videoTimeout: timeout?.config?.video_timeout ?? 1500,
        atAutoRefreshEnabled: atRefresh?.config?.at_auto_refresh_enabled ?? false
      }

      // Update local ref as well
      atAutoRefreshEnabled.value = settings.value.atAutoRefreshEnabled

    } catch (e) {
      console.error('Failed to load settings', e)
    } finally {
      loadingSettings.value = false
    }
  }

  const saveSettings = async (newSettings) => {
      // General config (ban threshold, debug)
      await updateSettings(mapSettingsToBackend(newSettings))
      await loadSettings()
  }

  const saveProxyConfig = async (enabled, url) => {
      await updateProxyConfig(enabled, url)
      await loadSettings()
  }

  const saveWatermarkConfig = async (data) => {
      await updateWatermarkConfig(data)
      await loadSettings()
  }

  const saveCacheConfig = async (enabled, timeout, baseUrl) => {
      await updateCacheEnabled(enabled)
      // Only update timeout/url if they are valid or we want to update them
      if (timeout !== undefined) await updateCacheTimeout(timeout)
      if (baseUrl !== undefined) await updateCacheBaseUrl(baseUrl)
      await loadSettings()
  }

  const saveGenerationTimeout = async (image, video) => {
      await updateGenerationTimeout(image, video)
      await loadSettings()
  }

  const downloadLogFile = async () => {
      return await downloadLogs()
  }

  const cancelTaskAction = async (taskId) => {
      await cancelTask(taskId)
      await loadLogs(currentPage.value) // Reload logs to update status
  }



  const updatePassword = async (username, oldPwd, newPwd) => {
      await updateAdminPassword(username, oldPwd, newPwd)
  }

  const updateApiKey = async (newKey) => {
      await updateAPIKey(newKey)
      await loadSettings()
  }

  // Actions - Logs（apiRequest 返回 { data, status }）
  const loadLogs = async (page = 1) => {
    loadingLogs.value = true
    try {
      const result = await fetchLogs(page)
      const payload = result?.data != null ? result.data : result
      if (Array.isArray(payload)) {
         logs.value = payload
      } else {
         logs.value = payload?.logs || []
      }
    } catch (e) {
      console.error('Failed to load logs', e)
    } finally {
      loadingLogs.value = false
    }
  }

  const clearAllLogs = async () => {
      await clearLogs()
      await loadLogs()
  }

  // Actions - Stats（apiRequest 返回 { data, status }）
  const loadStats = async () => {
    try {
      const res = await fetchStats()
      const data = res?.data != null ? res.data : res
      if (data) {
          stats.value = {
              total: data.total_tokens || 0,
              active: data.active_tokens || 0,
              todayImages: data.today_images || 0,
              totalImages: data.total_images || 0,
              todayVideos: data.today_videos || 0,
              totalVideos: data.total_videos || 0,
              todayErrors: data.today_errors || 0,
              totalErrors: data.total_errors || 0
          }
      }
    } catch (e) {
      console.error('Failed to load stats', e)
    }
  }

  // Actions - Auto Refresh
  const atAutoRefreshEnabled = ref(false)

  const loadATAutoRefreshConfig = async () => {
    try {
      const response = await fetchATAutoRefreshConfig()
      const payload = response?.data != null ? response.data : response
      if (payload && payload.success && payload.config) {
        atAutoRefreshEnabled.value = payload.config.at_auto_refresh_enabled
      }
    } catch (e) {
      console.error('Failed to load AT auto refresh config', e)
    }
  }

  const setATAutoRefresh = async (enabled) => {
    try {
      const response = await toggleATAutoRefresh(enabled)
      const payload = response?.data != null ? response.data : response
      if (payload && payload.success) {
        atAutoRefreshEnabled.value = enabled
        return true
      }
      return false
    } catch (e) {
      console.error('Failed to toggle AT auto refresh', e)
      return false
    }
  }

  return {
    tokens,
    totalTokens,
    currentPage,
    loadingTokens,
    pageSize,

    loadTokens,
    createToken,
    editToken,
    removeToken,
    verifyToken,
    toggleToken,

    batchCheckTokens,
    batchEnableTokens: handleBatchEnable,
    batchDisableTokens: handleBatchDisable,
    batchDeleteTokens: handleBatchDelete,
    batchUpdateProxy: handleBatchProxy,

    importTokens: handleImportTokens,
    convertST2AT: convertST,
    convertRT2AT: convertRT,

    settings,
    loadingSettings,
    loadSettings,
    saveSettings,
    saveProxyConfig,
    saveWatermarkConfig,
    saveCacheConfig,
    saveGenerationTimeout,
    updatePassword,
    updateApiKey,
    downloadLogs: downloadLogFile,
    cancelTask: cancelTaskAction,

    logs,
    loadingLogs,
    loadLogs,
    clearAllLogs,

    stats,
    loadStats,

    atAutoRefreshEnabled,
    loadATAutoRefreshConfig,
    setATAutoRefresh
  }
})
