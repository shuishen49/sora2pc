<script setup>
import { ref, computed } from 'vue'
import { useGenerateStore } from '../../stores/generate'

const store = useGenerateStore()

defineProps({
  // No modelValue needed if we use store, but keeping for compatibility if utilized elsewhere
})

const emit = defineEmits(['append-prompt'])

// Mock Roles Data (Legacy format adaptation)
const roles = ref([
  {
    name: "电影质感",
    desc: "35mm胶片，高对比度，颗粒感，诺兰风格",
    prompt: "shot on 35mm film, cinematic lighting, high contrast, film grain, Nolan style,",
    avatar: "",
    tags: ["摄影", "风格"]
  },
  {
    name: "赛博朋克",
    desc: "霓虹灯，雨夜，高科技低生活，未来城市",
    prompt: "cyberpunk city, neon lights, rain, high tech low life, futuristic,",
    avatar: "",
    tags: ["风格", "科幻"]
  },
  {
    name: "微距摄影",
    desc: "极度细节，昆虫视角，浅景深，虚化背景",
    prompt: "macro photography, extreme detail, shallow depth of field, bokeh,",
    avatar: "",
    tags: ["摄影"]
  },
    {
    name: "吉卜力风格",
    desc: "宫崎骏画风，手绘感，清新自然，蓝天白云",
    prompt: "Studio Ghibli style, anime style, hand drawn, vivid colors, lush nature,",
    avatar: "",
    tags: ["动漫", "风格"]
  },
  {
    name: "极简主义",
    desc: "简洁线条，留白，低饱和度，现代感",
    prompt: "minimalist, clean lines, negative space, low saturation, modern,",
    avatar: "",
    tags: ["风格"]
  }
])

const searchQuery = ref('')
const filterTab = ref('all') // all, fav

const filteredRoles = computed(() => {
  let list = roles.value

  if (filterTab.value === 'fav') {
      list = list.filter(r => store.isRoleFavorite(r.name))
  }

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(r =>
      r.name.toLowerCase().includes(q) ||
      r.desc.toLowerCase().includes(q) ||
      r.tags.some(t => t.toLowerCase().includes(q))
    )
  }
  return list
})

const onCardClick = (role) => {
    // Default action: append to prompt
    emit('append-prompt', role.prompt)
}

const toggleFav = (e, role) => {
    e.stopPropagation()
    store.toggleRoleFavorite(role.name)
}

const mountRole = (e, role) => {
    e.stopPropagation()
    store.attachRole(role)
}

// Generate a color gradient for avatar based on name if no avatar
const getAvatarStyle = (name) => {
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
        hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    const c1 = `hsl(${hash % 360}, 70%, 60%)`
    const c2 = `hsl(${(hash + 40) % 360}, 70%, 50%)`
    return { background: `linear-gradient(135deg, ${c1}, ${c2})` }
}
</script>

<template>
  <div class="role-selector">
    <div class="role-header">
      <h3>角色卡 / 预设</h3>
      <div class="tabs">
        <span class="tab" :class="{ active: filterTab === 'all' }" @click="filterTab = 'all'">全部</span>
        <span class="tab" :class="{ active: filterTab === 'fav' }" @click="filterTab = 'fav'">收藏</span>
      </div>
    </div>

    <div class="search-bar">
      <input v-model="searchQuery" placeholder="搜索角色/风格..." />
      <span class="search-icon">🔍</span>
    </div>

    <div class="roles-grid">
      <div
        v-for="role in filteredRoles"
        :key="role.name"
        class="role-card"
        @click="onCardClick(role)"
        title="点击添加到提示词"
      >
        <div class="role-avatar" :style="getAvatarStyle(role.name)">
             <span class="avatar-text">{{ role.name[0] }}</span>
        </div>
        <div class="role-info">
            <div class="role-top">
                <span class="role-name">{{ role.name }}</span>
                <div class="role-tags">
                    <span v-for="tag in role.tags.slice(0, 1)" :key="tag" class="tag">{{ tag }}</span>
                </div>
            </div>
            <p class="role-desc">{{ role.desc }}</p>
        </div>

        <!-- Actions -->
        <div class="role-actions">
            <button class="icon-btn" :class="{ active: store.isRoleFavorite(role.name) }" @click="toggleFav($event, role)" title="收藏">
                {{ store.isRoleFavorite(role.name) ? '❤️' : '🤍' }}
            </button>
            <button class="icon-btn" @click="mountRole($event, role)" title="挂载为全局角色">
                📌
            </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.role-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  overflow: hidden;
}

.role-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.role-header h3 {
  margin: 0;
  font-size: 16px;
  color: #f1f5f9;
}

.tabs {
  display: flex;
  gap: 4px;
  background: rgba(15, 23, 42, 0.4);
  padding: 3px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.1);
}

.tab {
  padding: 4px 10px;
  font-size: 11px;
  color: #94a3b8;
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
}

.tab.active {
  background: #334155;
  color: #f1f5f9;
  font-weight: 600;
}

.search-bar {
  position: relative;
}

.search-bar input {
  width: 100%;
  background: #0f172a;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 10px;
  padding: 10px 10px 10px 36px;
  color: #f1f5f9;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}

.search-bar input:focus {
  border-color: #3b82f6;
}

.search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 14px;
  opacity: 0.5;
}

.roles-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  padding-right: 4px;
  flex: 1;
  min-height: 0;
}

/* Scrollbar styling */
.roles-grid::-webkit-scrollbar { width: 4px; }
.roles-grid::-webkit-scrollbar-thumb { background: #334155; border-radius: 4px; }

.role-card {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.role-card:hover {
  background: rgba(51, 65, 85, 0.6);
  border-color: rgba(59, 130, 246, 0.4);
  transform: translateY(-1px);
}

.role-card:active {
  transform: translateY(0);
}

.role-avatar {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 10px rgba(0,0,0,0.2);
}

.avatar-text {
  font-size: 18px;
  font-weight: 700;
  color: white;
  text-shadow: 0 1px 2px rgba(0,0,0,0.3);
}

.role-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.role-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.role-name {
  font-size: 13px;
  font-weight: 600;
  color: #f1f5f9;
}

.role-tags {
    display: flex;
    gap: 4px;
}
.tag {
    font-size: 10px;
    padding: 2px 6px;
    background: rgba(148, 163, 184, 0.1);
    color: #94a3b8;
    border-radius: 4px;
}

.role-desc {
  font-size: 11px;
  color: #94a3b8;
  margin: 0;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.role-actions {
    display: flex;
    flex-direction: column;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.2s;
}

.role-card:hover .role-actions {
    opacity: 1;
}

.icon-btn {
    width: 24px;
    height: 24px;
    background: rgba(148, 163, 184, 0.1);
    border: none;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 12px;
    transition: all 0.2s;
}

.icon-btn:hover {
    background: rgba(148, 163, 184, 0.3);
}

.icon-btn.active {
    color: #ef4444; /* Red heart */
}
</style>
