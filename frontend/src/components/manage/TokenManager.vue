<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import { useAdminStore } from '../../stores/admin'
import { storeToRefs } from 'pinia'
import { CheckAccountAndSave } from '../../../wailsjs/go/main/App'

const adminStore = useAdminStore()
const { tokens, loadingTokens } = storeToRefs(adminStore)
const showModal = ref(false)
const modalMode = ref('add') // 'add' or 'edit'
const editId = ref(null)
const submitting = ref(false)

// Import Modal State
const showImportModal = ref(false)
const importFile = ref(null)
const importMode = ref('at')
const importResult = ref(null)
const importing = ref(false)

// Batch Proxy Modal State
const showBatchProxyModal = ref(false)
const batchProxyUrl = ref('')
const batchSubmitting = ref(false)

// Conversion Loading States
const converting = ref(false)
const refreshHintVisible = ref(false)

const form = reactive({
  accessToken: '',
  sessionToken: '',
  refreshToken: '', // Added
  clientId: '',     // Added
  proxyUrl: '',     // Added
  remark: '',
  imageEnabled: true,
  videoEnabled: true,
  imageConcurrency: -1, // Changed default to -1 (Unlimited)
  videoConcurrency: 3,
})

const resetForm = () => {
  form.accessToken = ''
  form.sessionToken = ''
  form.refreshToken = ''
  form.clientId = ''
  form.proxyUrl = ''
  form.remark = ''
  form.imageEnabled = true
  form.videoEnabled = true
  form.imageConcurrency = -1
  form.videoConcurrency = 3
  refreshHintVisible.value = false
}

const openAddModal = () => {
  modalMode.value = 'add'
  editId.value = null
  resetForm()
  showModal.value = true
}

const openEditModal = (token) => {
  modalMode.value = 'edit'
  editId.value = token.id
  form.accessToken = token.accessToken || ''
  form.sessionToken = token.sessionToken || ''
  form.refreshToken = token.refreshToken || ''
  form.clientId = token.clientId || ''
  form.proxyUrl = token.proxyUrl || ''
  form.remark = token.remark || ''
  form.imageEnabled = token.imageEnabled ?? true
  form.videoEnabled = token.videoEnabled ?? true
  form.imageConcurrency = token.imageConcurrency ?? -1
  form.videoConcurrency = token.videoConcurrency ?? 3
  refreshHintVisible.value = false
  showModal.value = true
}

// Convert Helpers
const handleConvertST = async () => {
  if (!form.sessionToken) return alert('请先输入 Session Token')
  converting.value = true
  try {
    const res = await adminStore.convertST2AT(form.sessionToken)
    if (res && res.success && res.access_token) {
        form.accessToken = res.access_token
        alert('转换成功！AT已自动填入')
    } else {
        alert('转换失败: ' + (res?.message || '未知错误'))
    }
  } catch (e) {
    alert('转换出错: ' + e.message)
  } finally {
    converting.value = false
  }
}

const handleConvertRT = async () => {
  if (!form.refreshToken) return alert('请先输入 Refresh Token')
  converting.value = true
  try {
    const res = await adminStore.convertRT2AT(form.refreshToken, form.clientId)
    if (res && res.success && res.access_token) {
        form.accessToken = res.access_token
        if (res.refresh_token) {
            form.refreshToken = res.refresh_token
            refreshHintVisible.value = true
        }
        alert('转换成功！AT已自动填入')
    } else {
        alert('转换失败: ' + (res?.message || '未知错误'))
    }
  } catch (e) {
    alert('转换出错: ' + e.message)
  } finally {
    converting.value = false
  }
}

const handleSubmit = async () => {
  if (submitting.value) return
  submitting.value = true
  try {
    const data = { ...form }
    // Ensure concurrency is int
    data.imageConcurrency = parseInt(data.imageConcurrency) || -1
    data.videoConcurrency = parseInt(data.videoConcurrency) || -1

    // 保存时发 2 个请求（远程服务器为配置的 BaseURL）：1) /account/status 2) /account/me
    let statusResponse = null
    const token = data.accessToken || ''
    if (typeof CheckAccountAndSave === 'function') {
      try {
        statusResponse = await CheckAccountAndSave(token)
      } catch (err) {
        alert('账号状态检查失败: ' + (err.message || err))
        return
      }
    }
    // 不再请求 /account/me，邮箱从 JWT 解析即可

    if (modalMode.value === 'add') {
      await adminStore.createToken(data, statusResponse)
    } else {
      await adminStore.editToken(editId.value, data)
    }
    showModal.value = false
  } catch (e) {
    alert('操作失败: ' + (e.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

const selectedTokens = ref([])

// ========== 默认/示例数据（不存数据库，仅前端展示） ==========
// 当数据库/接口返回的 Token 列表为空时，表格显示下面 3 条假数据；
// 你看到的「demo-user-1@example.com」「示例账号 A」等都来自这里。
const FAKE_DEMO_TOKENS = [
  {
    id: -1,
    email: 'demo-user-1@example.com',
    accessToken: 'eyJhbGciOiJSUzI1NiIsInR5cCI6...',
    sessionToken: null,
    refreshToken: null,
    clientId: 'client-abc12345',
    expireTime: Math.floor(Date.now() / 1000) + 86400 * 30,
    valid: true,
    isActive: true,
    isExpired: false,
    planType: 'chatgpt_plus',
    isPlus: true,
    usage: 28,
    limit: 30,
    imageCount: 12,
    videoCount: 5,
    errorCount: 0,
    remark: '示例账号 A',
    proxyUrl: '',
    imageEnabled: true,
    videoEnabled: true,
    imageConcurrency: -1,
    videoConcurrency: 3,
  },
  {
    id: -2,
    email: 'demo-user-2@example.com',
    accessToken: 'eyJhbGciOiJSUzI1NiIsInR5cCI6...',
    sessionToken: null,
    refreshToken: null,
    clientId: null,
    expireTime: null,
    valid: false,
    isActive: false,
    isExpired: true,
    planType: null,
    isPlus: false,
    usage: 0,
    limit: '∞',
    imageCount: 0,
    videoCount: 0,
    errorCount: 3,
    remark: '已过期示例',
    proxyUrl: 'http://127.0.0.1:7890',
    imageEnabled: true,
    videoEnabled: false,
    imageConcurrency: 2,
    videoConcurrency: 1,
  },
  {
    id: -3,
    email: 'team@company.com',
    accessToken: 'eyJhbGciOiJSUzI1NiIsInR5cCI6...',
    sessionToken: null,
    refreshToken: null,
    clientId: 'team-client-xyz',
    expireTime: Math.floor(Date.now() / 1000) + 86400 * 7,
    valid: true,
    isActive: true,
    isExpired: false,
    planType: 'chatgpt_team',
    isPlus: true,
    usage: 15,
    limit: 20,
    imageCount: 8,
    videoCount: 3,
    errorCount: 1,
    remark: '团队账号',
    proxyUrl: '',
    imageEnabled: true,
    videoEnabled: true,
    imageConcurrency: 5,
    videoConcurrency: 2,
  },
]

// 表格实际渲染的数据：有真实 Token 用真实数据，否则用上面的 FAKE_DEMO_TOKENS
const displayTokens = computed(() => (tokens.value.length ? tokens.value : FAKE_DEMO_TOKENS))
// 当前是否在展示示例（无真实数据时为 true，此时蓝色提示条和顶部统计也是假数据）
const isShowingDemo = computed(() => !loadingTokens.value && !tokens.value.length)

// 示例模式下顶部统计也显示假数据（与 3 条示例 Token 对应）
const displayStats = computed(() => {
  if (!isShowingDemo.value) return adminStore.stats
  return {
    total: 3,
    active: 2,
    todayImages: 5,
    totalImages: 25,
    todayVideos: 3,
    totalVideos: 8,
    todayErrors: 1,
    totalErrors: 4,
  }
})

// Helper: Format Date
const formatDate = (ts) => {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

// Helper: 多少秒后恢复 -> "约X小时后恢复" / "约X天后恢复"
const formatResetTime = (seconds) => {
  if (seconds == null || seconds <= 0) return ''
  const h = Math.floor(seconds / 3600)
  const d = Math.floor(h / 24)
  if (d >= 1) return `约${d}天后恢复`
  if (h >= 1) return `约${h}小时后恢复`
  const m = Math.floor(seconds / 60)
  if (m >= 1) return `约${m}分钟后恢复`
  return `${seconds}秒后恢复`
}

// Helper: Format Plan with Color
const getPlanClass = (type) => {
  if (!type) return 'badge-normal'
  if (['chatgpt_team', 'chatgpt_plus', 'chatgpt_pro'].includes(type)) return 'badge-plus'
  return 'badge-normal'
}

// Selection Logic
const toggleSelectAll = (e) => {
  if (e.target.checked) {
    selectedTokens.value = tokens.value.map(t => t.id)
  } else {
    selectedTokens.value = []
  }
}

// Auto Refresh Logic
const toggleAutoRefresh = async (e) => {
    const checked = e.target.checked
    const success = await adminStore.setATAutoRefresh(checked)
    if (!success) {
        // Revert if failed
        e.target.checked = !checked
        alert('设置自动刷新失败')
    }
}

// Batch Logic
const handleBatch = async (action) => {
  if (!selectedTokens.value.length) return alert('请先选择 Token')

  if (action === 'delete') {
      if (!confirm(`确定要删除选中的 ${selectedTokens.value.length} 个 Token 吗？`)) return
  } else if (action === 'proxy') {
      showBatchProxyModal.value = true
      return
  }

  try {
    switch (action) {
      case 'check': await adminStore.batchCheckTokens(selectedTokens.value); break;
      case 'enable': await adminStore.batchEnableTokens(selectedTokens.value); break;
      case 'disable': await adminStore.batchDisableTokens(selectedTokens.value); break;
      case 'delete': await adminStore.batchDeleteTokens(selectedTokens.value); break;
    }
    if (action !== 'check') { // check usually runs in background or returns quick status, avoiding double alert if api generic
         alert('操作完成')
    } else {
        alert('测试请求已发送，请稍后刷新查看结果')
    }
    selectedTokens.value = []
    adminStore.loadTokens(adminStore.currentPage)
  } catch (e) {
    alert('操作失败: ' + (e.message || '未知错误'))
  }
}

const submitBatchProxy = async () => {
    if (batchSubmitting.value) return
    batchSubmitting.value = true
    try {
        await adminStore.batchUpdateProxy(selectedTokens.value, batchProxyUrl.value)
        alert('批量修改代理成功')
        showBatchProxyModal.value = false
        selectedTokens.value = []
    } catch (e) {
        alert('操作失败: ' + e.message)
    } finally {
        batchSubmitting.value = false
    }
}

// Export
const handleExport = () => {
    // Ideally use backend export, but frontend json generation is fine for now as per legacy
    // Map internal structure to export structure if needed
  const exportData = tokens.value.map(t => ({
      email: t.email,
      access_token: t.accessToken,
      session_token: t.sessionToken,
      refresh_token: t.refreshToken,
      client_id: t.clientId,
      proxy_url: t.proxyUrl,
      remark: t.remark,
      is_active: t.isActive,
      image_enabled: t.imageEnabled,
      video_enabled: t.videoEnabled,
      image_concurrency: t.imageConcurrency,
      video_concurrency: t.videoConcurrency
  }))
  const dataStr = JSON.stringify(exportData, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sora_tokens_${new Date().toISOString().split('T')[0]}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

// Import
const openImportModalTrigger = () => {
  showImportModal.value = true
  importFile.value = null
  importResult.value = null
  importMode.value = 'at'
}

const handleFileSelect = (e) => {
    const file = e.target.files[0]
    if (file && file.name.endsWith('.json')) {
        importFile.value = file
    } else {
        alert('请选择有效的 JSON 文件')
        e.target.value = ''
    }
}

const submitImport = async () => {
    if (!importFile.value) return alert('请选择文件')
    importing.value = true
    importResult.value = null
    try {
        const text = await importFile.value.text()
        const json = JSON.parse(text)
        if (!Array.isArray(json)) throw new Error('JSON 必须是数组格式')

        const res = await adminStore.importTokens(json, importMode.value)
        if (res.success) {
            importResult.value = res
            // Refresh list
            adminStore.loadTokens()
        } else {
            alert('导入失败: ' + (res.detail || '未知错误'))
        }
    } catch (e) {
        alert('导入出错: ' + e.message)
    } finally {
        importing.value = false
    }
}

const handleDelete = async (id) => {
  if (!confirm('确定要删除这个 Token 吗？')) return
  try {
    await adminStore.removeToken(id)
    // removeToken 内部已经会刷新列表，这里只需要刷新统计
    await adminStore.loadStats()
  } catch (e) {
    console.error('删除失败:', e)
    alert('删除失败: ' + (e.message || e))
    // 即使删除失败，也刷新一下列表，确保数据是最新的
    await adminStore.loadTokens(adminStore.currentPage)
  }
}

const handleCheck = async (id) => {
  try {
    await adminStore.verifyToken(id)
    alert('验证请求已发送')
  } catch (e) {
    alert('请求失败:' + e)
  }
}

const handleToggle = async (token) => {
    try {
        await adminStore.toggleToken(token.id, token.isActive)
        // No alert needed, UI updates
    } catch (e) {
        alert('操作失败: ' + e)
    }
}

const changePage = (page) => {
  if (page < 1 || page > totalPages.value) return
  adminStore.loadTokens(page)
}

const totalPages = computed(() => {
  return Math.ceil((adminStore.totalTokens || 0) / adminStore.pageSize)
})

onMounted(() => {
  adminStore.loadTokens()
  adminStore.loadStats()
  adminStore.loadATAutoRefreshConfig()
})
</script>

<template>
  <div class="token-manager">
    <div class="toolbar">
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">Token 总数</div>
          <div class="stat-value">{{ displayStats.total || 0 }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">活跃 Token</div>
          <div class="stat-value highlight-green">{{ displayStats.active || 0 }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">今日图片 / 总数</div>
          <div class="stat-value highlight-blue">
            {{ displayStats.todayImages || 0 }} / {{ displayStats.totalImages || 0 }}
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-label">今日视频 / 总数</div>
          <div class="stat-value highlight-purple">
            {{ displayStats.todayVideos || 0 }} / {{ displayStats.totalVideos || 0 }}
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-label">今日错误 / 总数</div>
          <div class="stat-value highlight-red">
            {{ displayStats.todayErrors || 0 }} / {{ displayStats.totalErrors || 0 }}
          </div>
        </div>
      </div>
    </div>

    <!-- Main List Card -->
    <div class="main-card">
      <div class="card-header">
        <h2 class="card-title">Token 列表</h2>
        <div class="header-actions">
           <!-- Auto Refresh AT Toggle -->
           <div class="auto-refresh-capsule">
               <label class="toggle-switch">
                   <input type="checkbox" :checked="adminStore.atAutoRefreshEnabled" @change="toggleAutoRefresh">
                   <span class="slider round"></span>
               </label>
               <span class="toggle-label">自动刷新 AT</span>
           </div>

           <button class="btn-icon transparent" :class="{ spinning: adminStore.loadingTokens }" @click="() => { adminStore.loadTokens(); adminStore.loadStats(); }" title="刷新数据">
               <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.3"/></svg>
           </button>

           <div class="divider-vertical"></div>

           <!-- Batch Actions Dropdown -->
           <div class="dropdown-wrapper">
             <button class="btn-glimmer-indigo dropdown-trigger">
               <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>
               批量操作
               <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
             </button>
             <div class="dropdown-menu">
               <div class="menu-item action-cyan" @click="handleBatch('check')">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="menu-icon"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                   <span>测试更新</span>
               </div>
               <div class="menu-item action-emerald" @click="handleBatch('enable')">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="menu-icon"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                   <span>批量启用</span>
               </div>
               <div class="menu-item action-amber" @click="handleBatch('disable')">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="menu-icon"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
                   <span>批量禁用</span>
               </div>
               <div class="menu-item action-purple" @click="handleBatch('proxy')">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="menu-icon"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                   <span>修改代理</span>
               </div>
               <div class="menu-divider"></div>
               <div class="menu-item danger" @click="handleBatch('delete')">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="menu-icon"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                   <span>批量删除</span>
               </div>
             </div>
           </div>

           <div class="group-buttons">
               <button class="btn-glimmer-blue" @click="handleExport" title="导出 JSON">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                   导出
               </button>
               <button class="btn-glimmer-emerald" @click="openImportModalTrigger" title="导入 JSON">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                   导入
               </button>
           </div>

           <button class="btn-primary" @click="openAddModal">
             <span>+ 添加 Token</span>
           </button>
        </div>
      </div>

      <div class="list-container">
        <div v-if="loadingTokens && !tokens.length" class="loading-state">
          <div class="spinner"></div> 加载中...
        </div>

        <template v-else>
          <!-- 示例模式时显示的蓝色提示（文案来自这里，与 FAKE_DEMO_TOKENS 配套） -->
          <div v-if="isShowingDemo" class="demo-hint">
            以下为示例数据，请点击「添加 Token」添加真实数据。
          </div>
          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th class="w-checkbox"><input type="checkbox" :disabled="isShowingDemo" @change="toggleSelectAll" /></th>
                  <th>邮箱</th>
                  <th>状态</th>
                  <th>Client ID</th>
                  <th>过期时间</th>
                  <th>账户类型</th>
                  <th>可用次数</th>
                  <th>图片</th>
                  <th>视频</th>
                  <th>错误</th>
                  <th>备注</th>
                  <th class="text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <!-- 表格行：数据来自 displayTokens（空时=FAKE_DEMO_TOKENS，否则=真实 tokens） -->
                <tr v-for="token in displayTokens" :key="token.id" :class="{ 'row-demo': token.id < 0 }">
                  <td><input type="checkbox" v-model="selectedTokens" :value="token.id" :disabled="token.id < 0" /></td>
                  <td class="font-mono text-sm">{{ token.email || 'Unknown' }}</td>
                  <td>
                    <span class="status-dot" :class="token.valid ? 'valid' : 'invalid'" :title="token.valid ? '有效' : '无效'"></span>
                  </td>
                  <td class="font-mono text-xs text-muted" :title="token.clientId">{{ token.clientId ? token.clientId.substring(0,8)+'...' : '-' }}</td>
                  <td class="text-xs">{{ formatDate(token.expireTime) }}</td>
                  <td class="type-cell">
                     <span class="plan-tag" :class="getPlanClass(token.planType)">
                         {{ token.planType ? token.planType.replace('chatgpt_', '').toUpperCase() : 'FREE' }}
                     </span>
                  </td>
                  <td class="font-mono usage-cell" :title="token.accessResetsInSeconds != null ? formatResetTime(token.accessResetsInSeconds) : ''">
                    <span>{{ token.usage ?? 0 }} / {{ token.limit ?? '∞' }}</span>
                    <span v-if="token.accessResetsInSeconds != null && token.accessResetsInSeconds > 0" class="reset-time-hint">{{ formatResetTime(token.accessResetsInSeconds) }}</span>
                  </td>
                  <td class="text-center">{{ token.imageCount ?? 0 }}</td>
                  <td class="text-center">{{ token.videoCount ?? 0 }}</td>
                  <td class="text-center text-red">{{ token.errorCount ?? 0 }}</td>
                  <td class="text-xs text-muted truncate max-w-[100px]" :title="token.remark">{{ token.remark || '-' }}</td>
                  <td>
                    <div class="actions justify-end">
                      <button class="btn-icon-sm" :disabled="token.id < 0" @click="token.id >= 0 && handleCheck(token.id)" title="验证">↻</button>
                      <button class="btn-icon-sm" :disabled="token.id < 0" @click="token.id >= 0 && openEditModal(token)" title="编辑">✎</button>
                      <button class="btn-icon-sm" :class="token.isActive ? 'warning' : 'success'" :disabled="token.id < 0" @click="token.id >= 0 && handleToggle(token)" :title="token.isActive ? '禁用' : '启用'">
                          {{ token.isActive ? '⊘' : 'ok' }}
                      </button>
                      <button class="btn-icon-sm danger" :disabled="token.id < 0" @click="() => { if (token.id >= 0) handleDelete(token.id) }" title="删除">🗑</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>

      <div class="card-footer">
          <div class="page-size-selector">
              每页显示
              <select v-model="adminStore.pageSize" @change="adminStore.loadTokens(1)">
                  <option :value="10">10</option>
                  <option :value="20">20</option>
                  <option :value="50">50</option>
                  <option :value="100">100</option>
              </select>
              条
          </div>
          <div class="pagination-controls" v-if="totalPages > 1">
              <span class="page-info">{{ isShowingDemo ? '共 3 条（示例）' : `共 ${adminStore.totalTokens || 0} 条` }}</span>
              <button class="btn-page" :disabled="currentPage <= 1" @click="changePage(currentPage - 1)">上一页</button>
              <button class="btn-page" :disabled="currentPage >= totalPages" @click="changePage(currentPage + 1)">下一页</button>
          </div>
      </div>
    </div>

    <!-- Modals -->
    <Teleport to="body">
      <!-- Add/Edit Modal -->
      <div v-if="showModal" class="modal-backdrop" @click.self="showModal = false">
        <div class="modal large-modal">
          <div class="modal-header">
            <h3>{{ modalMode === 'add' ? '添加 Token' : '编辑 Token' }}</h3>
            <button class="close-btn" @click="showModal = false">×</button>
          </div>
          <div class="modal-body">
            <div class="field">
              <label>Access Token (AT) <span class="required">*</span></label>
              <textarea v-model="form.accessToken" rows="3" placeholder="eyJh... (JWT格式)" :disabled="converting"></textarea>
            </div>

            <div class="field">
                <label>Session Token (ST)</label>
                <div class="input-group">
                    <textarea v-model="form.sessionToken" rows="2" placeholder="用于转换AT"></textarea>
                    <button class="btn-action" @click="handleConvertST" :disabled="converting">ST→AT</button>
                </div>
            </div>

            <div class="field">
                <label>Refresh Token (RT)</label>
                <div class="input-group">
                    <textarea v-model="form.refreshToken" rows="2" placeholder="用于转换AT"></textarea>
                    <button class="btn-action green" @click="handleConvertRT" :disabled="converting">RT→AT</button>
                </div>
                <p v-if="refreshHintVisible" class="hint-success">✓ RT已被刷新，已填入更新后的RT</p>
            </div>

            <div class="row">
                <div class="field">
                  <label>Client ID</label>
                  <input v-model="form.clientId" type="text" placeholder="留空默认" />
                </div>
                <div class="field">
                  <label>Proxy URL</label>
                  <input v-model="form.proxyUrl" type="text" placeholder="http://..." />
                </div>
            </div>

            <div class="field">
              <label>备注</label>
              <input v-model="form.remark" type="text" placeholder="例如：主账号" />
            </div>

            <div class="section-divider"></div>

            <div class="row">
                <div class="field-checkbox">
                    <label class="checkbox-label">
                       <input type="checkbox" v-model="form.imageEnabled"> 启用图片
                    </label>
                    <input v-model.number="form.imageConcurrency" type="number" class="mini-input" placeholder="并发" title="并发数 (-1不限)" />
                </div>
                <div class="field-checkbox">
                    <label class="checkbox-label">
                       <input type="checkbox" v-model="form.videoEnabled"> 启用视频
                    </label>
                    <input v-model.number="form.videoConcurrency" type="number" class="mini-input" placeholder="并发" title="并发数 (-1不限)" />
                </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-text" @click="showModal = false">取消</button>
            <button class="btn-primary" :disabled="submitting || converting" @click="handleSubmit">
              {{ submitting ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Import Modal -->
      <div v-if="showImportModal" class="modal-backdrop" @click.self="showImportModal = false">
          <div class="modal">
              <div class="modal-header">
                  <h3>导入 Token</h3>
                  <button class="close-btn" @click="showImportModal = false">×</button>
              </div>
              <div class="modal-body">
                  <div class="field">
                      <label>选择文件 (.json)</label>
                      <input type="file" accept=".json" @change="handleFileSelect" class="file-input" />
                  </div>
                  <div class="field">
                      <label>导入模式</label>
                      <select v-model="importMode" class="select-input">
                          <option value="at">AT 导入 (更新状态)</option>
                          <option value="offline">离线导入</option>
                          <option value="st">ST 导入 (自动转换)</option>
                          <option value="rt">RT 导入 (自动转换)</option>
                      </select>
                  </div>

                  <div v-if="importResult" class="import-result">
                      <div class="result-summary">
                          新增: {{ importResult.added }} | 更新: {{ importResult.updated }} | 失败: {{ importResult.failed }}
                      </div>
                  </div>
              </div>
              <div class="modal-footer">
                  <button class="btn-text" @click="showImportModal = false">关闭</button>
                  <button class="btn-primary" :disabled="importing" @click="submitImport">
                      {{ importing ? '导入中...' : '开始导入' }}
                  </button>
              </div>
          </div>
      </div>

      <!-- Batch Proxy Modal -->
      <div v-if="showBatchProxyModal" class="modal-backdrop" @click.self="showBatchProxyModal = false">
          <div class="modal">
              <div class="modal-header">
                  <h3>批量修改代理</h3>
                  <button class="close-btn" @click="showBatchProxyModal = false">×</button>
              </div>
              <div class="modal-body">
                  <div class="field">
                      <label>代理地址</label>
                      <input v-model="batchProxyUrl" type="text" placeholder="http://127.0.0.1:7890" />
                      <p class="hint">留空则清除代理设置</p>
                  </div>
                  <p class="info-text">将应用于 {{ selectedTokens.length }} 个选中的 Token</p>
              </div>
              <div class="modal-footer">
                  <button class="btn-text" @click="showBatchProxyModal = false">取消</button>
                  <button class="btn-primary" :disabled="batchSubmitting" @click="submitBatchProxy">
                      {{ batchSubmitting ? '保存中...' : '确认修改' }}
                  </button>
              </div>
          </div>
      </div>
    </Teleport>

  </div>
</template>

<style scoped>
/* Main Layout - 随分辨率自适应 */
.token-manager {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  height: 100%;
  min-width: 0;
  margin: 0 auto;
  overflow: hidden;
}

/* Stats (Top) - 响应式列数 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 20px;
  margin-bottom: 8px;
}
@media (max-width: 1200px) {
  .stats-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 768px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
}
@media (max-width: 480px) {
  .stats-grid { grid-template-columns: 1fr; }
}
/* Stat Cards */
.stat-card {
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 16px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s ease;
  backdrop-filter: blur(10px);
}
.stat-card:hover { border-color: rgba(56, 189, 248, 0.4); transform: translateY(-4px); box-shadow: 0 10px 30px -10px rgba(0,0,0,0.5); }

.stat-label { font-size: 12px; color: #94a3b8; font-weight: 500; }
.stat-value { font-size: 18px; font-weight: 700; color: #f1f5f9; font-family: 'SF Pro Display', sans-serif; }
.highlight-green { color: #4ade80; }
.highlight-blue { color: #60a5fa; }
.highlight-purple { color: #a78bfa; }
.highlight-red { color: #f87171; }

/* Dropdown Fix */
.dropdown-menu {
  /* ... existing basic styles will be merged or overridden below if needed, but let's ensure positioning matches new header ... */
  right: 0;
  left: auto; /* Align to right since it's on the right side now */
  min-width: 160px;
}

/* Main Card Container - 宽度随父级变化，子元素同宽 */
.main-card {
  background: #1e293b; /* Fallback */
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  backdrop-filter: blur(10px);
  width: 100%;
  min-width: 0;
  overflow: hidden;
  flex: 1;
  min-height: 0;
}
.main-card > .list-container {
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Header - 小屏时标题与操作区换行 */
.card-header {
  padding: 16px 24px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  background: rgba(30, 41, 59, 0.2);
  border-top-left-radius: 12px;
  border-top-right-radius: 12px;
}
@media (max-width: 900px) {
  .card-header { padding: 12px 16px; }
}

/* --- Dropdown Colors (Pro Max) --- */
.menu-item.action-cyan { color: #22d3ee; }
.menu-item.action-cyan:hover { background: rgba(34, 211, 238, 0.1); }

.menu-item.action-emerald { color: #34d399; }
.menu-item.action-emerald:hover { background: rgba(52, 211, 153, 0.1); }

.menu-item.action-amber { color: #fbbf24; }
.menu-item.action-amber:hover { background: rgba(251, 191, 36, 0.1); }

.menu-item.action-purple { color: #c084fc; }
.menu-item.action-purple:hover { background: rgba(192, 132, 252, 0.1); }

.menu-item.danger { color: #f87171; }
.menu-item.danger:hover { background: rgba(248, 113, 113, 0.1); }


.card-title {
  font-size: 18px;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
@media (max-width: 640px) {
  .header-actions { gap: 8px; }
}

.dropdown-menu {
    display: none;
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-5%);
    right: auto;
    margin-top: 8px; /* Visual gap */
    background: #1e293b;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 8px;
    min-width: 120px;
    z-index: 100;
    box-shadow: 0 10px 25px -5px rgba(0,0,0,0.5);
    overflow: visible; /* Allow pseudo-element outside */
    padding: 4px 0;
}

/* Invisible bridge for hover stability */
.dropdown-menu::before {
    content: '';
    position: absolute;
    top: -10px;
    left: 0;
    width: 100%;
    height: 10px;
}

/* Button & Controls Overrides */
.btn-glass {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #e2e8f0;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}
.btn-glass:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.btn-primary-outline {
  background: transparent;
  border: 1px solid #3b82f6;
  color: #3b82f6;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-primary-outline:hover { background: rgba(59, 130, 246, 0.1); }

.btn-success-outline {
  background: transparent;
  border: 1px solid #10b981;
  color: #10b981;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-success-outline:hover { background: rgba(16, 185, 129, 0.1); }


/* Table Styling Updates - 与父级同宽，保证表格区域可见 */
.list-container {
  padding: 0;
  margin: 0;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
}
.table-wrapper {
  border-radius: 0;
  border: none;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow-x: auto;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
}
.table-wrapper table {
  width: 100%;
  min-width: 800px;
  table-layout: auto;
  display: table;
  flex-shrink: 0;
}
table { border-collapse: separate; border-spacing: 0; }
th { background: rgba(15, 23, 42, 0.3); border-bottom: 1px solid rgba(148, 163, 184, 0.1); padding: 14px 16px; font-weight: 600; font-size: 13px; }
td { border-bottom: 1px solid rgba(148, 163, 184, 0.05); padding: 12px 16px; }
tr:last-child td { border-bottom: none; }
.usage-cell { display: flex; flex-direction: column; gap: 2px; }
.usage-cell .reset-time-hint { font-size: 11px; color: #94a3b8; }

/* Status Dot */
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.status-dot.valid { background: #4ade80; box-shadow: 0 0 8px rgba(74, 222, 128, 0.4); }
.status-dot.invalid { background: #f87171; }

/* Plan Tag */
.plan-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 12px; /* Pill shape */
  font-weight: 600;
  border: 1px solid transparent;
}
.badge-plus { background: rgba(245, 158, 11, 0.1); color: #fbbf24; border-color: rgba(245, 158, 11, 0.2); }
.badge-normal { background: rgba(148, 163, 184, 0.1); color: #94a3b8; }

/* Action Buttons Small */
/* 操作列：按钮横排 */
.actions {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
}
.actions.justify-end { justify-content: flex-end; }

.btn-icon-sm {
  width: 24px;
  height: 24px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  flex-shrink: 0;
  margin-left: 0;
}
.btn-icon-sm:first-child { margin-left: 0; }
.btn-icon-sm:hover { color: #f1f5f9; border-color: rgba(148, 163, 184, 0.6); }
.btn-icon-sm.danger:hover { color: #f87171; border-color: #f87171; }


/* Footer - 小屏换行 */
.card-footer {
  padding: 12px 24px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  background: rgba(30, 41, 59, 0.2);
}
@media (max-width: 600px) {
  .card-footer { padding: 12px 16px; }
}

.page-size-selector {
  font-size: 13px;
  color: #94a3b8;
  display: flex;
  align-items: center;
  gap: 8px;
}
.page-size-selector select {
    background: #0f172a;
    border: 1px solid #334155;
    color: #e2e8f0;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
}


.modal {
  background: #1e293b;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  width: 100%;
  max-width: 500px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.modal-header h3 { margin: 0; font-size: 18px; color: #f1f5f9; }
.close-btn { background: none; border: none; font-size: 24px; color: #64748b; cursor: pointer; }

.modal-body { padding: 20px; display: flex; flex-direction: column; gap: 16px; }

.field { display: flex; flex-direction: column; gap: 6px; }
.field label { font-size: 12px; color: #94a3b8; font-weight: 500; }
.field input, .field textarea {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 10px;
  color: #f1f5f9;
  font-family: inherit;
  outline: none;
}
.field input:focus, .field textarea:focus { border-color: #3b82f6; }

.row { display: flex; gap: 16px; }
.row .field { flex: 1; }

.modal-footer {
  padding: 16px 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-text {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 8px 16px;
}
.btn-text:hover { color: #f1f5f9; }

.loading-state, .empty-state {
  padding: 40px;
  text-align: center;
  color: #94a3b8;
  font-size: 14px;
}

.demo-hint {
  width: 100%;
  box-sizing: border-box;
  padding: 10px 16px;
  margin: 0;
  flex-shrink: 0;
  background: rgba(59, 130, 246, 0.12);
  border: 1px solid rgba(59, 130, 246, 0.25);
  border-radius: 8px;
  color: #93c5fd;
  font-size: 13px;
  text-align: center;
}
.row-demo td { opacity: 0.95; }
.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
  vertical-align: middle;
}
@keyframes spin { to { transform: rotate(360deg); } }
/* Auto Refresh & Toggle */
.auto-refresh-control {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(30, 41, 59, 0.4);
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.1);
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 32px;
  height: 18px;
}
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute;
  cursor: pointer;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: #475569;
  transition: .4s;
  border-radius: 34px;
}
.slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  transition: .4s;
  border-radius: 50%;
}
input:checked + .slider { background-color: #3b82f6; }
input:checked + .slider:before { transform: translateX(14px); }

/* Tooltip */
.tooltip-wrapper { position: relative; display: inline-flex; align-items: center; cursor: help; }
.tooltip-icon {
  background: rgba(148, 163, 184, 0.2);
  color: #94a3b8;
  border-radius: 50%;
  width: 16px;
  height: 16px;
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tooltip-content {
  visibility: hidden;
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background-color: #1e293b;
  color: #f1f5f9;
  text-align: center;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 11px;
  white-space: nowrap;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.2);
  z-index: 10;
  margin-bottom: 6px;
  opacity: 0;
  transition: opacity 0.2s;
}
.tooltip-wrapper:hover .tooltip-content { visibility: visible; opacity: 1; }



/* Import/Export & Form Extras */
.file-input, .select-input {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 8px;
  color: #f1f5f9;
  width: 100%;
  font-size: 13px;
}

.import-result {
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.1);
  padding: 12px;
  border-radius: 8px;
}
.result-summary { font-size: 13px; color: #cbd5e1; text-align: center; }

.hint { font-size: 12px; color: #64748b; margin-top: 4px; margin-bottom: 0px; }
/* Button Standards */
.btn-glass {
  height: 36px;
  padding: 0 16px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  white-space: nowrap;
}
.btn-glass:hover {
  background: rgba(51, 65, 85, 0.8);
  border-color: rgba(148, 163, 184, 0.4);
  color: #fff;
  transform: translateY(-1px);
}
.btn-glass:active { transform: translateY(0); }

.btn-primary {
  height: 36px;
  padding: 0 20px;
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  border: none;
  border-radius: 8px;
  color: white;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  white-space: nowrap;
}
.btn-primary:hover {
  opacity: 0.95;
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(59, 130, 246, 0.4);
}
.btn-primary:active { transform: translateY(0); }

/* Auto Refresh Pill */
.auto-refresh-control {
  height: 36px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px 0 16px;
  background: rgba(15, 23, 42, 0.3);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 20px; /* Pill shape */
  margin-right: 8px;
}
.auto-refresh-control span {
    font-size: 12px;
    color: #94a3b8;
    font-weight: 500;
}

/* Refresh Icon Button */
.btn-icon-refresh {
  height: 36px;
  width: 36px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(30, 41, 59, 0.6);
  color: #94a3b8;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: all 0.2s;
}
.btn-icon-refresh:hover {
  background: #334155;
  color: #fff;
  border-color: rgba(148, 163, 184, 0.4);
}
.btn-icon-refresh.spinning {
    cursor: wait;
    opacity: 0.8;
}
.btn-icon-refresh.spinning .refresh-symbol {
    display: inline-block;
    animation: spin 1s linear infinite;
}


.mini-input { width: 80px !important; text-align: center; }

.field-checkbox {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #0f172a;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid #334155;
}
.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #e2e8f0;
  cursor: pointer;
  user-select: none;
}
.section-divider {
  height: 1px;
  background: rgba(148, 163, 184, 0.1);
  margin: 8px 0;
}

.input-group { display: flex; gap: 8px; }
.input-group textarea { flex: 1; }
.btn-action {
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  padding: 0 12px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.btn-action.green { background: #10b981; }
.btn-action:hover { opacity: 0.9; }
.btn-action:disabled { opacity: 0.5; cursor: not-allowed; }

.hint-success { color: #4ade80; font-size: 12px; margin-top: 4px; }

/* Modal Fix */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999 !important;
}
</style>

<style scoped>
/* --- Standard Utilities --- */
.btn-secondary {
    background: #334155;
    color: #f1f5f9;
    border: none;
    padding: 9px 16px;
    border-radius: 8px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    transition: background 0.2s;
}
.btn-secondary:hover { background: #475569; }

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #e2e8f0;
}
.checkbox-row.compact {
    font-size: 13px;
    padding: 0 8px;
    background: rgba(30, 41, 59, 0.4);
    border-radius: 6px;
    border: 1px solid rgba(148, 163, 184, 0.1);
    height: 36px;
}

/* Dropdown Interaction */
.dropdown-wrapper {
    position: relative;
    display: inline-block;
}
.dropdown-wrapper:hover .dropdown-menu {
    display: block;
}
.dropdown-menu {
    display: none;
    position: absolute;
    top: 100%;
    right: 0;
    left: auto;
    margin-top: 4px;
    background: #1e293b;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 8px;
    min-width: 120px;
    z-index: 100;
    box-shadow: 0 10px 25px -5px rgba(0,0,0,0.5);
    padding: 4px 0;
    /* display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center; */
}
.menu-item {
    padding: 10px 12px;
    font-size: 13px;
    color: #cbd5e1;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 10px;
}
.menu-item:hover {
    background: rgba(51, 65, 85, 0.5);
    color: #f1f5f9;
}
.menu-icon {
    opacity: 0.7;
    transition: opacity 0.2s;
}
.menu-item:hover .menu-icon {
    opacity: 1;
}

.menu-item.danger { color: #f87171; }
.menu-item.danger:hover { background: rgba(248, 113, 113, 0.1); }

.menu-divider {
    height: 1px;
    background: rgba(148, 163, 184, 0.1);
    margin: 4px 0;
}


/* Auto Refresh Capsule */
.auto-refresh-capsule {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 12px 0 8px;
    background: rgba(30, 41, 59, 0.6); /* Slate-800/60 */
    border: 1px solid rgba(148, 163, 184, 0.1);
    border-radius: 20px; /* Pill */
    height: 36px;
    margin-right: 8px;
}

/* Base Glimmer Button */
.btn-glimmer-indigo, .btn-glimmer-blue, .btn-glimmer-emerald {
    height: 36px;
    padding: 0 16px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    border: none;
    color: white;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
    position: relative;
    overflow: hidden;
}

/* Indigo Glimmer */
.btn-glimmer-indigo {
    background: linear-gradient(135deg, #4f46e5, #4338ca); /* Indigo 600-700 */
    box-shadow: 0 0 10px rgba(79, 70, 229, 0.3);
}
.btn-glimmer-indigo:hover {
    background: linear-gradient(135deg, #6366f1, #4f46e5);
    box-shadow: 0 0 15px rgba(99, 102, 241, 0.5);
    transform: translateY(-1px);
}

/* Blue Glimmer (Export) */
.btn-glimmer-blue {
    background: linear-gradient(135deg, #2563eb, #1d4ed8); /* Blue 600-700 */
    box-shadow: 0 0 10px rgba(37, 99, 235, 0.3);
}
.btn-glimmer-blue:hover {
    background: linear-gradient(135deg, #3b82f6, #2563eb);
    box-shadow: 0 0 15px rgba(59, 130, 246, 0.5);
    transform: translateY(-1px);
}

/* Emerald Glimmer (Import) */
.btn-glimmer-emerald {
    background: linear-gradient(135deg, #059669, #047857); /* Emerald 600-700 */
    box-shadow: 0 0 10px rgba(16, 185, 129, 0.3);
}
.btn-glimmer-emerald:hover {
    background: linear-gradient(135deg, #10b981, #059669);
    box-shadow: 0 0 15px rgba(52, 211, 153, 0.5);
    transform: translateY(-1px);
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 32px;
  height: 18px;
  cursor: pointer;
}
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute;
  cursor: pointer;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: #475569;
  transition: .4s;
}
.slider.round { border-radius: 34px; }
.slider.round:before { border-radius: 50%; }
.slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  transition: .4s;
}
input:checked + .slider { background-color: #3b82f6; box-shadow: 0 0 8px rgba(59, 130, 246, 0.5); }
input:checked + .slider:before { transform: translateX(14px); }
.toggle-label { font-size: 12px; color: #cbd5e1; font-weight: 500; }

.btn-icon.transparent {
    background: transparent;
    border: none;
    color: #94a3b8;
    padding: 6px;
    height: auto;
    width: auto;
}
.btn-icon.transparent:hover {
    color: #f1f5f9;
    cursor: pointer;
}

.divider-vertical {
    width: 1px;
    height: 24px;
    background: rgba(148, 163, 184, 0.2);
    margin: 0 8px;
}

.group-buttons {
    display: flex;
    gap: 8px;
}
</style>
