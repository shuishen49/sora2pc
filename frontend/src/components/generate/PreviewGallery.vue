<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  tasks: { type: Array, default: () => [] }
})

const emit = defineEmits(['open-preview', 'download'])

const filter = ref('all') // all, image, video

const filteredTasks = computed(() => {
    let list = props.tasks.filter(t => t.url && (t.status === 'done' || t.result))
    if (filter.value === 'image') list = list.filter(t => t.url && /\.(png|jpg|webp)$/i.test(t.url))
    if (filter.value === 'video') list = list.filter(t => t.url && /\.(mp4|mov|webm)$/i.test(t.url))
    return list
})

const isVideo = (url) => /\.(mp4|mov|webm)$/i.test(url)

const openPreview = (task) => {
    emit('open-preview', task)
}
</script>

<template>
  <div class="preview-gallery">
    <div class="filter-bar">
        <span :class="{ active: filter === 'all' }" @click="filter = 'all'">全部</span>
        <span :class="{ active: filter === 'image' }" @click="filter = 'image'">图片</span>
        <span :class="{ active: filter === 'video' }" @click="filter = 'video'">视频</span>
    </div>

    <div class="gallery-grid" v-if="filteredTasks.length">
        <div v-for="task in filteredTasks" :key="task.id" class="gallery-item" @click="openPreview(task)">
            <video v-if="isVideo(task.url)" :src="task.url" muted loop playsinline onmouseover="this.play()" onmouseout="this.pause()"></video>
            <img v-else :src="task.url" />

            <div class="overlay">
                <button class="action-btn" @click.stop="$emit('download', task)">⬇</button>
            </div>
            <div class="task-id">#{{ task.id }}</div>
        </div>
    </div>
    <div v-else class="empty-state">
        暂无预览内容
    </div>
  </div>
</template>

<style scoped>
.preview-gallery {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
}

.filter-bar {
    display: flex;
    gap: 8px;
    padding: 0 4px;
}

.filter-bar span {
    font-size: 11px;
    color: #64748b;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    background: rgba(148, 163, 184, 0.1);
}

.filter-bar span.active {
    background: #3b82f6;
    color: white;
}

.gallery-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
    gap: 10px;
    overflow-y: auto;
    padding-right: 4px;
}

.gallery-grid::-webkit-scrollbar { width: 4px; }
.gallery-grid::-webkit-scrollbar-thumb { background: #334155; border-radius: 4px; }

.gallery-item {
    aspect-ratio: 16/9;
    background: black;
    border-radius: 8px;
    overflow: hidden;
    position: relative;
    cursor: pointer;
    border: 1px solid rgba(148, 163, 184, 0.1);
    transition: all 0.2s;
}

.gallery-item:hover {
    border-color: #3b82f6;
    transform: scale(1.02);
}

.gallery-item img, .gallery-item video {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.overlay {
    position: absolute;
    top: 4px;
    right: 4px;
    display: none;
}

.gallery-item:hover .overlay {
    display: block;
}

.action-btn {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.6);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    cursor: pointer;
}

.task-id {
    position: absolute;
    bottom: 4px;
    left: 4px;
    font-size: 10px;
    color: rgba(255, 255, 255, 0.7);
    text-shadow: 0 1px 2px black;
    background: rgba(0,0,0,0.3);
    padding: 1px 3px;
    border-radius: 3px;
}

.empty-state {
    color: #64748b;
    font-size: 12px;
    text-align: center;
    padding-top: 40px;
}
</style>
