<script setup>
import { onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import ServerSetupGuide from './components/ServerSetupGuide.vue'

// 是否已完成服务器配置
const isConfigured = ref(false)

const checkConfigured = async () => {
  // 优先从 Wails 后端的本地 config 文件判断
  if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetBaseURL) {
    try {
      const url = await window.go.main.App.GetBaseURL()
      if (url) {
        isConfigured.value = true
        return
      }
    } catch (e) {
      console.error('GetBaseURL failed:', e)
    }
  }

  // 兜底逻辑：旧版 localStorage 标记
  isConfigured.value =
    !!localStorage.getItem('sora_server_configured') && !!localStorage.getItem('sora_base_url')
}

onMounted(() => {
  checkConfigured()
})

const onSetupComplete = () => {
  isConfigured.value = true
}
</script>

<template>
  <ServerSetupGuide v-if="!isConfigured" @complete="onSetupComplete" />
  <RouterView v-else />
</template>

