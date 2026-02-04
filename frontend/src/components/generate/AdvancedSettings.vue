<script setup>
import { ref, reactive, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['update:modelValue'])

const settings = reactive({
  batchEnabled: false,
  batchCount: 5,
  proxyUrl: '',
  timeout: 300,
  debug: false
})

const isOpen = ref(false)

watch(settings, (newVal) => {
  emit('update:modelValue', newVal)
}, { deep: true })
</script>

<template>
  <div class="advanced-settings">
    <div class="toggle-header" @click="isOpen = !isOpen">
      <span class="icon">{{ isOpen ? '▼' : '▶' }}</span>
      <span class="label">高级设置 / Advanced Settings</span>
      <span v-if="settings.batchEnabled" class="badge">批处理 ON</span>
    </div>

    <div v-show="isOpen" class="settings-body">
      <!-- Batch Mode -->
      <div class="setting-group">
        <div class="checkbox-row">
            <input type="checkbox" id="batch" v-model="settings.batchEnabled" />
            <label for="batch">启用批量生成 / Batch Mode</label>
        </div>
        <div v-if="settings.batchEnabled" class="sub-field">
            <label>生成份数</label>
            <input type="number" v-model="settings.batchCount" min="1" max="100" />
        </div>
      </div>

      <!-- Proxy -->
      <div class="setting-group">
          <label>自定义代理 API (覆盖全局)</label>
          <input type="text" v-model="settings.proxyUrl" placeholder="http://..." />
      </div>

       <!-- Debug -->
       <div class="setting-group">
          <div class="checkbox-row">
            <input type="checkbox" id="debug-adv" v-model="settings.debug" />
            <label for="debug-adv">调试模式 (Debug)</label>
          </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.advanced-settings {
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.3);
  overflow: hidden;
}

.toggle-header {
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  background: rgba(30, 41, 59, 0.3);
  transition: background 0.2s;
}
.toggle-header:hover { background: rgba(30, 41, 59, 0.6); }

.icon { font-size: 10px; color: #94a3b8; }
.label { font-size: 13px; font-weight: 500; color: #cbd5e1; }
.badge {
    font-size: 10px;
    background: rgba(34, 197, 94, 0.2);
    color: #4ade80;
    padding: 2px 6px;
    border-radius: 4px;
    margin-left: auto;
}

.settings-body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
}

.setting-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
}
.setting-group label { font-size: 12px; color: #94a3b8; }
.setting-group input[type="text"], .setting-group input[type="number"] {
    background: #0f172a;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 6px;
    padding: 8px;
    font-size: 13px;
    color: #f1f5f9;
    outline: none;
}
.setting-group input:focus { border-color: #3b82f6; }

.checkbox-row {
    display: flex;
    align-items: center;
    gap: 8px;
}
.checkbox-row label { font-size: 13px; color: #e2e8f0; cursor: pointer; }

.sub-field {
    margin-left: 24px;
    display: flex;
    align-items: center;
    gap: 10px;
}
.sub-field input { width: 80px; }
</style>
