<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  globalCount: {
    type: Number,
    default: 2
  }
})

const emit = defineEmits(['update:modelValue'])

// Local prompt rows
const rows = computed({
  get: () => props.modelValue.length > 0 ? props.modelValue : [{ prompt: '', count: props.globalCount, file: null }],
  set: (val) => emit('update:modelValue', val)
})

const addRow = () => {
  const newRows = [...rows.value, { prompt: '', count: props.globalCount, file: null }]
  emit('update:modelValue', newRows)
}

const removeRow = (index) => {
  if (rows.value.length <= 1) return
  const newRows = rows.value.filter((_, i) => i !== index)
  emit('update:modelValue', newRows)
}

const updateRow = (index, field, value) => {
  const newRows = [...rows.value]
  newRows[index] = { ...newRows[index], [field]: value }
  emit('update:modelValue', newRows)
}

const onFileChange = (index, e) => {
  const f = e.target.files?.[0]
  if (f) {
    updateRow(index, 'file', f)
  }
}
</script>

<template>
  <div class="multi-prompt-editor">
    <div class="editor-header">
      <span class="title">多提示编辑器</span>
      <span class="hint">每行一条提示，可单独设置生成份数和参考文件</span>
    </div>

    <div class="prompt-rows">
      <div
        v-for="(row, index) in rows"
        :key="index"
        class="prompt-row"
      >
        <div class="row-header">
          <span class="row-index">#{{ index + 1 }}</span>
          <input
            type="number"
            :value="row.count"
            @input="updateRow(index, 'count', parseInt($event.target.value) || 1)"
            class="count-input"
            min="1"
            max="99"
            title="生成份数"
          />
          <label class="file-btn" title="选择参考文件">
            <input type="file" accept="image/*,video/*" hidden @change="onFileChange(index, $event)" />
            {{ row.file ? row.file.name.slice(0, 12) + '...' : '📎 文件' }}
          </label>
          <button
            class="remove-btn"
            @click="removeRow(index)"
            :disabled="rows.length <= 1"
            title="删除此行"
          >
            ✕
          </button>
        </div>
        <textarea
          :value="row.prompt"
          @input="updateRow(index, 'prompt', $event.target.value)"
          placeholder="输入提示词..."
          class="prompt-textarea"
          rows="3"
        ></textarea>
      </div>
    </div>

    <button class="add-btn" @click="addRow">
      <span>+</span> 新增提示
    </button>
  </div>
</template>

<style scoped>
.multi-prompt-editor {
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
  font-size: 12px; /* Smaller font */
  font-weight: 700;
  color: #e2e8f0;
  white-space: nowrap;
}

.hint {
  font-size: 10px; /* Smaller font */
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.prompt-rows {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.prompt-row {
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 10px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.row-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.row-index {
  font-size: 12px;
  font-weight: 700;
  color: #3b82f6;
  min-width: 30px;
}

.count-input {
  width: 50px;
  padding: 4px 8px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 6px;
  color: #e2e8f0;
  font-size: 12px;
  text-align: center;
}

.file-btn {
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 6px;
  color: #93c5fd;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
}

.file-btn:hover {
  background: rgba(59, 130, 246, 0.25);
}

.remove-btn {
  margin-left: auto;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid rgba(248, 113, 113, 0.3);
  background: rgba(248, 113, 113, 0.1);
  color: #f87171;
  cursor: pointer;
  font-size: 12px;
}

.remove-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.prompt-textarea {
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

.prompt-textarea:focus {
  border-color: #3b82f6;
}

.add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px;
  background: rgba(34, 197, 94, 0.1);
  border: 1px dashed rgba(34, 197, 94, 0.3);
  border-radius: 8px;
  color: #4ade80;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.add-btn:hover {
  background: rgba(34, 197, 94, 0.2);
  border-color: rgba(34, 197, 94, 0.5);
}
</style>
