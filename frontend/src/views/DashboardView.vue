<script setup>
import { ref, computed } from 'vue'
import GenerateView from './GenerateView.vue'
import TokenManager from '../components/manage/TokenManager.vue'
import SettingsPanel from '../components/manage/SettingsPanel.vue'
import LogViewer from '../components/manage/LogViewer.vue'

const activeTab = ref('generate')

const tabs = [
  { id: 'tokens', label: 'Token 管理', icon: '🔑' },
  { id: 'generate', label: '生成面板', icon: '🎨' },
  { id: 'settings', label: '系统配置', icon: '⚙️' },
  { id: 'logs', label: '请求日志', icon: '📜' },
]

const currentTabLabel = computed(() => {
    return tabs.find(t => t.id === activeTab.value)?.label || 'Console'
})

const isSidebarCollapsed = ref(false)
const toggleSidebar = () => {
    isSidebarCollapsed.value = !isSidebarCollapsed.value
}
</script>

<template>
  <div class="admin-layout">
    <!-- Sidebar -->
    <aside class="sidebar" :class="{ collapsed: isSidebarCollapsed }">
      <div class="brand-area">
        <div class="logo-box">S2</div>
        <div class="brand-info" v-show="!isSidebarCollapsed">
          <h1>Sora2 API</h1>
        </div>
      </div>

      <!-- Toggle Button (Absolute on border) -->
      <button class="collapse-btn" @click="toggleSidebar">
          {{ isSidebarCollapsed ? '»' : '«' }}
      </button>

      <nav class="nav-menu">
        <div
          v-for="tab in tabs"
          :key="tab.id"
          class="nav-item"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
          :title="isSidebarCollapsed ? tab.label : ''"
        >
          <span class="nav-icon">{{ tab.icon }}</span>
          <transition name="fade">
            <span class="nav-label" v-show="!isSidebarCollapsed">{{ tab.label }}</span>
          </transition>
        </div>
      </nav>

      <div class="user-profile">
        <div class="user-avatar" :title="isSidebarCollapsed ? 'Administrator' : ''">AD</div>
        <div class="user-info" v-show="!isSidebarCollapsed">
          <span class="username">Administrator</span>
          <button class="logout-link" @click="auth.logout()">退出登录</button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content">
      <header class="top-bar">
        <div class="breadcrumbs">
            <span class="crumb-root">控制台</span>
            <span class="separator">/</span>
            <span class="crumb-current">{{ currentTabLabel }}</span>
        </div>
        <div class="top-actions">
           <a href="https://github.com/shuishen49/sora2pc" target="_blank" class="github-link">
               GitHub
           </a>
        </div>
      </header>

      <div class="content-body" :class="{ 'fixed-body': activeTab === 'generate' }">
        <transition name="fade" mode="out-in">
          <div :key="activeTab" class="view-container">
            <TokenManager v-if="activeTab === 'tokens'" />
            <GenerateView v-else-if="activeTab === 'generate'" />
            <SettingsPanel v-else-if="activeTab === 'settings'" />
            <LogViewer v-else-if="activeTab === 'logs'" />
          </div>
        </transition>
      </div>
    </main>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: radial-gradient(circle at 10% 20%, #0f172a, #020617 80%);
  color: #f8fafc;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: rgba(15, 23, 42, 0.6);
  border-right: 1px solid rgba(148, 163, 184, 0.1);
  backdrop-filter: blur(20px);
  padding: 24px;
  z-index: 20;
  padding: 24px;
  z-index: 20;
  transition: width 0.4s cubic-bezier(0.2, 0, 0, 1), padding 0.4s cubic-bezier(0.2, 0, 0, 1);
  position: relative;
}

/* ... existing sidebar.collapsed ... */

.collapse-btn {
    position: absolute;
    top: 50%;
    right: -12px; /* Half width overlapping border */
    transform: translateY(-50%);
    width: 24px;
    height: 24px;
    background: #0f172a;
    border: 1px solid rgba(148, 163, 184, 0.3);
    border-radius: 50%;
    color: #94a3b8;
    cursor: pointer;
    font-size: 14px;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 30;
    transition: all 0.2s;
    padding: 0;
}

.collapse-btn:hover {
    background: #3b82f6;
    color: white;
    border-color: #3b82f6;
}

/* Hide button when collapsed if desired, OR check rotation logic */
.sidebar.collapsed .collapse-btn {
    /* No special positioning needed strictly if it stays on border, but maybe rotation */
    /* transform: translateY(-50%) rotate(180deg); -> if verify arrow direction */
    /* Assuming « is for collapse (pointing left? or typical hamburger logic).
       Actually « usually means "collapse left". » means "expand right".
       So icon logic in template handles text.
    */
}

.sidebar.collapsed {
  width: 80px;
  padding: 24px 12px; /* Reduce horizontal padding */
}

.brand-area {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 40px;
  padding-left: 8px;
  position: relative;
  min-height: 40px;
}

.sidebar.collapsed .brand-area {
  justify-content: center;
  padding-left: 0;
}

/* Remove old button CSS if exists implicitly or explicitly in replaced chunks */

.logo-box {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #38bdf8, #818cf8);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: #fff;
  font-size: 14px;
  box-shadow: 0 0 20px rgba(56, 189, 248, 0.3);
}

.brand-info h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: #f1f5f9;
  letter-spacing: 0.5px;
}

.brand-info p {
  margin: 2px 0 0;
  font-size: 11px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.nav-menu {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #94a3b8;
  font-size: 14px;
  font-weight: 500;
  overflow: hidden;
  white-space: nowrap;
}

/* Smooth fade for labels */
.nav-label {
    transition: opacity 0.3s ease 0.1s; /* Delayed fade in */
    opacity: 1;
}
.brand-info {
    transition: opacity 0.3s ease 0.1s;
    opacity: 1;
}
.user-info {
    transition: opacity 0.3s ease 0.1s;
    opacity: 1;
}

.sidebar.collapsed .nav-label,
.sidebar.collapsed .brand-info,
.sidebar.collapsed .user-info {
    opacity: 0;
    transition: opacity 0.1s ease; /* Fast fade out */
    pointer-events: none;
}

/* Rely on padding transition for centering, avoid justify-content jumps */
.sidebar.collapsed .nav-item {
    padding: 12px;
    /* justify-content: center; REMOVED to prevent jumps */
}

.nav-item:hover {
  background: rgba(30, 41, 59, 0.5);
  color: #e2e8f0;
}

.nav-item.active {
  background: linear-gradient(90deg, rgba(56, 189, 248, 0.1), transparent);
  color: #38bdf8;
  border-left: 2px solid #38bdf8;
  padding-left: 14px; /* Adjust for border */
}
.nav-item.active .nav-icon {
    filter: drop-shadow(0 0 8px rgba(56, 189, 248, 0.5));
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  overflow: hidden;
  white-space: nowrap;
}

.sidebar.collapsed .user-profile {
    justify-content: center;
}

.user-avatar {
  width: 36px;
  height: 36px;
  background: #1e293b;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #94a3b8;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.user-info {
  display: flex;
  flex-direction: column;
}

.username {
  font-size: 13px;
  font-weight: 600;
  color: #e2e8f0;
}

.logout-link {
  background: none;
  border: none;
  padding: 0;
  font-size: 11px;
  color: #64748b;
  cursor: pointer;
  text-align: left;
}
.logout-link:hover { color: #f87171; }


/* Main Content */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden; /* Header fixed, body scrolls */
}

.top-bar {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(148, 163, 184, 0.05);
}

.breadcrumbs {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: #94a3b8;
}
.crumb-current { color: #f1f5f9; font-weight: 500; }
.separator { opacity: 0.4; }

.github-link {
    font-size: 13px;
    color: #94a3b8;
    text-decoration: none;
    transition: color 0.2s;
}
.github-link:hover { color: #fff; }

.content-body {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 32px;
  position: relative;
}
@media (max-width: 768px) {
  .content-body { padding: 16px; }
}
@media (max-width: 480px) {
  .content-body { padding: 12px; }
}

.content-body.fixed-body {
  overflow: hidden;
  padding: 0; /* Remove padding so child view controls spacing */
  display: flex;
  flex-direction: column;
}

.view-container {
  width: 100%;
  min-width: 0;
}
.content-body.fixed-body .view-container {
  flex: 1;
  min-height: 0;
}

/* Animations */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

::-webkit-scrollbar {
  width: 8px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.2);
  border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.4);
}
</style>
