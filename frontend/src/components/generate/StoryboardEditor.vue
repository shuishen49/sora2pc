<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  title: {
    type: String,
    default: ''
  },
  context: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'update:title', 'update:context'])

// Local shots
const shots = computed({
  get: () => props.modelValue.length > 0 ? props.modelValue : generateShots(8),
  set: (val) => emit('update:modelValue', val)
})

const generateShots = (count) => {
  return Array.from({ length: count }, (_, i) => ({
    index: i + 1,
    prompt: '',
    file: null
  }))
}

const localTitle = computed({
  get: () => props.title,
  set: (val) => emit('update:title', val)
})

const localContext = computed({
  get: () => props.context,
  set: (val) => emit('update:context', val)
})

const shotCount = ref(8)

const applyCount = () => {
  const newShots = generateShots(shotCount.value)
  // Preserve existing prompts
  shots.value.forEach((s, i) => {
    if (newShots[i]) {
      newShots[i].prompt = s.prompt
      newShots[i].file = s.file
    }
  })
  emit('update:modelValue', newShots)
}

const updateShot = (index, field, value) => {
  const newShots = [...shots.value]
  newShots[index] = { ...newShots[index], [field]: value }
  emit('update:modelValue', newShots)
}

const clearAll = () => {
  emit('update:modelValue', generateShots(shotCount.value))
  emit('update:context', '')
}

const importFromLines = (text) => {
  const lines = text.split('\n').filter(l => l.trim())
  shotCount.value = lines.length || 1
  const newShots = lines.map((line, i) => ({
    index: i + 1,
    prompt: line.trim(),
    file: null
  }))
  emit('update:modelValue', newShots)
}
</script>

<template>
  <div class="storyboard-editor">
    <div class="editor-header">
      <span class="title">分镜编辑器</span>
      <span class="hint">每一镜一条提示，适合连续剧情</span>
    </div>

    <!-- Meta Row -->
    <div class="meta-row">
      <input
        type="text"
        v-model="localTitle"
        placeholder="分镜组标题（可选：例如《篮球裁决》）"
        class="title-input"
      />
      <div class="shot-count-row">
        <span class="label">镜头数</span>
        <input
          type="number"
          v-model.number="shotCount"
          class="count-input"
          min="1"
          max="200"
        />
        <button class="apply-btn" @click="applyCount">应用</button>
      </div>
      <button class="clear-btn" @click="clearAll">清空分镜</button>
    </div>

    <!-- Context (Global Setting) -->
    <div class="context-section">
      <label>连续性 / 统一设定（会自动加到每一镜前面）</label>
      <textarea
        v-model="localContext"
        placeholder="例如：统一画风、服装、场景、镜头语言、人物特征（建议英文更稳定）"
        class="context-textarea"
        rows="3"
      ></textarea>
    </div>

    <!-- Shots -->
    <div class="shots-list">
      <div
        v-for="(shot, index) in shots"
        :key="index"
        class="shot-row"
      >
        <div class="shot-header">
          <span class="shot-index">镜头 {{ shot.index }}</span>
        </div>
        <textarea
          :value="shot.prompt"
          @input="updateShot(index, 'prompt', $event.target.value)"
          placeholder="描述这一镜的内容..."
          class="shot-textarea"
          rows="2"
        ></textarea>
      </div>
    </div>
  </div>
</template>

<style scoped>
.storyboard-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  background: rgba(15, 23, 42, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 12px;
}

.editor-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.title {
  font-size: 14px;
  font-weight: 700;
  color: #e2e8f0;
}

.hint {
  font-size: 11px;
  color: #64748b;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.title-input {
  flex: 1;
  min-width: 200px;
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 13px;
}

.shot-count-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label {
  font-size: 12px;
  color: #94a3b8;
}

.count-input {
  width: 60px;
  padding: 6px 8px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 6px;
  color: #e2e8f0;
  font-size: 12px;
  text-align: center;
}

.apply-btn, .clear-btn {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.apply-btn {
  background: rgba(59, 130, 246, 0.2);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #93c5fd;
}

.apply-btn:hover {
  background: rgba(59, 130, 246, 0.3);
}

.clear-btn {
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid rgba(248, 113, 113, 0.3);
  color: #f87171;
}

.clear-btn:hover {
  background: rgba(248, 113, 113, 0.2);
}

.context-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.context-section label {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 600;
}

.context-textarea {
  width: 100%;
  padding: 10px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
  outline: none;
}

.shots-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
  padding-right: 4px;
}

.shot-row {
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 8px;
  padding: 8px;
}

.shot-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.shot-index {
  font-size: 11px;
  font-weight: 700;
  color: #a78bfa;
  background: rgba(167, 139, 250, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.shot-textarea {
  width: 100%;
  padding: 8px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 6px;
  color: #e2e8f0;
  font-size: 12px;
  line-height: 1.4;
  resize: none;
  outline: none;
}

.shot-textarea:focus {
  border-color: #a78bfa;
}

/* Custom scrollbar */
.shots-list::-webkit-scrollbar {
  width: 4px;
}
.shots-list::-webkit-scrollbar-thumb {
  background: #334155;
  border-radius: 4px;
}
</style>
