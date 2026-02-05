<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useAdminStore } from '../../stores/admin'
import { useGenerateStore } from '../../stores/generate'

const adminStore = useAdminStore()
const generateStore = useGenerateStore()
const saving = ref(false)
const serverTesting = ref(false)
const checkingUpdate = ref(false)
const updateInfo = ref(null)
const downloading = ref(false)
const downloadProgress = ref(0)
const currentVersion = ref('')

const form = reactive({
  adminUsername: '',
  oldPassword: '',
  newPassword: '',

  currentApiKey: '****************', // Read-only placeholder
  newApiKey: '',

  proxyEnabled: false,
  proxyUrl: '',

  watermarkEnabled: true,
  watermarkParseMethod: 'third_party',
  watermarkCustomUrl: '',
  watermarkCustomToken: '',

  errorBanThreshold: 3,

  cacheEnabled: true,
  cacheTimeout: 7200,
  cacheBaseUrl: '',
  cacheEffectiveUrl: '', // Read-only display

  imageTimeout: 300,
  videoTimeout: 1500,

  debugEnabled: false,
  atAutoRefreshEnabled: false,

  // 本地桌面客户端的 Sora 服务器地址（只影响本客户端）
  serverBaseUrl: ''
})

onMounted(async () => {
  await adminStore.loadSettings()
  // Mapping API settings to form
  const s = adminStore.settings
  if (s) {
    form.adminUsername = s.adminUsername || 'admin'

    // Proxy
    form.proxyEnabled = s.proxyEnabled || false
    form.proxyUrl = s.proxyUrl || ''

    // Watermark
    form.watermarkEnabled = s.watermarkEnabled !== false
    form.watermarkParseMethod = s.watermarkParseMethod || 'third_party'
    form.watermarkCustomUrl = s.watermarkCustomUrl || ''
    form.watermarkCustomToken = s.watermarkCustomToken || ''

    // General
    form.errorBanThreshold = s.errorBanThreshold || 3
    form.debugEnabled = s.debugEnabled || false
    form.atAutoRefreshEnabled = s.atAutoRefreshEnabled || false

    // Cache
    form.cacheEnabled = s.cacheEnabled !== false
    form.cacheTimeout = s.cacheTimeout || 7200
    form.cacheBaseUrl = s.cacheBaseUrl || ''
    form.cacheEffectiveUrl = s.cacheEffectiveUrl || ''

    // Timeouts
    form.imageTimeout = s.imageTimeout || 300
    form.videoTimeout = s.videoTimeout || 1500
  }

  // 优先从后端（SQLite settings.base_url）读取服务器地址，保证与实际配置一致
  if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetBaseURL) {
    try {
      const url = await window.go.main.App.GetBaseURL()
      if (url) {
        // 同步到全局 store 和当前表单
        generateStore.setBaseUrl(url)
        form.serverBaseUrl = url
        return
      }
    } catch (e) {
      console.error('GetBaseURL in SettingsPanel failed:', e)
    }
  }

  // 回退：使用当前 store 中的 baseUrl（可能是默认值）
  form.serverBaseUrl = generateStore.baseUrl || ''
  
  // 获取当前版本
  if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetCurrentVersion) {
    try {
      currentVersion.value = await window.go.main.App.GetCurrentVersion()
    } catch (e) {
      console.error('获取版本失败:', e)
      currentVersion.value = '未知'
    }
  }
})

const handlePasswordChange = async () => {
  if (!form.newPassword) return alert('请输入新密码')
  try {
    await adminStore.updatePassword(form.adminUsername, form.oldPassword, form.newPassword)
    alert('密码修改成功，请重新登录')
    form.oldPassword = ''
    form.newPassword = ''
  } catch (e) {
    alert('修改失败: ' + (e.message || '未知错误'))
  }
}

const handleUpdateKey = async () => {
    if (!form.newApiKey) return alert('请输入新 Key')
    if (!confirm('确定更新 API Key 吗？更新后旧 Key 将失效。')) return
    try {
        await adminStore.updateApiKey(form.newApiKey)
        alert('API Key 更新成功')
        form.newApiKey = ''
    } catch (e) {
        alert('更新失败: ' + e.message)
    }
}

const wrapSave = async (fn, successMsg = '配置已保存') => {
    if (saving.value) return
    saving.value = true
    try {
        await fn()
        alert(successMsg)
    } catch (e) {
        alert('保存失败: ' + e.message)
    } finally {
        saving.value = false
    }
}

const handleSaveGeneral = () => wrapSave(async () => {
    await adminStore.saveSettings({
        errorBanThreshold: form.errorBanThreshold,
        debugEnabled: form.debugEnabled
    })
    // Also save AT auto refresh separately if needed, or let user toggle it directly
    await adminStore.setATAutoRefresh(form.atAutoRefreshEnabled)
})

const handleSaveProxy = () => wrapSave(async () => {
    await adminStore.saveProxyConfig(form.proxyEnabled, form.proxyUrl)
})

const handleSaveWatermark = () => wrapSave(async () => {
    await adminStore.saveWatermarkConfig({
        watermark_free_enabled: form.watermarkEnabled,
        parse_method: form.watermarkParseMethod,
        custom_parse_url: form.watermarkCustomUrl,
        custom_parse_token: form.watermarkCustomToken
    })
})

const handleSaveCache = () => wrapSave(async () => {
    await adminStore.saveCacheConfig(form.cacheEnabled, form.cacheTimeout, form.cacheBaseUrl)
})

const handleSaveTimeouts = () => wrapSave(async () => {
    await adminStore.saveGenerationTimeout(form.imageTimeout, form.videoTimeout)
})

// 保存本地服务器地址（写入 SQLite settings 等）
const handleSaveServer = () => {
  if (!form.serverBaseUrl || !form.serverBaseUrl.trim()) {
    alert('请输入服务器地址，例如 http://127.0.0.1:8000')
    return
  }
  // 直接使用 generateStore 的统一入口
  generateStore.setBaseUrl(form.serverBaseUrl.trim())
  alert('服务器地址已保存，仅对当前客户端生效')
}

// 测试当前服务器是否可用（通过 Go 后端请求 /health，避免前端跨域）
const handleTestServer = async () => {
  if (!form.serverBaseUrl || !form.serverBaseUrl.trim()) {
    alert('请先输入服务器地址')
    return
  }

  serverTesting.value = true
  try {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.TestServerHealth) {
      const result = await window.go.main.App.TestServerHealth(form.serverBaseUrl.trim())
      if (result && result.ok) {
        alert(result.message || '服务器连接正常，可以使用')
      } else {
        alert(result?.message || '服务器连接失败或返回异常，请检查配置')
      }
    } else {
      alert('当前环境不支持本地测试（缺少 TestServerHealth），请在桌面客户端中使用。')
    }
  } catch (e) {
    alert('测试服务器失败: ' + (e.message || e))
  } finally {
    serverTesting.value = false
  }
}

const wmDropdownOpen = ref(false)
const selectWmMethod = (method) => {
    form.watermarkParseMethod = method
    wmDropdownOpen.value = false
}

// Reuse downloading helper if needed or keep existing download log logic
const handleDownloadLogs = async () => {
    try {
        const blob = await adminStore.downloadLogs()
        const url = URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `logs_${new Date().toISOString().split('T')[0]}.txt`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        URL.revokeObjectURL(url)
    } catch (e) {
        alert('下载失败: ' + e.message)
    }
}

// 检查更新
const handleCheckUpdate = async () => {
    if (checkingUpdate.value) return
    
    checkingUpdate.value = true
    updateInfo.value = null
    
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CheckForUpdates) {
            const resultStr = await window.go.main.App.CheckForUpdates()
            const result = JSON.parse(resultStr)
            updateInfo.value = result
            
            if (result.has_update) {
                // 有更新，显示提示
            } else if (result.error) {
                alert('检查更新失败: ' + result.error)
            } else {
                alert('当前已是最新版本')
            }
        } else {
            alert('当前环境不支持更新检查（请在桌面客户端中使用）')
        }
    } catch (e) {
        alert('检查更新失败: ' + (e.message || e))
        updateInfo.value = null
    } finally {
        checkingUpdate.value = false
    }
}

// 下载并安装更新
const handleDownloadAndInstall = async () => {
    if (!updateInfo.value || !updateInfo.value.has_update || !updateInfo.value.download_url) {
        alert('没有可用的更新')
        return
    }
    
    if (!confirm(`确定要下载并安装版本 ${updateInfo.value.latest_version} 吗？\n\n应用将在安装完成后自动关闭。`)) {
        return
    }
    
    downloading.value = true
    downloadProgress.value = 0
    
    try {
        // 先下载
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DownloadUpdate) {
            const downloadResultStr = await window.go.main.App.DownloadUpdate(updateInfo.value.download_url)
            const downloadResult = JSON.parse(downloadResultStr)
            
            if (!downloadResult.success) {
                throw new Error(downloadResult.message || '下载失败')
            }
            
            downloadProgress.value = 100
            
            // 下载完成后安装
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.InstallUpdate) {
                const installResultStr = await window.go.main.App.InstallUpdate(downloadResult.local_path)
                const installResult = JSON.parse(installResultStr)
                
                if (!installResult.success) {
                    throw new Error(installResult.message || '安装失败')
                }
                
                alert(installResult.message || '安装程序已启动，应用即将关闭')
            } else {
                alert('下载完成，但无法自动安装。请手动运行: ' + downloadResult.local_path)
            }
        } else {
            alert('当前环境不支持自动更新（请在桌面客户端中使用）')
        }
    } catch (e) {
        alert('更新失败: ' + (e.message || e))
    } finally {
        downloading.value = false
        downloadProgress.value = 0
    }
}

</script>

<template>
  <div class="settings-panel">
    <div class="grid-layout">
      <!-- General -->
      <div class="card">
        <h3>通用配置</h3>
        <div class="field">
            <label>错误封禁阈值 (Error Count)</label>
            <input v-model.number="form.errorBanThreshold" type="number" />
        </div>

        <div class="section-divider"></div>

        <div class="checkbox-row">
            <input type="checkbox" id="atRefresh" v-model="form.atAutoRefreshEnabled" />
            <label for="atRefresh">启用 AT 自动刷新 (每日0点)</label>
        </div>

        <div class="checkbox-row">
            <input type="checkbox" id="debug" v-model="form.debugEnabled" />
            <label for="debug">启用调试模式</label>
        </div>

        <div class="action-row">
             <button class="btn-secondary" @click="handleDownloadLogs">下载日志</button>
             <button class="btn-primary" @click="handleSaveGeneral">保存通用配置</button>
        </div>
      </div>

      <!-- Local Server Config -->
      <div class="card">
        <h3>服务器 IP 设置（本客户端）</h3>
        <div class="field">
          <label>服务器地址</label>
          <input
            v-model="form.serverBaseUrl"
            placeholder="例如：http://127.0.0.1:8000"
          />
        </div>
        <p class="hint">
          仅影响当前桌面客户端访问的 Sora API 地址，不会改动远端服务端配置。
        </p>
        <div class="action-row">
          <button class="btn-secondary" :disabled="serverTesting" @click="handleTestServer">
            {{ serverTesting ? '测试中...' : '测试服务器' }}
          </button>
          <button class="btn-primary" @click="handleSaveServer">保存服务器地址</button>
        </div>
      </div>

      <!-- Proxy -->
      <div class="card">
        <h3>代理配置</h3>
        <div class="checkbox-row">
          <input type="checkbox" id="proxy" v-model="form.proxyEnabled" />
          <label for="proxy">启用代理</label>
        </div>
        <div class="field">
          <label>代理地址</label>
          <input v-model="form.proxyUrl" placeholder="http://127.0.0.1:7890" :disabled="!form.proxyEnabled" />
        </div>
        <button class="btn-primary" @click="handleSaveProxy">保存代理配置</button>
      </div>

      <!-- Watermark -->
      <div class="card">
        <h3>无水印配置</h3>
        <div class="checkbox-row">
            <input type="checkbox" id="watermark" v-model="form.watermarkEnabled" />
            <label for="watermark">启用无水印模式</label>
        </div>

        <div v-if="form.watermarkEnabled" class="sub-fields">
            <div class="field">
                <label>解析方式</label>
                <div class="hover-dropdown" @mouseenter="wmDropdownOpen = true" @mouseleave="wmDropdownOpen = false">
                    <div class="dropdown-trigger">
                        {{ form.watermarkParseMethod === 'third_party' ? '第三方解析 (API)' : '自建解析服务' }}
                        <span class="arrow">▼</span>
                    </div>
                    <div class="dropdown-menu" :class="{ show: wmDropdownOpen }">
                        <div class="dropdown-item" :class="{ active: form.watermarkParseMethod === 'third_party' }" @click="selectWmMethod('third_party')">第三方解析 (API)</div>
                        <div class="dropdown-item" :class="{ active: form.watermarkParseMethod === 'custom' }" @click="selectWmMethod('custom')">自建解析服务</div>
                    </div>
                </div>
            </div>

            <div v-if="form.watermarkParseMethod === 'custom'" class="sub-fields-inner">
                 <div class="field">
                    <label>解析服务地址</label>
                    <input v-model="form.watermarkCustomUrl" placeholder="https://..." />
                </div>
                <div class="field">
                    <label>访问令牌 (Token)</label>
                    <input v-model="form.watermarkCustomToken" placeholder="可选" />
                </div>
            </div>
        </div>

        <button class="btn-primary" @click="handleSaveWatermark">保存无水印配置</button>
      </div>

      <!-- Timeouts -->
      <div class="card">
        <h3>生成超时配置</h3>
        <div class="field">
          <label>图片超时 (秒)</label>
          <input v-model.number="form.imageTimeout" type="number" />
        </div>
        <div class="field">
          <label>视频超时 (秒)</label>
          <input v-model.number="form.videoTimeout" type="number" />
        </div>
        <button class="btn-primary" @click="handleSaveTimeouts">保存超时配置</button>
      </div>

      <!-- Update Check -->
      <div class="card">
        <h3>应用更新</h3>
        <div class="field">
          <label>当前版本</label>
          <div class="version-display">{{ currentVersion || '检查中...' }}</div>
        </div>
        
        <div v-if="updateInfo && updateInfo.has_update" class="update-available">
          <div class="update-info">
            <p class="update-title">🎉 发现新版本 {{ updateInfo.latest_version }}</p>
            <p class="update-current">当前版本: {{ updateInfo.current_version }}</p>
            <div v-if="updateInfo.release_notes" class="release-notes">
              <strong>更新内容:</strong>
              <pre>{{ updateInfo.release_notes }}</pre>
            </div>
          </div>
          
          <div v-if="downloading" class="download-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
            </div>
            <p>下载中... {{ downloadProgress }}%</p>
          </div>
          
          <div class="action-row">
            <button 
              class="btn-update" 
              :disabled="downloading" 
              @click="handleDownloadAndInstall"
            >
              {{ downloading ? '下载中...' : '下载并安装更新' }}
            </button>
          </div>
        </div>
        
        <div v-else-if="updateInfo && !updateInfo.has_update && !updateInfo.error" class="update-status">
          <p class="status-text">✓ 已是最新版本</p>
        </div>
        
        <div class="action-row">
          <button 
            class="btn-secondary" 
            :disabled="checkingUpdate || downloading" 
            @click="handleCheckUpdate"
          >
            {{ checkingUpdate ? '检查中...' : '检查更新' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel {
  padding-bottom: 40px;
}

.grid-layout {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.card {
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

h3 {
  margin: 0 0 4px;
  font-size: 16px;
  color: #f1f5f9;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field label { font-size: 12px; color: #94a3b8; font-weight: 500; }
.field input,
.field select {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 10px;
  color: #f1f5f9;
  outline: none;
  font-size: 14px;
}
.field select {
  cursor: pointer;
}
.field select option {
  background: #0f172a;
  color: #f1f5f9;
}
.field input:disabled, .disabled-input {
    opacity: 0.6;
    cursor: not-allowed;
    background: #1e293b;
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #e2e8f0;
}

.sub-fields {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-left: 10px;
    border-left: 2px solid #334155;
}

.btn-primary {
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  color: white;
  border: none;
  padding: 9px 16px;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
  margin-top: auto; /* Push to bottom */
}
.btn-primary:active { transform: scale(0.98); }


.action-row {
    display: flex;
    gap: 10px;
    margin-top: auto;
}

.sub-fields-inner {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding-left: 10px;
    border-left: 2px solid #475569;
    margin-top: 8px;
}

.hint-text {
    font-size: 11px;
    color: #64748b;
    margin-top: 2px;
}

.btn-secondary {
    background: #334155;
    color: #f1f5f9;
    border: none;
    padding: 9px 16px;
    border-radius: 8px;
    font-weight: 500;
    cursor: pointer;
}
.btn-secondary:hover { background: #475569; }

.hint {
    font-size: 12px;
    color: #eab308;
    margin: 0;
}

/* Hover Dropdown */
.hover-dropdown {
    position: relative;
    width: 100%;
}

.dropdown-trigger {
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
    padding: 10px;
    color: #f1f5f9;
    font-size: 14px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: border-color 0.15s;
}

.dropdown-trigger:hover {
    border-color: #3b82f6;
}

.dropdown-trigger .arrow {
    font-size: 10px;
    color: #94a3b8;
    transition: transform 0.2s;
}

.hover-dropdown:hover .dropdown-trigger .arrow {
    transform: rotate(180deg);
}

.dropdown-menu {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
    margin-top: 4px;
    z-index: 100;
    opacity: 0;
    visibility: hidden;
    transform: translateY(-8px);
    transition: all 0.15s ease;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.hover-dropdown .dropdown-menu.show {
    opacity: 1;
    visibility: visible;
    transform: translateY(0);
}

.dropdown-item {
    padding: 10px 12px;
    color: #e2e8f0;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.15s;
}

.dropdown-item:first-child {
    border-radius: 8px 8px 0 0;
}

.dropdown-item:last-child {
    border-radius: 0 0 8px 8px;
}

.dropdown-item:hover {
    background: rgba(59, 130, 246, 0.2);
}

.dropdown-item.active {
    background: rgba(59, 130, 246, 0.3);
    color: #60a5fa;
}

/* Update Section */
.version-display {
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
    padding: 10px;
    color: #38bdf8;
    font-weight: 600;
    font-size: 14px;
}

.update-available {
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.3);
    border-radius: 8px;
    padding: 16px;
    margin-top: 12px;
}

.update-info {
    margin-bottom: 16px;
}

.update-title {
    font-size: 16px;
    font-weight: 600;
    color: #22c55e;
    margin: 0 0 8px 0;
}

.update-current {
    font-size: 12px;
    color: #94a3b8;
    margin: 0 0 12px 0;
}

.release-notes {
    margin-top: 12px;
    font-size: 12px;
    color: #cbd5e1;
}

.release-notes strong {
    color: #e2e8f0;
    display: block;
    margin-bottom: 6px;
}

.release-notes pre {
    background: rgba(15, 23, 42, 0.6);
    border: 1px solid #334155;
    border-radius: 6px;
    padding: 10px;
    margin: 8px 0 0 0;
    white-space: pre-wrap;
    word-wrap: break-word;
    max-height: 200px;
    overflow-y: auto;
    font-size: 11px;
    line-height: 1.5;
}

.download-progress {
    margin: 16px 0;
}

.progress-bar {
    width: 100%;
    height: 8px;
    background: #1e293b;
    border-radius: 4px;
    overflow: hidden;
    margin-bottom: 8px;
}

.progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #22c55e);
    transition: width 0.3s ease;
}

.update-status {
    background: rgba(59, 130, 246, 0.1);
    border: 1px solid rgba(59, 130, 246, 0.3);
    border-radius: 8px;
    padding: 12px;
    margin-top: 12px;
    text-align: center;
}

.status-text {
    color: #60a5fa;
    font-size: 14px;
    margin: 0;
}

.btn-update {
    background: linear-gradient(135deg, #22c55e, #16a34a);
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    width: 100%;
    transition: transform 0.2s;
}

.btn-update:hover:not(:disabled) {
    transform: scale(1.02);
}

.btn-update:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}
</style>
