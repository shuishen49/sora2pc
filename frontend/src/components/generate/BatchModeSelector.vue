<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: 'single'
  }
})

const emit = defineEmits(['update:modelValue'])

const modes = [
  { value: 'single', title: '单次', sub: '1 条', description: '只创建 1 条任务（取第 1 个文件）' },
  { value: 'same_prompt_files', title: '同提示', sub: '多文件', description: '多文件共享同一提示；可设置每文件生成份数' },
  { value: 'multi_prompt', title: '多提示', sub: '按行', description: '每条提示各自生成，可给某行附带文件' },
  { value: 'storyboard', title: '分镜', sub: '连续', description: '连续剧情更适合；任务区会打分镜编号' },
  { value: 'character', title: '角色卡', sub: '创建', description: '上传视频创建角色卡，无需提示词' }
]

const currentMode = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>

<template>
  <div class="batch-mode-selector">
    <span class="mode-label">模式</span>
    <div class="mode-options">
      <label
        v-for="mode in modes"
        :key="mode.value"
        class="mode-option"
        :class="{ active: currentMode === mode.value }"
        :title="mode.description"
      >
        <input
          type="radio"
          :value="mode.value"
          v-model="currentMode"
          class="sr-only"
        />
        <span class="mode-title">{{ mode.title }}</span>
        <span class="mode-sub">{{ mode.sub }}</span>
      </label>
    </div>
  </div>
</template>

<style scoped>
.batch-mode-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  background: rgba(15, 23, 42, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 12px;
  max-width: 100%;
  box-sizing: border-box;
  overflow-x: auto;
}

.mode-label {
  font-size: 13px;
  font-weight: 700;
  color: #94a3b8;
}

.mode-options {
  display: flex;
  gap: 2px;
  padding: 3px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 8px;
  border-radius: 8px;
  flex: 1; /* Allow container to fill space */
  justify-content: space-between; /* Distribute items */
  min-width: 0;
}

.mode-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  padding: 6px 8px;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
  transition: all 0.15s ease;
  transition: all 0.15s ease;
  min-width: 50px;
  flex: 1; /* Allow each option to grow */
  justify-content: center; /* Center content vertically/horizontally */
}

.mode-option:hover {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.3);
}

.mode-option.active {
  background: linear-gradient(135deg, #2563eb, #4f46e5);
  border-color: rgba(255, 255, 255, 0.3);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
}

.mode-title {
  font-size: 12px;
  font-weight: 700;
  color: #e2e8f0;
  text-align: center;
}

.mode-sub {
  font-size: 10px;
  color: #94a3b8;
  text-align: center;
}

.mode-option.active .mode-title,
.mode-option.active .mode-sub {
  color: #fff;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  border: 0;
}
</style>
