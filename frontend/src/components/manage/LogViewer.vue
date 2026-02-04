<script setup>
import { onMounted } from 'vue'
import { useAdminStore } from '../../stores/admin'
import { storeToRefs } from 'pinia'

const adminStore = useAdminStore()
const { logs, loadingLogs } = storeToRefs(adminStore)

onMounted(() => {
  adminStore.loadLogs()
})

const refresh = () => adminStore.loadLogs()

const handleClearLogs = async () => {
    if (!confirm('确定要清空所有日志吗？')) return
    try {
        await adminStore.clearAllLogs()
    } catch (e) {
        alert('清空失败: ' + e)
    }
}
</script>

<template>
  <div class="log-viewer">
    <div class="toolbar">
      <h3>最近请求日志</h3>
      <div class="actions">
          <button class="btn-secondary" @click="handleClearLogs" :disabled="loadingLogs">
            🗑 清空日志
          </button>
          <button class="btn-secondary" @click="refresh" :disabled="loadingLogs">
            {{ loadingLogs ? '加载中...' : '刷新日志' }}
          </button>
      </div>
    </div>

    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>时间</th>
            <th>操作</th>
            <th>路径</th>
            <th>状态</th>
            <th>耗时 (ms)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(log, index) in logs" :key="index">
            <td class="time">{{ new Date(log.timestamp * 1000).toLocaleString() }}</td>
            <td><span class="method" :class="log.method">{{ log.method }}</span></td>
            <td class="path">{{ log.path }}</td>
            <td>
              <span class="status" :class="log.status >= 400 ? 'error' : 'success'">
                {{ log.status }}
              </span>
            </td>
            <td>{{ log.duration }}</td>
          </tr>
          <tr v-if="logs.length === 0">
            <td colspan="5" class="empty">暂无日志</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.toolbar h3 { margin: 0; font-size: 16px; color: #f1f5f9; }

.btn-secondary {
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: #e2e8f0;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}
.btn-secondary:hover { background: rgba(51, 65, 85, 0.8); }

.table-container {
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.4);
  overflow: auto;
  max-height: 600px;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th {
  text-align: left;
  padding: 10px 14px;
  background: rgba(30, 41, 59, 0.6);
  color: #94a3b8;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  position: sticky;
  top: 0;
}

td {
  padding: 8px 14px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.05);
  color: #cbd5e1;
}

.method {
  font-weight: 700;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}
.method.GET { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
.method.POST { background: rgba(16, 185, 129, 0.15); color: #34d399; }
.method.DELETE { background: rgba(239, 68, 68, 0.15); color: #f87171; }

.status.success { color: #4ade80; }
.status.error { color: #f87171; font-weight: bold; }

.empty { text-align: center; padding: 20px; color: #64748b; }
.time { white-space: nowrap; color: #94a3b8; font-family: monospace; }
</style>
