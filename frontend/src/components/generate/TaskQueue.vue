<script setup>
import { computed, reactive } from 'vue'

const props = defineProps({
  tasks: {
    type: Array,
    default: () => [],
  },
  infoLoadingMap: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['download', 'info', 'retry', 'delete'])

const statusMap = {
  queued: { label: '排队中', class: 'status-queue' },
  running: { label: '生成中', class: 'status-running' },
  done: { label: '完成', class: 'status-done' },
  failed: { label: '失败', class: 'status-error' },
}

const getStatus = (status) => statusMap[status] || { label: status, class: '' }

// Parse result to get media URL if available
const mediaCache = reactive({})

const isVideoUrl = (url) =>
    typeof url === 'string' &&
    (url.toLowerCase().includes('.mp4') || url.startsWith('data:video/') || url.startsWith('http'))

const getMedia = (task) => {
    if (task.status !== 'done') return null
    if (task.localPath) {
        if (mediaCache[task.id] === undefined && window.go?.main?.App?.GetLocalFileURL) {
            mediaCache[task.id] = null
            window.go.main.App.GetLocalFileURL(task.localPath)
                .then((url) => { mediaCache[task.id] = url })
                .catch(() => { mediaCache[task.id] = null })
        }
        return mediaCache[task.id]
    }
    if (!task.result) return null
    try {
        const res = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        if (res.data && res.data[0] && res.data[0].url) return res.data[0].url
        if (res.url) return res.url
    } catch(e) { return null }
    return null
}

const playVideo = (e) => {
    const el = e?.currentTarget
    if (el && el.paused) el.play()
}

const progressPct = (task) => {
    const p = Number(task?.progress || 0)
    if (Number.isNaN(p)) return 0
    return Math.max(0, Math.min(100, Math.round(p)))
}

const isInfoLoading = (task) => {
    const key = task?.remoteTaskId || task?.id
    return !!(key && props.infoLoadingMap && props.infoLoadingMap[key])
}
</script>

<template>
  <div class="task-queue">
    <div v-if="tasks.length === 0" class="empty-tasks">
      <div class="empty-icon">📂</div>
      <p>暂无任务，快去创建一个吧</p>
    </div>

    <div v-else class="tasks-list">
      <div v-for="(task, index) in tasks" :key="task.id" class="task-card">
        <div class="task-header">
          <div class="task-info">
            <span class="task-id" :title="task.remoteTaskId || task.id">#{{ index + 1 }}</span>
            <span class="task-model">{{ task.model.replace('sora2-', '') }}</span>
          </div>
          <div class="task-status" :class="getStatus(task.status).class">
            {{ getStatus(task.status).label }}
          </div>
        </div>

        <div class="task-body">
          <p class="prompt" :title="task.prompt">{{ task.prompt }}</p>

          <!-- Media Preview if done -->
          <div v-if="task.status === 'done'" class="media-preview">
             <div v-if="getMedia(task)" class="media-box">
                 <video
                     v-if="isVideoUrl(getMedia(task)) || task.model.includes('sora')"
                     :src="getMedia(task)"
                     controls
                     preload="metadata"
                     @click="playVideo"
                 ></video>
                 <img v-else :src="getMedia(task)" alt="result" />
             </div>
             <div v-else class="no-preview">无预览数据</div>
          </div>

          <!-- Error Msg -->
          <div v-if="task.status === 'failed' && task.result" class="error-box">
              {{ task.result }}
          </div>
        </div>

        <div v-if="task.status === 'running'" class="progress-bar-shell">
            <div class="progress-bar" :style="{ width: progressPct(task) + '%' }"></div>
        </div>
        <div v-if="task.status === 'running'" class="progress-text">
            进度 {{ progressPct(task) }}%
        </div>

        <div class="task-footer">
            <div class="actions">
                <button v-if="task.status === 'done'" class="btn-icon" title="下载" @click="emit('download', task)">
                    ⬇
                </button>
                <button
                    class="btn-icon btn-info"
                    title="获取无水印视频"
                    @click="emit('info', task)"
                    :disabled="isInfoLoading(task)"
                >
                    {{ isInfoLoading(task) ? '处理中' : '无水印' }}
                </button>
                <button v-if="task.status === 'failed'" class="btn-icon" title="重试" @click="emit('retry', task)">
                    ↻
                </button>
                <button class="btn-icon danger" title="删除" @click="emit('delete', task)">
                    ✕
                </button>
            </div>
            <span class="time">{{ new Date(task.timestamp).toLocaleTimeString() }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.task-queue {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-tasks {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  border: 1px dashed rgba(148, 163, 184, 0.3);
  border-radius: 12px;
  color: #94a3b8;
}
.empty-icon { font-size: 32px; margin-bottom: 10px; opacity: 0.5; }

.task-card {
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: all 0.2s;
}
.task-card:hover {
  background: rgba(30, 41, 59, 0.6);
  border-color: rgba(56, 189, 248, 0.3);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.task-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.task-id {
  font-family: monospace;
  font-size: 11px;
  color: #64748b;
  background: rgba(15, 23, 42, 0.3);
  padding: 2px 6px;
  border-radius: 4px;
}
.task-model {
  font-size: 11px;
  font-weight: 600;
  color: #60a5fa;
  background: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.task-status {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.status-queue { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
.status-running { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
.status-done { background: rgba(34, 197, 94, 0.15); color: #4ade80; }
.status-error { background: rgba(239, 68, 68, 0.15); color: #f87171; }

.task-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.prompt {
  margin: 0;
  font-size: 13px;
  color: #cbd5e1;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.media-preview {
    margin-top: 4px;
    border-radius: 8px;
    overflow: hidden;
    background: #000;
    max-height: 180px;
}
.media-box, .media-box video, .media-box img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
}

.error-box {
    font-size: 11px;
    color: #f87171;
    background: rgba(239, 68, 68, 0.1);
    padding: 8px;
    border-radius: 6px;
    word-break: break-all;
}

.progress-bar-shell {
    height: 3px;
    background: rgba(148, 163, 184, 0.2);
    border-radius: 2px;
    overflow: hidden;
}
.progress-bar {
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #6366f1);
}

.progress-text {
    font-size: 11px;
    color: #94a3b8;
}

.task-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 4px;
}
.time {
    font-size: 11px;
    color: #64748b;
}

.actions {
    display: flex;
    gap: 6px;
}
.btn-icon {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #94a3b8;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    transition: all 0.2s;
}
.btn-icon.btn-info {
    width: auto;
    padding: 0 8px;
    font-size: 11px;
}
.btn-icon:disabled {
    cursor: not-allowed;
    opacity: 0.6;
}
.btn-icon:hover {
    background: rgba(255,255,255,0.1);
    color: #f1f5f9;
}
.btn-icon.danger:hover {
    background: rgba(248, 113, 113, 0.2);
    color: #f87171;
    border-color: rgba(248, 113, 113, 0.4);
}
</style>
