<script setup>
import { computed } from 'vue'

const props = defineProps({
  show: Boolean,
  task: { type: Object, default: null }
})

const emit = defineEmits(['close', 'download'])

const isVideo = computed(() => {
    if (!props.task || !props.task.url) return false
    return /\.(mp4|mov|webm)$/i.test(props.task.url) || props.task.type === 'video'
})

const download = () => {
    emit('download', props.task)
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
        <button class="close-btn" @click="$emit('close')">✕</button>

        <div class="media-container" v-if="task">
            <video v-if="isVideo" :src="task.url" controls autoplay loop playsinline></video>
            <img v-else :src="task.url" />
        </div>

        <div class="info-bar" v-if="task">
            <div class="task-meta">
                <span class="id">#{{ task.id }}</span>
                <span class="model">{{ task.model }}</span>
            </div>
            <div class="actions">
                <button class="btn primary" @click="download">下载</button>
            </div>
        </div>

        <div class="prompt-box" v-if="task">
            <p>{{ task.prompt }}</p>
        </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.85);
    backdrop-filter: blur(5px);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
}

.modal-content {
    background: #0f172a;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 16px;
    width: 100%;
    max-width: 900px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    box-shadow: 0 20px 50px rgba(0,0,0,0.5);
}

.close-btn {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 10;
    background: rgba(0,0,0,0.5);
    border: none;
    color: white;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    cursor: pointer;
    font-size: 16px;
    transition: all 0.2s;
}

.close-btn:hover {
    background: rgba(255, 255, 255, 0.2);
}

.media-container {
    flex: 1;
    background: black;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 300px;
    max-height: 60vh;
}

.media-container img, .media-container video {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
}

.info-bar {
    padding: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top: 1px solid rgba(148, 163, 184, 0.1);
    background: rgba(15, 23, 42, 0.8);
}

.task-meta {
    display: flex;
    gap: 12px;
    align-items: center;
}

.id {
    font-family: monospace;
    color: #64748b;
    font-size: 12px;
}

.model {
    padding: 2px 8px;
    background: rgba(59, 130, 246, 0.1);
    color: #93c5fd;
    border-radius: 4px;
    font-size: 11px;
}

.btn {
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
}

.btn.primary {
    background: #3b82f6;
    color: white;
}

.btn.primary:hover {
    background: #2563eb;
}

.prompt-box {
    padding: 16px;
    background: #0b1120;
    color: #cbd5e1;
    font-size: 13px;
    line-height: 1.6;
    max-height: 150px;
    overflow-y: auto;
    border-top: 1px solid rgba(148, 163, 184, 0.1);
}
</style>
