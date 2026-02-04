<script setup>
import { computed } from 'vue'

defineProps({
  logs: { type: Array, default: () => [] }
})

const emit = defineEmits(['clear'])

</script>

<template>
  <div class="task-log">
      <div class="log-actions">
          <button @click="emit('clear')" class="clear-btn">清空日志</button>
      </div>
      <div class="log-content">
          <div v-for="(log, idx) in logs" :key="idx" class="log-line">
              <span class="time">[{{ log.time }}]</span>
              <span class="msg" :class="log.type">{{ log.message }}</span>
          </div>
          <div v-if="logs.length === 0" class="log-empty">暂无日志</div>
      </div>
  </div>
</template>

<style scoped>
.task-log {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #0b1120;
    font-family: monospace;
}

.log-actions {
    padding: 8px;
    border-bottom: 1px solid rgba(148, 163, 184, 0.1);
    text-align: right;
}

.clear-btn {
    font-size: 11px;
    color: #64748b;
    background: transparent;
    border: 1px solid rgba(148, 163, 184, 0.2);
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
}

.log-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    font-size: 11px;
    color: #cbd5e1;
}

.log-line {
    margin-bottom: 4px;
    line-height: 1.4;
    word-break: break-all;
}

.time {
    color: #475569;
    margin-right: 8px;
}

.msg.error { color: #f87171; }
.msg.success { color: #4ade80; }
.msg.info { color: #94a3b8; }

.log-empty {
    text-align: center;
    color: #334155;
    margin-top: 20px;
}
</style>
