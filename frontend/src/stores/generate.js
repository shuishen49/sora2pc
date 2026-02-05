import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { streamCompletion } from '../api/generate'
import { humanizeUpstreamError } from '../utils/errorUtils'

// ========== Helpers ==========
const fileToDataUrl = (file) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(e.target.result)
    reader.onerror = (e) => reject(e)
    reader.readAsDataURL(file)
  })
}



export const useGenerateStore = defineStore('generate', () => {
  // ========== Configuration ==========
  const apiKey = ref(localStorage.getItem('sora_api_key') || '')

  // Base URL 默认值
  const baseUrl = ref('http://127.0.0.1:8000')

  // 如果在 Wails 环境下，优先从本地 config 文件读取
  if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetBaseURL) {
    window.go.main.App.GetBaseURL()
      .then((val) => {
        if (val) {
          baseUrl.value = val
        }
      })
      .catch((err) => {
        console.error('GetBaseURL failed:', err)
        const saved = localStorage.getItem('sora_base_url')
        if (saved) baseUrl.value = saved
      })
  } else {
    // 非 Wails 环境（纯浏览器调试）依然使用 localStorage
    const saved = localStorage.getItem('sora_base_url')
    if (saved) baseUrl.value = saved
  }

  const setApiKey = (val) => {
    apiKey.value = val
    localStorage.setItem('sora_api_key', val)
  }

  const setBaseUrl = (val) => {
    baseUrl.value = val

    // 在 Wails 中写入本地 config 文件
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetBaseURL) {
      window.go.main.App.SetBaseURL(val).catch((err) => {
        console.error('SetBaseURL failed:', err)
      })
    } else {
      // 纯浏览器环境兜底到 localStorage
      localStorage.setItem('sora_base_url', val)
    }
  }

  // ========== Models ==========
  // 暂只做普通视频（标准版）10s 和 15s
  const models = [
    { value: 'sora2-landscape-15s', label: '横屏视频 15s', group: '标准版视频' },
    { value: 'sora2-landscape-10s', label: '横屏视频 10s', group: '标准版视频' },
    { value: 'sora2-portrait-15s', label: '竖屏视频 15s', group: '标准版视频' },
    { value: 'sora2-portrait-10s', label: '竖屏视频 10s', group: '标准版视频' },
    { value: 'gpt-image', label: '方图 360×360', group: '图片' },
    { value: 'gpt-image-landscape', label: '横图 540×360', group: '图片' },
    { value: 'gpt-image-portrait', label: '竖图 360×540', group: '图片' },
  ]

  const modelGroups = computed(() => {
    const groups = {}
    models.forEach(m => {
      if (!groups[m.group]) groups[m.group] = []
      groups[m.group].push(m)
    })
    return groups
  })

  // ========== State ==========
  const selectedModel = ref('sora2-landscape-10s')
  const batchMode = ref('single')
  const tasks = ref([])
  const attachedRoles = ref([])
  const favoriteRoles = ref([]) // IDs or Names
  const logs = ref([])

  const addLog = (message, type = 'info') => {
      const time = new Date().toLocaleTimeString()
      logs.value.push({ time, message, type })
      // Keep max logs
      if (logs.value.length > 200) logs.value.shift()
  }

  const clearLogs = () => logs.value = []

  // Load favorites
  try {
      const saved = localStorage.getItem('sora_fav_roles')
      if (saved) favoriteRoles.value = JSON.parse(saved)
  } catch (_) {
      // ignore
  }

  const toggleRoleFavorite = (roleName) => {
      if (favoriteRoles.value.includes(roleName)) {
          favoriteRoles.value = favoriteRoles.value.filter(n => n !== roleName)
      } else {
          favoriteRoles.value.push(roleName)
      }
      localStorage.setItem('sora_fav_roles', JSON.stringify(favoriteRoles.value))
  }

  const isRoleFavorite = (roleName) => favoriteRoles.value.includes(roleName)

  const attachRole = (role) => {
      if (!attachedRoles.value.find(r => r.name === role.name)) {
          attachedRoles.value.push(role)
      }
  }

  const detachRole = (roleName) => {
      attachedRoles.value = attachedRoles.value.filter(r => r.name !== roleName)
  }

  // 任务列表：Wails 下从 SQLite 读取（在 loadTaskList 中异步加载），否则从 localStorage 同步加载
  if (!window.go?.main?.App?.GetTaskList) {
      try {
          const saved = localStorage.getItem('sora_tasks_v2')
          if (saved) tasks.value = JSON.parse(saved)
      } catch (e) {
          console.warn('Failed to load tasks', e)
      }
  }

  const loadTaskList = async () => {
      if (!window.go?.main?.App?.GetTaskList) return
      try {
          const res = await window.go.main.App.GetTaskList()
          const str = typeof res === 'string' ? res : JSON.stringify(res)
          const arr = JSON.parse(str)
          if (Array.isArray(arr)) tasks.value = arr
      } catch (e) {
          console.warn('Load task list from SQLite failed', e)
      }
  }

  const persistTasks = () => {
      const json = JSON.stringify(tasks.value.slice(0, 50))
      if (window.go?.main?.App?.SetTaskList) {
          window.go.main.App.SetTaskList(json).catch(() => {})
      } else {
          localStorage.setItem('sora_tasks_v2', json)
      }
  }

  const addTask = (task) => {
    tasks.value.unshift(task)
    persistTasks()
  }

  const updateTask = (id, updates) => {
    const t = tasks.value.find(t => t.id === id)
    if (t) {
      Object.assign(t, updates)
      persistTasks()
    }
  }

  const removeTask = (id) => {
    tasks.value = tasks.value.filter(t => t.id !== id)
    persistTasks()
  }

  const clearAllTasks = () => {
      tasks.value = []
      persistTasks()
  }

  // 测试用：清除 localStorage 中的任务列表并刷新页面，使下次加载仅从 SQLite 恢复 pending（孤儿任务）
  const clearLocalTasksAndReload = () => {
      localStorage.removeItem('sora_tasks_v2')
      window.location.reload()
  }

  // ========== Execution ==========
  const abortControllers = new Map() // taskId -> AbortController
  const pendingIntervals = new Map() // taskId -> intervalId（CreateVideo 成功后每 10s 轮询 pending）

  // 为指定任务启动 pending 轮询（每 10s 一次，用该任务记录的账号）；页面加载时可为未完成任务恢复轮询
  const startPendingInterval = (taskId) => {
     if (!window.go?.main?.App?.PollPending || pendingIntervals.has(taskId)) return
     const apiBase = baseUrl.value.replace(/\/$/, '')
     const POLL_INTERVAL_MS = 10 * 1000
     const tick = async () => {
         const current = tasks.value.find(x => x.id === taskId)
         if (!current || current.status === 'done' || current.status === 'failed') {
             const id = pendingIntervals.get(taskId)
             if (id != null) clearInterval(id)
             pendingIntervals.delete(taskId)
             return
         }
         try {
             let bearer = ''
             let tokenId = current.tokenIdForPending
             // 若任务未记录 tokenIdForPending（如旧任务或刷新前创建的），从 video_task_results 按 remoteTaskId 查 token_id
             if (tokenId == null && current.remoteTaskId && window.go?.main?.App?.GetTokenIDByRemoteTaskID) {
                 const res = await window.go.main.App.GetTokenIDByRemoteTaskID(current.remoteTaskId)
                 const data = typeof res === 'string' ? JSON.parse(res) : res
                 if (data?.token_id != null) {
                     tokenId = data.token_id
                     updateTask(taskId, { tokenIdForPending: tokenId })
                 }
             }
             if (tokenId != null && window.go?.main?.App?.GetBearerByTokenID) {
                 const res = await window.go.main.App.GetBearerByTokenID(tokenId)
                 const data = typeof res === 'string' ? JSON.parse(res) : res
                 if (data?.bearer_token) bearer = data.bearer_token
             }
             if (!bearer) {
                 addLog(`Task ${taskId} 无法获取账号 bearer，跳过 pending`, 'warning')
                 return
             }
             const body = await window.go.main.App.PollPending(apiBase, bearer)
             const list = typeof body === 'string' ? JSON.parse(body) : body

             // 如何知道「继续 pending」还是「已完成」：完全由 pending 接口的返回决定
             // - 有任务：返回非空列表（如 [{ id, progress_pct, status, ... }] 或 { tasks: [...] }）→ 继续轮询，用 progress_pct 更新进度
             // - 已完成：返回空列表（[] 或 { raw: [], tasks: [] }）→ 该账号下没有 pending 任务，说明视频已生成完毕，停止轮询并执行 drafts
             const isEmpty = Array.isArray(list)
                 ? list.length === 0
                 : (list && Array.isArray(list.tasks) && list.tasks.length === 0)
             if (isEmpty) {
                 const id = pendingIntervals.get(taskId)
                 if (id != null) clearInterval(id)
                 pendingIntervals.delete(taskId)
                 const remoteId = current.remoteTaskId
                 if (remoteId && window.go?.main?.App?.UpdateVideoTaskProgress) {
                     window.go.main.App.UpdateVideoTaskProgress(remoteId, 100).catch(() => {})
                 }
                 updateTask(taskId, { status: 'done', progress: 100, message: '已完成（pending 返回 []）' })
                 addLog(`Task ${taskId} pending 返回 []，任务完成`, 'info')
                 // pending 返回空后执行 drafts，仅下载该 task_id 对应的那条
                 if (window.go?.main?.App?.FetchDrafts && window.go?.main?.App?.SaveDraftsAndDownload) {
                     try {
                         const draftsBody = await window.go.main.App.FetchDrafts(apiBase, bearer)
                         const draftsJson = typeof draftsBody === 'string' ? draftsBody : JSON.stringify(draftsBody)
                         await window.go.main.App.SaveDraftsAndDownload(draftsJson, remoteId)
                         addLog(`Task ${taskId} 已拉取 drafts 并下载`, 'info')
                     } catch (e) {
                         addLog(`Task ${taskId} drafts 拉取/下载失败: ${e?.message || e}`, 'warning')
                     }
                 }
                 return
             }
             // 有任务：取第一个的 progress（兼容 [] 或 { tasks: [...] }）
             const tasksArr = Array.isArray(list) ? list : (list?.tasks || [])
             const first = tasksArr[0]
             const rawPct = first.progress_pct != null ? first.progress_pct : 0
             // progress_pct 可能是 0-1 的小数或 0-100 的整数
             const pct = rawPct <= 1 ? rawPct * 100 : rawPct
             const remoteId = current.remoteTaskId
             if (remoteId && window.go?.main?.App?.UpdateVideoTaskProgress) {
                 window.go.main.App.UpdateVideoTaskProgress(remoteId, pct).catch(() => {})
             }
             updateTask(taskId, { progress: pct, message: `pending 进度 ${pct.toFixed(0)}%` })
             
             // 如果进度达到 100%（progress_pct >= 1 或 >= 100），停止轮询并去草稿箱拉取
             if (rawPct >= 1 || pct >= 100) {
                 const id = pendingIntervals.get(taskId)
                 if (id != null) clearInterval(id)
                 pendingIntervals.delete(taskId)
                 updateTask(taskId, { status: 'done', progress: 100, message: '已完成（progress_pct=100%）' })
                 addLog(`Task ${taskId} progress_pct=100%，任务完成，去草稿箱拉取`, 'info')
                 // 去草稿箱拉取结果
                 if (window.go?.main?.App?.FetchDrafts && window.go?.main?.App?.SaveDraftsAndDownload) {
                     try {
                         const draftsBody = await window.go.main.App.FetchDrafts(apiBase, bearer)
                         const draftsJson = typeof draftsBody === 'string' ? draftsBody : JSON.stringify(draftsBody)
                         await window.go.main.App.SaveDraftsAndDownload(draftsJson, remoteId)
                         addLog(`Task ${taskId} 已拉取 drafts 并下载`, 'info')
                     } catch (e) {
                         addLog(`Task ${taskId} drafts 拉取/下载失败: ${e?.message || e}`, 'warning')
                     }
                 }
                 return
             }
         } catch (err) {
             addLog(`Task ${taskId} pending 轮询失败: ${err?.message || err}`, 'warning')
         }
     }
     tick()
     pendingIntervals.set(taskId, setInterval(tick, POLL_INTERVAL_MS))
  }

  // 孤儿任务：仅存在于 SQLite，无本地 UI 任务；只轮询 pending、更新 DB、完成后拉取 drafts
  const startOrphanPending = (remoteTaskId, tokenId) => {
     const key = 'remote:' + remoteTaskId
     if (!window.go?.main?.App?.PollPending || pendingIntervals.has(key)) return
     const apiBase = baseUrl.value.replace(/\/$/, '')
     const POLL_INTERVAL_MS = 10 * 1000
     const tick = async () => {
         try {
             let bearer = ''
             if (tokenId != null && window.go?.main?.App?.GetBearerByTokenID) {
                 const res = await window.go.main.App.GetBearerByTokenID(tokenId)
                 const data = typeof res === 'string' ? JSON.parse(res) : res
                 if (data?.bearer_token) bearer = data.bearer_token
             }
             if (!bearer) {
                 addLog(`孤儿任务 ${remoteTaskId} 无法获取 bearer，停止轮询`, 'warning')
                 const id = pendingIntervals.get(key)
                 if (id != null) clearInterval(id)
                 pendingIntervals.delete(key)
                 return
             }
             const body = await window.go.main.App.PollPending(apiBase, bearer)
             const list = typeof body === 'string' ? JSON.parse(body) : body
             const isEmpty = Array.isArray(list)
                 ? list.length === 0
                 : (list && Array.isArray(list.tasks) && list.tasks.length === 0)
             if (isEmpty) {
                 const id = pendingIntervals.get(key)
                 if (id != null) clearInterval(id)
                 pendingIntervals.delete(key)
                 if (window.go?.main?.App?.UpdateVideoTaskProgress) {
                     window.go.main.App.UpdateVideoTaskProgress(remoteTaskId, 100).catch(() => {})
                 }
                 addLog(`孤儿任务 ${remoteTaskId} pending 完成，拉取 drafts`, 'info')
                 if (window.go?.main?.App?.FetchDrafts && window.go?.main?.App?.SaveDraftsAndDownload) {
                     try {
                         const draftsBody = await window.go.main.App.FetchDrafts(apiBase, bearer)
                         const draftsJson = typeof draftsBody === 'string' ? draftsBody : JSON.stringify(draftsBody)
                         await window.go.main.App.SaveDraftsAndDownload(draftsJson, remoteTaskId)
                     } catch (e) {
                         addLog(`孤儿任务 ${remoteTaskId} drafts 失败: ${e?.message || e}`, 'warning')
                     }
                 }
                 return
             }
             const tasksArr = Array.isArray(list) ? list : (list?.tasks || [])
             const first = tasksArr[0]
             const rawPct = first.progress_pct != null ? first.progress_pct : 0
             // progress_pct 可能是 0-1 的小数或 0-100 的整数
             const pct = rawPct <= 1 ? rawPct * 100 : rawPct
             if (window.go?.main?.App?.UpdateVideoTaskProgress) {
                 window.go.main.App.UpdateVideoTaskProgress(remoteTaskId, pct).catch(() => {})
             }
             
             // 如果进度达到 100%（progress_pct >= 1 或 >= 100），停止轮询并去草稿箱拉取
             if (rawPct >= 1 || pct >= 100) {
                 const id = pendingIntervals.get(key)
                 if (id != null) clearInterval(id)
                 pendingIntervals.delete(key)
                 addLog(`孤儿任务 ${remoteTaskId} progress_pct=100%，任务完成，去草稿箱拉取`, 'info')
                 // 去草稿箱拉取结果
                 if (window.go?.main?.App?.FetchDrafts && window.go?.main?.App?.SaveDraftsAndDownload) {
                     try {
                         const draftsBody = await window.go.main.App.FetchDrafts(apiBase, bearer)
                         const draftsJson = typeof draftsBody === 'string' ? draftsBody : JSON.stringify(draftsBody)
                         await window.go.main.App.SaveDraftsAndDownload(draftsJson, remoteTaskId)
                         addLog(`孤儿任务 ${remoteTaskId} 已拉取 drafts 并下载`, 'info')
                     } catch (e) {
                         addLog(`孤儿任务 ${remoteTaskId} drafts 拉取/下载失败: ${e?.message || e}`, 'warning')
                     }
                 }
                 return
             }
         } catch (err) {
             addLog(`孤儿任务 ${remoteTaskId} pending 轮询失败: ${err?.message || err}`, 'warning')
         }
     }
     tick()
     pendingIntervals.set(key, setInterval(tick, POLL_INTERVAL_MS))
  }

  // 页面加载时：从 SQLite 读取未完成视频任务，恢复 pending 轮询（不依赖 localStorage 任务列表）
  const startPendingForExistingTasks = async () => {
     if (!window.go?.main?.App?.PollPending || !window.go?.main?.App?.GetIncompleteVideoTasks) return
     try {
         const res = await window.go.main.App.GetIncompleteVideoTasks()
         const data = typeof res === 'string' ? JSON.parse(res) : res
         if (data?.error) {
             addLog('获取未完成视频任务失败: ' + data.error, 'warning')
             return
         }
         const list = data?.tasks || []
         addLog(`[SQLite] 读取未完成视频任务: ${JSON.stringify(list)}`, 'info')
         addLog(`[SQLite] 共 ${list.length} 条，是否需要继续 pending: ${list.length > 0 ? '是' : '否'}`, 'info')
         for (const item of list) {
             const remoteTaskId = item?.task_id
             const tokenId = item?.token_id
             if (!remoteTaskId || tokenId == null) {
                 addLog(`[pending] 跳过无效项: task_id=${remoteTaskId} token_id=${tokenId}`, 'info')
                 continue
             }
             const key = 'remote:' + remoteTaskId
             if (pendingIntervals.has(key)) {
                 addLog(`[pending] ${remoteTaskId} 已在轮询中，跳过`, 'info')
                 continue
             }
             const local = tasks.value.find(t => t.remoteTaskId === remoteTaskId)
             if (local) {
                 if (local.status === 'done' || local.status === 'failed') {
                     addLog(`[pending] ${remoteTaskId} 本地状态已为 ${local.status}，跳过`, 'info')
                     continue
                 }
                 if (pendingIntervals.has(local.id)) {
                     addLog(`[pending] ${remoteTaskId} 本地任务 ${local.id} 已在轮询，跳过`, 'info')
                     continue
                 }
                 if (local.tokenIdForPending == null) updateTask(local.id, { tokenIdForPending: tokenId })
                 addLog(`[pending] ${remoteTaskId} 继续 pending（本地任务 ${local.id}）`, 'info')
                 startPendingInterval(local.id)
             } else {
                 addLog(`[pending] ${remoteTaskId} 继续 pending（孤儿任务，无本地条目）`, 'info')
                 startOrphanPending(remoteTaskId, tokenId)
             }
         }
     } catch (e) {
         addLog('恢复 pending 失败: ' + (e?.message || e), 'warning')
     }
  }

  const runTask = async (taskId) => {
     const t = tasks.value.find(x => x.id === taskId)
     if (!t) return

     // Update Status
     updateTask(taskId, { status: 'running', progress: 0, message: '' })

     addLog(`Starting task ${taskId}`, 'info')

     // Prepare Payload
     const contentArr = []

     // Append Attached Roles
     let finalPrompt = t.prompt || ''
     if (attachedRoles.value.length) {
         const rolePrompts = attachedRoles.value.map(r => r.prompt).join(' ')
         if (rolePrompts) {
             finalPrompt = finalPrompt ? `${finalPrompt} ${rolePrompts}` : rolePrompts
         }
     }

     if (finalPrompt) contentArr.push({ type: 'text', text: finalPrompt })

     // Handle File (Stored as _fileDataUrl or we need to read file object if fresh)
     // In store state, we can't store File objects well in localStorage.
     // We assume fileDataUrl is passed or attached to the task object in memory transiently.
     // If page refreshed, we depend on t.fileDataUrl (if persisted? dataURL is large).
     // Ideally we re-read from input if available, or fail if missing.

     let fileUrl = t.fileDataUrl
     if (!fileUrl && t._fileObject) {
         try {
             fileUrl = await fileToDataUrl(t._fileObject)
         } catch {
             updateTask(taskId, { status: 'failed', message: 'Read file failed' })
             return
         }
     }

     if (fileUrl) {
         const isVideo = fileUrl.startsWith('data:video') || /\.(mp4|mov|webm)$/i.test(t.fileName || '')
         contentArr.push({
             type: isVideo ? 'video_url' : 'image_url',
             [isVideo ? 'video_url' : 'image_url']: { url: fileUrl }
         })
     }

     const payload = {
         model: t.model,
         stream: true,
         messages: [ { role: 'user', content: contentArr.length ? contentArr : t.prompt } ]
     }

     // 视频任务：从数据库随机取一个状态正常、有剩余次数的 token 作为 bearer，并记下 token_id 供收到 create 结果时写入 SQLite
     let bearerForRequest = apiKey.value
     let videoTokenId = null
     if (typeof t.model === 'string' && t.model.startsWith('sora2-') && window.go?.main?.App?.GetRandomVideoToken) {
         try {
             const res = await window.go.main.App.GetRandomVideoToken()
             const data = typeof res === 'string' ? JSON.parse(res) : res
             if (data?.error) {
                 updateTask(taskId, { status: 'failed', message: data.error })
                 addLog(`Task ${taskId}: ${data.error}`, 'error')
                 return
             }
             if (data?.bearer_token) bearerForRequest = data.bearer_token
             if (data?.token_id != null) videoTokenId = data.token_id
         } catch (e) {
             updateTask(taskId, { status: 'failed', message: e?.message || '获取视频 Token 失败' })
             addLog(`Task ${taskId} 获取 Token 异常: ${e?.message}`, 'error')
             return
         }
     }

     const controller = new AbortController()
     abortControllers.set(taskId, controller)

     const isVideoTask = typeof t.model === 'string' && t.model.startsWith('sora2-')

     try {
         if (isVideoTask && window.go?.main?.App?.CreateVideo) {
             // 视频任务：调用与 testsh/create.sh 相同的接口 POST /videos（由 Go 发起，控制台打 CREATE 请求/响应）
             const orientation = t.model.includes('portrait') ? 'portrait' : 'landscape'
             const nFrames = t.model.includes('15s') ? '450' : '300'
             const apiBase = baseUrl.value.replace(/\/$/, '')
             const res = await window.go.main.App.CreateVideo(apiBase, bearerForRequest, finalPrompt, orientation, nFrames)
             const data = typeof res === 'string' ? JSON.parse(res) : res
             const taskIdRemote = data?.id || data?.task_id
             if (taskIdRemote) {
                 // 记录创建该任务时使用的账号（token_id），pending 时用同一账号
                 updateTask(taskId, {
                     remoteTaskId: taskIdRemote,
                     tokenIdForPending: videoTokenId != null ? videoTokenId : undefined,
                     message: '已提交，每 10s 轮询 pending…'
                 })
                if (videoTokenId != null && window.go?.main?.App?.SaveVideoTaskResult) {
                    try {
                        await window.go.main.App.SaveVideoTaskResult(
                            videoTokenId,
                            typeof res === 'string' ? res : JSON.stringify(data),
                            finalPrompt || ''
                        )
                     } catch (err) {
                         addLog(`保存视频任务结果失败: ${err}`, 'warning')
                     }
                 }
                 // 每 10s 轮询 pending（与 test_pending.sh 一致），返回 [] 表示任务完成
                 if (window.go?.main?.App?.PollPending) {
                     startPendingInterval(taskId)
                 }
             } else {
                 updateTask(taskId, { status: 'failed', message: data?.error || '未返回 task_id' })
             }
         } else {
             // 非视频任务（图片等）：仍走流式 /v1/chat/completions
             await streamCompletion(payload, bearerForRequest, baseUrl.value, {
                 signal: controller.signal,
                 onMessage: (msg) => {
                     if (msg.error) {
                         updateTask(taskId, { status: 'failed', message: msg.error.message || 'Error' })
                         return
                     }
                     const choice = msg.choices?.[0] || {}
                     const delta = choice.delta || {}
                     if (msg.progress) updateTask(taskId, { progress: msg.progress })
                     if (delta.wm) {
                         updateTask(taskId, {
                             wmStage: delta.wm.stage,
                             wmAttempt: delta.wm.attempt,
                             remoteTaskId: delta.wm.task_id
                         })
                     }
                     const urlCandidate = msg.url || msg.video_url?.url || msg.image_url?.url ||
                                         msg.output?.[0]?.url || choice.message?.url
                     if (urlCandidate) {
                         updateTask(taskId, { url: urlCandidate, status: 'done', progress: 100, result: JSON.stringify(msg) })
                     }
                 },
                 onFinish: () => {
                     const current = tasks.value.find(x => x.id === taskId)
                     if (current && current.status !== 'done' && current.status !== 'failed') {
                         if (current.url) updateTask(taskId, { status: 'done', progress: 100 })
                         else updateTask(taskId, { status: 'failed', message: 'Finished without URL' })
                     }
                 },
                 onError: (err) => {
                     if (err.name === 'AbortError') {
                         updateTask(taskId, { status: 'failed', message: 'Cancelled' })
                         addLog(`Task ${taskId} cancelled`, 'warning')
                     } else {
                         const humanErr = humanizeUpstreamError(err)
                         updateTask(taskId, { status: 'failed', message: humanErr.message })
                         addLog(`Task ${taskId} error: ${humanErr.message}`, humanErr.type || 'error')
                     }
                 }
             })
         }
     } catch (e) {
         const humanErr = humanizeUpstreamError(e)
         updateTask(taskId, { status: 'failed', message: humanErr.message })
         addLog(`Task ${taskId} exception: ${humanErr.message}`, humanErr.type || 'error')
     } finally {
         abortControllers.delete(taskId)
     }
  }

  const cancelTask = (taskId) => {
      const c = abortControllers.get(taskId)
      if (c) c.abort()
      const pendingId = pendingIntervals.get(taskId)
      if (pendingId != null) {
          clearInterval(pendingId)
          pendingIntervals.delete(taskId)
      }
      updateTask(taskId, { status: 'failed', message: 'Cancelled manually' })
  }

  return {
    apiKey,
    baseUrl,
    setApiKey,
    setBaseUrl,
    models,
    modelGroups,
    selectedModel,
    batchMode,
    tasks,
    attachedRoles,
    favoriteRoles,
    logs,
    addLog,
    clearLogs,
    toggleRoleFavorite,
    isRoleFavorite,
    attachRole,
    detachRole,
    addTask,
    updateTask,
    removeTask,
    clearAllTasks,
    clearLocalTasksAndReload,
    runTask,
    cancelTask,
    loadTaskList,
    startPendingForExistingTasks
  }
})
