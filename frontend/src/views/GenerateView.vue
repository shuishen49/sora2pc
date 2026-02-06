<script setup>
import { ref, reactive, computed } from 'vue'
import RoleSelector from '../components/generate/RoleSelector.vue'
import AdvancedSettings from '../components/generate/AdvancedSettings.vue'
import FileUploader from '../components/generate/FileUploader.vue'
import BatchModeSelector from '../components/generate/BatchModeSelector.vue'
import MultiPromptEditor from '../components/generate/MultiPromptEditor.vue'
import StoryboardEditor from '../components/generate/StoryboardEditor.vue'
import TaskQueue from '../components/generate/TaskQueue.vue'
import PreviewGallery from '../components/generate/PreviewGallery.vue'
import PreviewModal from '../components/generate/PreviewModal.vue'
import TaskLog from '../components/generate/TaskLog.vue'
import ToastNotification from '../components/common/ToastNotification.vue'
// RoleSelector is not used in the provided snippet's template or script logic.
// AdvancedSettings is not used in the provided snippet's template or script logic.
// FileUploader is not used in the provided snippet's template or script logic.

// TaskQueue is not used in the provided snippet's template or script logic.
// PreviewGallery is not used in the provided snippet's template or script logic.
// PreviewModal is not used in the provided snippet's template or script logic.
// TaskLog is not used in the provided snippet's template or script logic.
// BatchModeSelector is not used in the provided snippet's template or script logic.
// MultiPromptEditor is not used in the provided snippet's template or script logic.
// StoryboardEditor is not used in the provided snippet's template or script logic.
// ToastNotification is not used in the provided snippet's template or script logic.

const onFileSelected = ({ url, type }) => {
    // Basic recommendation logic
    if (type === 'image') {
        const img = new Image()
        img.onload = () => {
            const ratio = img.width / img.height
            // If currently selected model is text-to-video (no image input support usually, but here we assume img2video support)
            // If user uploads vertical image but has landscape model selected:
            const isLandscapeModel = store.selectedModel.includes('landscape')
            // const isPortraitModel = store.selectedModel.includes('portrait') // This variable is unused.

            if (ratio < 0.8 && isLandscapeModel) {
                 // Suggest Portrait
                 // We could auto-switch or just show toast. For now, auto-switch to safe default.
                 // store.selectedModel = 'sora2-portrait-10s'
                 // Or just log it. Let's keep it simple for now as per requirements "Recommend model switching"
            }
        }
        img.src = url
    }
}

import { useGenerateStore } from '../stores/generate'
import { useAdminStore } from '../stores/admin'

const store = useGenerateStore()
const adminStore = useAdminStore()
const infoLoadingMap = reactive({})

// Computed from store
const tasks = computed(() => store.tasks)
const attachedRoles = computed(() => store.attachedRoles)

// Form state (synced with store where needed)
const form = reactive({
  model: store.selectedModel,
  prompt: '',
  file: null,
  fileUrl: null,
  fileType: null,
  batchEnabled: false,
  batchCount: 5,
  proxyUrl: '',
  timeout: 300,
  debug: false
})

// Draft Auto-save
import { watch } from 'vue'

const DRAFT_KEY = 'gen_prompt_draft_v1'
const savedDraft = localStorage.getItem(DRAFT_KEY)
if (savedDraft) {
    form.prompt = savedDraft
}

// 记住上一次使用的模型，避免每次都重新选择
const MODEL_KEY = 'gen_last_model_v1'
const savedModel = localStorage.getItem(MODEL_KEY)
if (savedModel) {
    form.model = savedModel
}

watch(() => form.prompt, (newVal) => {
    localStorage.setItem(DRAFT_KEY, newVal || '')
})

watch(() => form.model, (newVal) => {
    if (newVal) {
        localStorage.setItem(MODEL_KEY, newVal)
    }
})

// Computed for template
const apiKey = computed({
  get: () => store.apiKey,
  set: (val) => store.setApiKey(val)
})
const baseUrl = computed({
  get: () => store.baseUrl,
  set: (val) => store.setBaseUrl(val)
})
const batchMode = computed({
  get: () => store.batchMode,
  set: (val) => { store.batchMode = val }
})

const advSettings = reactive({
  batchEnabled: false,
  batchCount: 5,
  proxyUrl: '',
  timeout: 300,
  debug: false
})

// Multi-prompt mode state
const multiPromptRows = ref([])

// Storyboard mode state
const storyboardShots = ref([])
const storyboardTitle = ref('')
const storyboardContext = ref('')

const generating = ref(false)

// Right Sidebar Tabs
const currentTab = ref('tasks') // tasks, preview, log
const showPreviewModal = ref(false)
const previewTask = ref(null)

const openPreview = (task) => {
    previewTask.value = task
    showPreviewModal.value = true
}

// Sync Advanced Settings
const updateAdvSettings = (val) => {
    Object.assign(form, val)
}

// ======= 模型选择拆分：版本 / 时长 / 横竖屏 =======
// 第一列：版本（标准 / Pro / Pro HD）
const versionOptions = [
  { value: 'standard', label: '标准版视频' },
  { value: 'pro', label: 'Pro版视频' },
  { value: 'pro-hd', label: 'Pro HD版视频' }
]

// 第二列：时长（HD 只有 10s / 15s，其他都有 25s）
const durationOptionsAll = [
  { value: '25s', label: '25s' },
  { value: '15s', label: '15s' },
  { value: '10s', label: '10s' }
]

const availableDurationOptions = computed(() => {
  // Pro HD 版不支持 25s，不展示
  if (selectedVersion.value === 'pro-hd') {
    return durationOptionsAll.filter(opt => opt.value !== '25s')
  }
  return durationOptionsAll
})

// 第三列：横竖屏
const orientationOptions = [
  { value: 'landscape', label: '横屏' },
  { value: 'portrait', label: '竖屏' }
]

const selectedVersion = ref('standard')
const selectedDuration = ref('10s')
const selectedOrientation = ref('landscape')

// 从 model 字符串解析出三要素
const parseModel = (value) => {
  if (!value || typeof value !== 'string') return null

  let version = 'standard'
  if (value.startsWith('sora2pro-hd')) {
    version = 'pro-hd'
  } else if (value.startsWith('sora2pro')) {
    version = 'pro'
  }

  let orientation = value.includes('-portrait-') ? 'portrait' : 'landscape'

  let duration = '10s'
  if (value.endsWith('25s')) duration = '25s'
  else if (value.endsWith('15s')) duration = '15s'
  else if (value.endsWith('10s')) duration = '10s'

  return { version, orientation, duration }
}

// 根据三要素组装 model 字符串
const buildModelFromParts = () => {
  let prefix = 'sora2'
  if (selectedVersion.value === 'pro') prefix = 'sora2pro'
  else if (selectedVersion.value === 'pro-hd') prefix = 'sora2pro-hd'

  return `${prefix}-${selectedOrientation.value}-${selectedDuration.value}`
}

// 将 form.model 拆分同步到三列
const syncPartsFromModel = (val) => {
  const parsed = parseModel(val)
  if (!parsed) return

  selectedVersion.value = parsed.version
  selectedOrientation.value = parsed.orientation

  // HD 版如果从外部传来 25s，强制回退为 15s
  if (parsed.version === 'pro-hd' && parsed.duration === '25s') {
    selectedDuration.value = '15s'
  } else {
    selectedDuration.value = parsed.duration
  }
}

// 将三列组合回 form.model
const syncModelFromParts = () => {
  form.model = buildModelFromParts()
}

// 初始化 & 双向同步
watch(
  () => form.model,
  (val) => {
    syncPartsFromModel(val)
  },
  { immediate: true }
)

watch(selectedVersion, (val) => {
  // 切到 Pro HD 时，如果当前是 25s，自动改为 15s
  if (val === 'pro-hd' && selectedDuration.value === '25s') {
    selectedDuration.value = '15s'
  }
  syncModelFromParts()
})

watch([selectedDuration, selectedOrientation], () => {
  syncModelFromParts()
})




const appendRolePrompt = (rolePrompt) => {
    if (!rolePrompt) return
    const current = form.prompt.trim()
    const extra = rolePrompt.trim()
    if (!current) {
        form.prompt = extra
    } else if (!current.includes(extra)) {
        // Avoid duplicate roughly
        form.prompt = current + (current.endsWith(',') || current.endsWith('，') ? ' ' : ', ') + extra
    }
    toast.success('已添加角色提示词')
}

// Shortcuts
import { onMounted, onUnmounted } from 'vue'
import { useToast } from '../composables/useToast'
const toast = useToast()

const handleKeydown = (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        handleGenerate()
    }
}

onMounted(async () => {
    window.addEventListener('keydown', handleKeydown)
    // 任务列表从 SQLite 加载（Wails 下）
    if (store.loadTaskList) await store.loadTaskList()
    // 为已有未完成任务恢复 pending 轮询
    if (store.startPendingForExistingTasks) store.startPendingForExistingTasks()
})

onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
})


// Generate Action
const handleGenerate = async () => {
  if (batchMode.value === 'multi_prompt') {
      // Create a task for each prompt row
      if (!multiPromptRows.value.length) return alert('请添加至少一条提示词')
      generating.value = true

      for (const row of multiPromptRows.value) {
          if (!row.prompt) continue
          const taskId = Date.now() + Math.random()
          const newTask = {
              id: taskId,
              model: form.model,
              prompt: row.prompt,
              status: 'queued',
              timestamp: Date.now(),
              _fileObject: row.file || null // Support per-row file
          }
          store.addTask(newTask)
          store.runTask(taskId)
      }
      generating.value = false
      return
  }

  if (batchMode.value === 'storyboard') {
      // Simplify storyboard logic for now: create sequential tasks
       if (!storyboardShots.value.length) return alert('请添加分镜')
       generating.value = true

       for (const [index, shot] of storyboardShots.value.entries()) {
           const taskId = Date.now() + index
           const newTask = {
              id: taskId,
              model: form.model,
              prompt: shot.prompt || form.prompt,
              status: 'queued',
              timestamp: Date.now(),
              tag: 'storyboard',
              storyboard: { title: storyboardTitle.value, idx: index + 1, label: `分镜${index+1}` }
           }
           store.addTask(newTask)
           store.runTask(taskId)
       }
       generating.value = false
       return
  }

  // Single / Same Prompt Mode
  if (!form.prompt.trim() && !form.file) return alert('请输入提示词或上传参考图/视频')
  if (generating.value) return

  generating.value = true

  // Logic for batch or single
  const count = form.batchEnabled ? form.batchCount : 1

  for (let i = 0; i < count; i++) {
      const taskId = Date.now() + i
      const newTask = {
          id: taskId,
          model: form.model,
          prompt: form.prompt,
          status: 'queued',
          timestamp: Date.now(),
          result: null,
          _fileObject: form.file // Pass file object to store
      }
      store.addTask(newTask)

      // Run Async via Store
      store.runTask(taskId)

      // Small delay between batch requests
      if(count > 1) await new Promise(r => setTimeout(r, 200))
  }

  generating.value = false
}

// Task Actions
import { buildDownloadFilename } from '../utils/fileUtils'

const downloadTask = async (task) => {
    if (!task) return
    const taskId = task.remoteTaskId || task.id
    if (window.go?.main?.App?.ReDownloadVideo && taskId) {
        try {
            const res = await window.go.main.App.ReDownloadVideo(String(taskId))
            const data = typeof res === 'string' ? JSON.parse(res) : res
            if (data?.success) {
                const link = data.downloadable_url || ''
                if (link && navigator.clipboard?.writeText) {
                    await navigator.clipboard.writeText(link)
                }
                toast.success('下载成功，链接已复制到剪贴板，可手动下载。')
                return
            }
            toast.error(data?.message || '下载失败')
            return
        } catch (e) {
            toast.error(`下载失败: ${e?.message || e}`)
            return
        }
    }

    if (!task.url) return toast.warning('任务未完成或无下载链接')
    const a = document.createElement('a')
    a.href = task.url
    const isImage = task.url.includes('.png') || task.url.includes('.jpg')
    const name = buildDownloadFilename(task, task.url, isImage ? 'image' : 'video')
    a.download = name
    a.target = '_blank'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
}

const handleInfo = async (task) => {
    if (!task) return
    const taskId = task.remoteTaskId || task.id
    if (!taskId) return
    const apiBase = (store.baseUrl?.value ?? store.baseUrl) || ''
    if (infoLoadingMap[taskId]) return
    infoLoadingMap[taskId] = true

    const currentSettings = (adminStore.settings?.value ?? adminStore.settings)
    if (!currentSettings || !currentSettings.watermarkCustomToken) {
        try {
            await adminStore.loadSettings()
        } catch (_) {}
    }
    const wm = (adminStore.settings?.value ?? adminStore.settings) || {}
    const parseUrl = wm.watermarkCustomUrl || ''
    const parseToken = wm.watermarkCustomToken || ''
    if (!parseToken) {
        toast.warning('请先在设置里填写无水印解析 Token')
        infoLoadingMap[taskId] = false
        return
    }

    if (!window.go?.main?.App?.PublishAndDownloadNoWatermark) {
        toast.error('后端方法不可用')
        infoLoadingMap[taskId] = false
        return
    }
    try {
        const res = await window.go.main.App.PublishAndDownloadNoWatermark(String(apiBase), String(taskId), String(parseUrl || ''), String(parseToken))
        const data = typeof res === 'string' ? JSON.parse(res) : res
        if (data?.success) {
            if (data?.no_watermark && navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(data.no_watermark)
            }
            toast.success('无水印视频已重新下载，链接已复制到剪贴板')
            if (store.loadTaskList) await store.loadTaskList()
            infoLoadingMap[taskId] = false
            return
        }
        toast.error(data?.message || '处理失败')
    } catch (e) {
        toast.error(`处理失败: ${e?.message || e}`)
    } finally {
        infoLoadingMap[taskId] = false
    }
}

const deleteModalOpen = ref(false)
const deleteTarget = ref(null)

const deleteTask = async (task) => {
    if (!task) return

    // 失败任务：直接删除记录，不弹确认框
    if (task.status === 'failed') {
        const taskId = task.remoteTaskId || task.id
        if (window.go?.main?.App?.DeleteTaskData && taskId) {
            try {
                // 失败任务一般没有本地文件，这里只删除数据记录
                await window.go.main.App.DeleteTaskData(String(taskId), false)
            } catch (_) {
                // 删除失败忽略，依然从前端列表移除，避免列表卡死
            }
        }
        store.removeTask(task.id)
        return
    }

    // 其他状态：仍然弹出确认弹窗，让用户选择删除方式
    deleteTarget.value = task
    deleteModalOpen.value = true
}

const closeDeleteModal = () => {
    deleteModalOpen.value = false
    deleteTarget.value = null
}

const confirmDelete = async (mode) => {
    const t = deleteTarget.value
    if (!t) {
        closeDeleteModal()
        return
    }
    const taskId = t.remoteTaskId || t.id
    if (window.go?.main?.App?.DeleteTaskData && taskId) {
        try {
            await window.go.main.App.DeleteTaskData(String(taskId), mode === 'both')
        } catch (_) {
            // ignore
        }
    }
    store.removeTask(t.id)
    closeDeleteModal()
}

const retryTask = (task) => {
    store.runTask(task.id)
}
</script>

<template>
  <div class="generate-page">
    <div class="layout-grid">

      <!-- LEFT: Roles -->
      <aside class="sidebar-left">
        <div class="panel-box full-height">
           <RoleSelector v-model="attachedRoles" @append-prompt="appendRolePrompt" />
        </div>
      </aside>

      <!-- CENTER: Main Input -->
      <main class="center-stage">
         <div class="panel-box input-panel">
            <!-- Config Row -->
            <div class="config-row">
                <div class="config-item">
                    <label>API Key</label>
                    <input
                        type="password"
                        v-model="apiKey"
                        placeholder="API Key"
                        class="config-input"
                    />
                </div>
                <div class="config-item flex-grow">
                    <label>服务器地址</label>
                    <input
                        type="text"
                        v-model="baseUrl"
                        placeholder="http://127.0.0.1:8000"
                        class="config-input"
                    />
                </div>
            </div>

            <!-- Batch Mode Selector -->
            <BatchModeSelector v-model="batchMode" />

            <div class="panel-header">
                <h2>创建生成任务</h2>
                <!-- 一排三列：版本 / 时长 / 横竖屏 -->
                <div class="model-select-row">
                    <div class="model-select-column">
                        <label class="model-group-title">版本</label>
                        <select v-model="selectedVersion" class="config-input">
                            <option
                              v-for="opt in versionOptions"
                              :key="opt.value"
                              :value="opt.value"
                            >
                                {{ opt.label }}
                            </option>
                        </select>
                    </div>
                    <div class="model-select-column">
                        <label class="model-group-title">时长</label>
                        <select v-model="selectedDuration" class="config-input">
                            <option
                              v-for="opt in availableDurationOptions"
                              :key="opt.value"
                              :value="opt.value"
                            >
                                {{ opt.label }}
                            </option>
                        </select>
                    </div>
                    <div class="model-select-column">
                        <label class="model-group-title">画幅</label>
                        <select v-model="selectedOrientation" class="config-input">
                            <option
                              v-for="opt in orientationOptions"
                              :key="opt.value"
                              :value="opt.value"
                            >
                                {{ opt.label }}
                            </option>
                        </select>
                    </div>
                </div>
            </div>

            <!-- Attached Global Roles Chips -->
            <div class="attached-roles" v-if="attachedRoles.length && ['single', 'same_prompt_files'].includes(batchMode)">
                <div v-for="role in attachedRoles" :key="role.name" class="role-chip" :title="role.prompt">
                    <span>📌 {{ role.name }}</span>
                    <span class="remove" @click="store.detachRole(role.name)">✕</span>
                </div>
            </div>

            <!-- Prompt Area -->
            <div class="prompt-area">
                <textarea
                    v-model="form.prompt"
                    placeholder="描述你的创意... (支持中英文，建议英文)"
                    class="prompt-input"
                    rows="6"
                ></textarea>
                <div class="prompt-tools">
                    <span class="tool-btn" @click="form.prompt = ''" title="清空">🗑️</span>
                    <span class="tool-btn" title="复制">📋</span>
                </div>
            </div>

            <!-- Mode-specific Editors -->
            <!-- Single & Same Prompt: Use basic prompt area above -->

            <!-- Multi-Prompt Mode -->
            <MultiPromptEditor
              v-if="batchMode === 'multi_prompt'"
              v-model="multiPromptRows"
              :globalCount="advSettings.batchCount"
            />

            <!-- Storyboard Mode -->
            <StoryboardEditor
              v-if="batchMode === 'storyboard'"
              v-model="storyboardShots"
              v-model:title="storyboardTitle"
              v-model:context="storyboardContext"
            />

            <!-- Upload Area -->
            <div class="upload-area" v-if="batchMode === 'single' || batchMode === 'same_prompt_files'">
                <label style="display:block; margin-bottom: 8px; font-size: 14px; font-weight: 500; color: #e2e8f0;">参考图 / 视频</label>
                <FileUploader v-model="form.file" @file-selected="onFileSelected" />
            </div>

            <!-- Advanced Settings -->
            <AdvancedSettings :modelValue="advSettings" @update:modelValue="updateAdvSettings" />

            <!-- Preview Modal -->
            <PreviewModal
                :show="showPreviewModal"
                :task="previewTask"
                @close="showPreviewModal = false"
                @download="downloadTask"
            />

            <!-- Action Bar -->
            <div class="action-bar">
                <button
                    class="generate-btn"
                    :class="{ 'btn-ready': !generating, 'btn-loading': generating }"
                    :disabled="generating"
                    @click="handleGenerate"
                >
                    <span v-if="generating" class="spinner"></span>
                    {{ generating ? '生成中...' : '立即生成 (Generate)' }}
                </button>
            </div>
         </div>
      </main>

      <!-- RIGHT: Tabs & Panels -->
      <aside class="sidebar-right">
          <div class="panel-box full-height">
              <div class="panel-header-tabs">
                  <div class="tab" :class="{ active: currentTab === 'tasks' }" @click="currentTab = 'tasks'">
                      任务 <span class="badge" v-if="tasks.length">{{ tasks.length }}</span>
                  </div>
                  <div class="tab" :class="{ active: currentTab === 'preview' }" @click="currentTab = 'preview'">
                      预览
                  </div>
                  <div class="tab" :class="{ active: currentTab === 'log' }" @click="currentTab = 'log'">
                      日志
                  </div>
              </div>

              <div class="panel-content">
                  <div v-show="currentTab === 'tasks'">
                      <div class="task-list-toolbar">
                          <button type="button" class="btn-refresh" @click="store.loadTaskList()" title="从 SQLite 重新读取任务列表">
                              🔄 从 SQLite 刷新
                          </button>
                      </div>
                      <TaskQueue
                          :tasks="tasks"
                          :info-loading-map="infoLoadingMap"
                          @download="downloadTask"
                          @info="handleInfo"
                          @delete="deleteTask"
                          @retry="retryTask"
                      />
                  </div>

                  <div v-show="currentTab === 'preview'">
                      <PreviewGallery
                          :tasks="tasks"
                          @download="downloadTask"
                          @open-preview="openPreview"
                      />
                  </div>

                  <div v-show="currentTab === 'log'">
                      <TaskLog :logs="store.logs" @clear="store.clearLogs" />
                  </div>
              </div>
          </div>
      </aside>

      <!-- Delete Task Modal -->
      <div v-if="deleteModalOpen" class="modal-mask">
          <div class="modal-card">
              <h3>确定删除该任务吗？</h3>
              <p class="modal-desc">请选择删除方式：</p>
              <div class="modal-actions">
                  <button class="btn-danger" @click="confirmDelete('both')">删除文件 + 数据</button>
                  <button class="btn-warn" @click="confirmDelete('data')">只删除数据</button>
                  <button class="btn-ghost" @click="closeDeleteModal">关闭</button>
              </div>
          </div>
      </div>

      <!-- Toast Container -->
      <ToastNotification />
  </div>
  </div>
</template>

<style scoped>
.generate-page {
  /* Fix width overflow: use 100% of parent (.content-body) instead of viewport width */
  width: 100%;

  /* Height: Fill the parent completely */
  height: 100%;

  overflow: hidden;
  padding: 16px; /* Checkers the spacing here instead of parent */
  box-sizing: border-box;
}

.layout-grid {
  display: grid;
  /* Left Sidebar (280px) | Main Content (Flex) | Right Sidebar (320px) */
  grid-template-columns: 280px minmax(0, 1fr) 320px;
  gap: 16px;
  height: 100% !important;
  width: 100%;
}

/* Force grid items to constrain height and not overflow */
.sidebar-left,
.sidebar-right {
  overflow: hidden;
  height: 100%;
}

.center-stage {
  overflow-x: hidden; /* Prevent horizontal scroll */
  overflow-y: hidden;
  height: 100%;
  min-width: 0; /* Prevent grid blowout */
}

.full-height {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden; /* Ensure matched parent */
}

.panel-box {
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 16px;
  padding: 16px;
  backdrop-filter: blur(12px);
  position: relative;
  /* Use block layout for natural flow, not flex */
  height: 100%;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  overflow: hidden; /* Children will handle scrolling */
}

.panel-content {
    flex: 1;
    overflow-y: auto; /* Enable scrolling for panel content */
    position: relative;
    padding-right: 4px; /* Prevent scrollbar overlapping content */
}

/* Custom Scrollbar for panels */
.panel-content::-webkit-scrollbar,
.custom-scroll::-webkit-scrollbar {
  width: 6px;
}
.panel-content::-webkit-scrollbar-track,
.custom-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.panel-content::-webkit-scrollbar-thumb,
.custom-scroll::-webkit-scrollbar-thumb {
  background-color: rgba(148, 163, 184, 0.2);
  border-radius: 3px;
}
.panel-content::-webkit-scrollbar-thumb:hover,
.custom-scroll::-webkit-scrollbar-thumb:hover {
  background-color: rgba(148, 163, 184, 0.4);
}

/* Header */
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0; /* Prevent header squishing */
}

.panel-header h2 { margin: 0; font-size: 18px; color: #f1f5f9; }

/* Model Select - 新的三列布局 */
.model-select-row {
    display: flex;
    gap: 12px;
    margin-left: 16px;
    flex: 1;
}

.model-select-column {
    flex: 1 1 0;
    min-width: 0;
}

.model-group-title {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    color: #94a3b8;
}

/* 旧的下拉样式仍用于其它地方，这里保留 */
.model-select-wrapper {
    position: relative;
    width: 200px;
}

.custom-select {
    position: relative;
    width: 100%;
}

.select-trigger {
    background: #0f172a;
    color: #f1f5f9;
    border: 1px solid rgba(148, 163, 184, 0.2);
    padding: 6px 10px;
    border-radius: 8px;
    font-size: 13px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
}

.select-trigger:hover {
    border-color: #3b82f6;
}

.select-trigger .arrow {
    font-size: 10px;
    color: #94a3b8;
    transition: transform 0.2s;
}

.custom-select.open .select-trigger .arrow {
    transform: rotate(180deg);
}

.select-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    width: 100%;
    background: #0f172a;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 8px;
    margin-top: 4px;
    z-index: 100;
    max-height: 400px; /* Limit height */
    overflow-y: auto;
    box-shadow: 0 10px 25px rgba(0,0,0,0.5);
    padding: 4px 0;
}

/* Scrollbar for dropdown */
.select-dropdown::-webkit-scrollbar { width: 4px; }
.select-dropdown::-webkit-scrollbar-thumb { background: rgba(148,163,184,0.3); border-radius: 4px; }
.select-dropdown::-webkit-scrollbar-track { background: transparent; }

.select-group {
    border-bottom: 1px solid rgba(148,163,184,0.1);
    padding-bottom: 4px;
    margin-bottom: 4px;
}
.select-group:last-child {
    border-bottom: none;
    padding-bottom: 0;
    margin-bottom: 0;
}

.group-label {
    font-size: 11px;
    font-weight: 700;
    color: #64748b;
    padding: 4px 8px;
    text-transform: uppercase;
}

.select-option {
    padding: 6px 12px;
    font-size: 13px;
    color: #cbd5e1;
    cursor: pointer;
    transition: background 0.15s;
}

.select-option:hover {
    background: rgba(59, 130, 246, 0.15);
    color: #fff;
}

.select-option.active {
    background: linear-gradient(135deg, #2563eb, #4f46e5);
    color: #fff;
}

/* Custom fade transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-5px);
}

/* Config Row */
.config-row {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  width: 100%;
  box-sizing: border-box;
}
.config-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0; /* Prevent overflow */
}
.config-item.flex-grow {
  flex: 1;
}
.config-item label {
  font-size: 11px;
  color: #94a3b8;
  font-weight: 600;
}
.config-input {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 8px;
  padding: 8px 12px;
  color: #e2e8f0;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}
.config-input:focus {
  border-color: #3b82f6;
}

/* Prompt */
.prompt-area {
  position: relative;
  flex-shrink: 0;
}
.prompt-input {
  width: 100%;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  padding: 16px;
  color: #e2e8f0;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
  /* Important: ensure padding doesn't overflow width */
}
.prompt-input:focus {
  border-color: #3b82f6;
  background: rgba(15, 23, 42, 0.8);
}
.prompt-tools {
  position: absolute;
  bottom: 12px;
  right: 12px;
  display: flex;
  gap: 8px;
}
.tool-btn {
  cursor: pointer;
  opacity: 0.5;
  transition: opacity 0.2s;
  font-size: 14px;
}
.tool-btn:hover { opacity: 1; }

/* Upload */
.upload-area {
  min-height: 100px;
}
.dropzone {
  border: 2px dashed rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  height: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  color: #94a3b8;
}
.dropzone:hover {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.05);
  color: #3b82f6;
}
.dropzone .icon { font-size: 24px; margin-bottom: 4px; }
.dropzone .text { font-size: 12px; }

.file-preview {
    display: flex;
    align-items: center;
    gap: 12px;
    background: rgba(15, 23, 42, 0.4);
    padding: 8px;
    border-radius: 10px;
    border: 1px solid rgba(148,163,184,0.1);
}
.preview-media {
    width: 60px;
    height: 60px;
    border-radius: 6px;
    overflow: hidden;
    background: #000;
}
.preview-media img, .preview-media video {
    width: 100%;
    height: 100%;
    object-fit: cover;
}
.file-info {
    flex: 1;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.name { font-size: 13px; color: #cbd5e1; }
.remove-btn { color: #f87171; cursor: pointer; padding: 4px; }

/* Action Bar */
.action-bar {
  margin-top: auto;
  flex-shrink: 0; /* Always visible at bottom */
  padding-top: 16px;
}

/* Input Panel Specific - Use block layout for natural document flow */
.input-panel {
    overflow-y: auto; /* Allow scroll when content exceeds height */
    overflow-x: hidden;
    height: 100%;
    display: block; /* Normal document flow, not flex */
}

/* Add spacing between children since we removed flex gap */
.input-panel > * {
    margin-bottom: 16px;
}
.input-panel > *:last-child {
    margin-bottom: 0;
}

.prompt-area {
  position: relative;
}

.prompt-input {
  width: 100%;
  min-height: 150px;
  resize: none;
  box-sizing: border-box;
}

.prompt-input {
  flex: 1; /* Input fills prompt area */
  min-height: 120px;
  resize: none;
}

.generate-btn {
  width: 100%;
  height: 48px;
  border-radius: 12px;
  border: none;
  font-size: 15px;
  font-weight: 700;
  color: white;
  cursor: pointer;
  background: linear-gradient(135deg, #2563eb, #4f46e5); /* Standard Blue */
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  transition: all 0.2s;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
}

/* Premium Button Animation */
@keyframes btn-ready-pulse {
  0% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.7); }
  70% { box-shadow: 0 0 0 10px rgba(59, 130, 246, 0); }
  100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0); }
}

.btn-ready:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 10px 25px rgba(37, 99, 235, 0.5);
  background: linear-gradient(135deg, #3b82f6, #6366f1);
}

.btn-loading {
    opacity: 0.8;
    cursor: wait;
}

.spinner {
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.8s infinite linear;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Sidebar Right */
.panel-header-sm {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}
.panel-header-sm h3 { font-size: 14px; margin: 0; color: #cbd5e1; }
.count { background: #334155; padding: 2px 8px; border-radius: 10px; font-size: 11px; }

.queue-container {
    flex: 1;
    overflow-y: auto;
    padding-right: 4px;
}
.queue-container::-webkit-scrollbar { width: 4px; }
.queue-container::-webkit-scrollbar-thumb { background: #334155; border-radius: 4px; }

/* Responsive */
@media (max-width: 1200px) {
  .layout-grid {
    grid-template-columns: 240px minmax(0, 1fr) 260px;
    gap: 12px;
  }
}

@media (max-width: 900px) {
  .generate-page {
    height: auto;
    overflow-y: visible; /* Allow page scroll on mobile */
  }

  .layout-grid {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    display: flex;
    flex-direction: column;
    height: auto;
  }

  .sidebar-left, .sidebar-right {
      height: 400px;
      flex-shrink: 0;
  }

  .center-stage {
      min-height: 600px;
      flex-shrink: 0;
  }
}

/* Tabs in Right Panel */
.panel-header-tabs {
    display: flex;
    background: rgba(15, 23, 42, 0.4);
    border-bottom: 1px solid rgba(148, 163, 184, 0.1);
    padding: 0 12px;
}

.tab {
    padding: 12px 16px;
    font-size: 13px;
    color: #94a3b8;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 6px;
}

.tab:hover {
    color: #cbd5e1;
}

.tab.active {
    color: #3b82f6;
    border-bottom-color: #3b82f6;
    font-weight: 600;
}

.tab .badge {
    background: rgba(59, 130, 246, 0.2);
    color: #60a5fa;
    padding: 1px 5px;
    border-radius: 4px;
    font-size: 10px;
}

.task-list-toolbar {
    margin-bottom: 10px;
}

.btn-refresh {
    font-size: 12px;
    padding: 6px 12px;
    color: #94a3b8;
    background: rgba(30, 41, 59, 0.8);
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 8px;
    cursor: pointer;
}

.btn-refresh:hover {
    color: #cbd5e1;
    border-color: rgba(148, 163, 184, 0.4);
}

.modal-mask {
    position: fixed;
    inset: 0;
    background: rgba(2, 6, 23, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
}

.modal-card {
    width: 360px;
    background: #0f172a;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    padding: 16px;
    color: #e2e8f0;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}

.modal-card h3 {
    margin: 0 0 6px 0;
    font-size: 16px;
}

.modal-desc {
    margin: 0 0 12px 0;
    font-size: 12px;
    color: #94a3b8;
}

.modal-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
}

.btn-danger,
.btn-warn,
.btn-ghost {
    padding: 6px 10px;
    font-size: 12px;
    border-radius: 8px;
    cursor: pointer;
    border: 1px solid transparent;
}

.btn-danger {
    background: rgba(248, 113, 113, 0.15);
    color: #f87171;
    border-color: rgba(248, 113, 113, 0.35);
}

.btn-warn {
    background: rgba(245, 158, 11, 0.15);
    color: #fbbf24;
    border-color: rgba(245, 158, 11, 0.35);
}

.btn-ghost {
    background: rgba(255, 255, 255, 0.05);
    color: #cbd5e1;
    border-color: rgba(148, 163, 184, 0.25);
}

.panel-content {
    flex: 1;
    min-height: 0; /* Critical for flex scroll to work */
    overflow-y: auto; /* Enable vertical scrolling */
    position: relative;
    padding: 12px;
}

.full-height {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.empty-state {
    align-items: center;
    justify-content: center;
    font-size: 13px;
}

.attached-roles {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 8px;
    padding: 0 4px;
}

.role-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    background: rgba(59, 130, 246, 0.15);
    border: 1px solid rgba(59, 130, 246, 0.3);
    border-radius: 6px;
    color: #93c5fd;
    font-size: 11px;
    cursor: default;
    transition: all 0.2s;
}

.role-chip:hover {
    background: rgba(59, 130, 246, 0.25);
}

.role-chip .remove {
    cursor: pointer;
    font-size: 10px;
    opacity: 0.6;
    margin-left: 2px;
}

.role-chip .remove:hover {
    opacity: 1;
    color: #f87171;
}
</style>
