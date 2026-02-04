<script setup>
import { ref, onUnmounted } from 'vue'

defineProps({
  modelValue: { type: [File, null], default: null }, // The file object
})

const emit = defineEmits(['update:modelValue', 'file-selected'])

const fileInput = ref(null)
const isDragging = ref(false)
const previewUrl = ref(null)
const fileMeta = ref('')
const fileType = ref('') // 'image' | 'video' | 'unknown'

const clearPreview = () => {
    if (previewUrl.value) {
        URL.revokeObjectURL(previewUrl.value)
        previewUrl.value = null
    }
    fileMeta.value = ''
    fileType.value = ''
}

const handleFile = (file) => {
    if (!file) return
    clearPreview()

    // Create Preview
    const url = URL.createObjectURL(file)
    previewUrl.value = url

    // Determine type
    if (file.type.startsWith('image/')) {
        fileType.value = 'image'
        // Create img to get dimensions
        const img = new Image()
        img.onload = () => {
            fileMeta.value = `${img.width}x${img.height} · ${(file.size / 1024 / 1024).toFixed(2)}MB`
        }
        img.src = url
    } else if (file.type.startsWith('video/')) {
        fileType.value = 'video'
        fileMeta.value = `${(file.size / 1024 / 1024).toFixed(2)}MB`
        // Video dimensions usually need a hidden video element to read metadata, simplified for now
    } else {
        fileType.value = 'unknown'
        fileMeta.value = `${(file.size / 1024 / 1024).toFixed(2)}MB`
    }

    emit('update:modelValue', file)
    emit('file-selected', { file, url, type: fileType.value })
}

const onFileChange = (e) => {
    const file = e.target.files?.[0]
    if (file) handleFile(file)

    // Reset input so same file can be selected again if needed
    e.target.value = ''
}

const onDrop = (e) => {
    isDragging.value = false
    const file = e.dataTransfer?.files?.[0]
    if (file) handleFile(file)
}

const removeFile = () => {
    clearPreview()
    emit('update:modelValue', null)
}

// Cleanup on unmount
onUnmounted(() => {
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

const triggerUpload = () => fileInput.value?.click()

</script>

<template>
  <div class="file-uploader"
       :class="{ dragging: isDragging, 'has-file': !!previewUrl }"
       @dragover.prevent="isDragging = true"
       @dragleave.prevent="isDragging = false"
       @drop.prevent="onDrop">

    <input ref="fileInput" type="file" accept="image/*,video/*" hidden @change="onFileChange" />

    <!-- Empty State -->
    <div v-if="!previewUrl" class="upload-placeholder" @click="triggerUpload">
        <div class="icon-box">
            <span class="icon">📂</span>
        </div>
        <div class="text-group">
            <span class="main-text">点击或拖拽上传参考图/视频</span>
            <span class="sub-text">支持 JPG, PNG, MP4, MOV 等</span>
        </div>
    </div>

    <!-- Preview State -->
    <div v-else class="preview-container">
        <div class="media-wrapper">
            <img v-if="fileType === 'image'" :src="previewUrl" class="preview-media" />
            <video v-else-if="fileType === 'video'" :src="previewUrl" class="preview-media" autoplay muted loop playsinline></video>
            <div v-else class="preview-unknown">
                <span>📄 {{ fileType }}</span>
            </div>

            <!-- Overlay Info -->
            <div class="media-overlay">
                <span class="meta-badge" v-if="fileMeta">{{ fileMeta }}</span>
            </div>
        </div>

        <button class="remove-btn" @click.stop="removeFile" title="移除文件">✕</button>
    </div>
  </div>
</template>

<style scoped>
.file-uploader {
    width: 100%;
    min-height: 120px;
    border: 2px dashed rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: rgba(15, 23, 42, 0.3);
    transition: all 0.2s;
    position: relative;
    display: flex;
    overflow: hidden;
}

.file-uploader:hover {
    border-color: rgba(148, 163, 184, 0.4);
    background: rgba(15, 23, 42, 0.5);
}

.file-uploader.dragging {
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
}

.file-uploader.has-file {
    border-style: solid;
    border-color: rgba(148, 163, 184, 0.1);
    background: black;
    padding: 0;
}

.upload-placeholder {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 20px;
    cursor: pointer;
    gap: 12px;
}

.icon-box {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: rgba(148, 163, 184, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
}

.icon {
    font-size: 24px;
    opacity: 0.7;
}

.text-group {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
}

.main-text {
    font-size: 14px;
    color: #e2e8f0;
    font-weight: 500;
}

.sub-text {
    font-size: 12px;
    color: #64748b;
}

.preview-container {
    width: 100%;
    height: 100%;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #000;
}

.media-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.preview-media {
    max-width: 100%;
    max-height: 240px; /* Limit height */
    object-fit: contain;
}

.media-overlay {
    position: absolute;
    bottom: 8px;
    left: 8px;
    display: flex;
    gap: 6px;
    pointer-events: none;
}

.meta-badge {
    padding: 2px 6px;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    border-radius: 4px;
    color: white;
    font-size: 11px;
    font-weight: 500;
}

.remove-btn {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.6);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: white;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    transition: all 0.2s;
}

.remove-btn:hover {
    background: rgba(239, 68, 68, 0.8);
    border-color: transparent;
}
</style>
