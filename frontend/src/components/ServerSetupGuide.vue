<script setup>
import { ref, computed } from 'vue'
import { useGenerateStore } from '../stores/generate'

const emit = defineEmits(['complete'])
const generateStore = useGenerateStore()

const protocol = ref('http://')
const serverAddress = ref('')
const isValidating = ref(false)
const errorMessage = ref('')

// 完整URL
const fullUrl = computed(() => {
  const addr = serverAddress.value.trim()
  if (!addr) return ''
  return protocol.value + addr
})

// 格式化URL（确保没有尾部斜杠）
const formatUrl = () => {
  return fullUrl.value.replace(/\/+$/, '')
}

// 测试连接
const testConnection = async () => {
  if (!serverAddress.value.trim()) {
    errorMessage.value = '请输入服务器地址'
    return
  }

  const formattedUrl = formatUrl()
  isValidating.value = true
  errorMessage.value = ''

  try {
    // 尝试请求服务器健康检查
    const response = await fetch(`${formattedUrl}/health`, {
      method: 'GET',
      signal: AbortSignal.timeout(10000) // 10秒超时
    })

    if (response.ok || response.status === 401) {
      // 连接成功（401表示需要认证但服务器可达）
      saveAndContinue(formattedUrl)
    } else {
      errorMessage.value = `服务器响应异常 (${response.status})`
    }
  } catch (err) {
    if (err.name === 'TimeoutError') {
      errorMessage.value = '连接超时，请检查服务器地址'
    } else if (err.name === 'TypeError') {
      // 可能是CORS问题，但服务器可能仍然可用
      // 直接保存，让用户尝试
      saveAndContinue(formattedUrl)
    } else {
      errorMessage.value = `连接失败: ${err.message}`
    }
  } finally {
    isValidating.value = false
  }
}

// 跳过验证直接保存
const skipAndSave = () => {
  if (!serverAddress.value.trim()) {
    errorMessage.value = '请输入服务器地址'
    return
  }
  const formattedUrl = formatUrl()
  saveAndContinue(formattedUrl)
}

// 保存配置并继续
const saveAndContinue = (url) => {
  generateStore.setBaseUrl(url)
  // 标记已完成初始设置
  localStorage.setItem('sora_server_configured', 'true')
  emit('complete')
}

// 快捷输入示例
const quickFill = (proto, addr) => {
  protocol.value = proto
  serverAddress.value = addr
}
</script>

<template>
  <div class="setup-overlay">
    <div class="setup-container">
      <!-- Logo -->
      <div class="setup-header">
        <div class="logo-box">S2</div>
        <h1>Sora2 API</h1>
        <p class="subtitle">首次使用配置</p>
      </div>

      <!-- 引导内容 -->
      <div class="setup-content">
        <div class="step-indicator">
          <div class="step active">
            <span class="step-number">1</span>
            <span class="step-label">服务器配置</span>
          </div>
        </div>

        <div class="form-section">
          <label class="form-label">
            API 服务器地址
            <span class="required">*</span>
          </label>
          <div class="input-group">
            <select v-model="protocol" class="protocol-select" :disabled="isValidating">
              <option value="http://">http://</option>
              <option value="https://">https://</option>
            </select>
            <input
              v-model="serverAddress"
              type="text"
              class="form-input"
              placeholder="127.0.0.1:8000 或 your-server.com"
              @keyup.enter="testConnection"
              :disabled="isValidating"
            />
          </div>

          <!-- 预览完整地址 -->
          <div v-if="serverAddress.trim()" class="url-preview">
            完整地址: <code>{{ fullUrl }}</code>
          </div>

          <!-- 快捷示例 -->
          <div class="quick-examples">
            <span class="examples-label">快捷填入:</span>
            <button class="example-btn" @click="quickFill('http://', '127.0.0.1:8000')">
              本地 HTTP
            </button>
            <button class="example-btn" @click="quickFill('https://', 'api.example.com')">
              远程 HTTPS
            </button>
          </div>

          <!-- 错误提示 -->
          <div v-if="errorMessage" class="error-message">
            {{ errorMessage }}
          </div>

          <!-- 说明 -->
          <div class="info-box">
            <div class="info-icon">i</div>
            <div class="info-text">
              <p><strong>HTTP</strong> - 适合本地或内网服务器（如 <code>127.0.0.1:8000</code>）</p>
              <p><strong>HTTPS</strong> - 适合有SSL证书的远程服务器</p>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <button
            class="btn btn-primary"
            @click="testConnection"
            :disabled="!serverAddress.trim() || isValidating"
          >
            <span v-if="isValidating" class="loading-spinner"></span>
            {{ isValidating ? '测试连接中...' : '测试连接并保存' }}
          </button>
          <button
            class="btn btn-secondary"
            @click="skipAndSave"
            :disabled="!serverAddress.trim() || isValidating"
          >
            跳过测试直接保存
          </button>
        </div>
      </div>

      <!-- 底部信息 -->
      <div class="setup-footer">
        <p>配置完成后可在 "系统配置" 中修改</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.setup-overlay {
  position: fixed;
  inset: 0;
  background: radial-gradient(circle at 30% 30%, #0f172a, #020617 80%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.setup-container {
  width: 100%;
  max-width: 480px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 20px;
  backdrop-filter: blur(20px);
  padding: 40px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.setup-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo-box {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #38bdf8, #818cf8);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: #fff;
  font-size: 20px;
  margin: 0 auto 16px;
  box-shadow: 0 0 30px rgba(56, 189, 248, 0.4);
}

.setup-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #f1f5f9;
}

.subtitle {
  margin: 8px 0 0;
  font-size: 14px;
  color: #64748b;
}

.step-indicator {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

.step {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(56, 189, 248, 0.1);
  border-radius: 20px;
  border: 1px solid rgba(56, 189, 248, 0.3);
}

.step-number {
  width: 24px;
  height: 24px;
  background: #38bdf8;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #0f172a;
}

.step-label {
  font-size: 13px;
  color: #38bdf8;
  font-weight: 500;
}

.form-section {
  margin-bottom: 24px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #94a3b8;
  margin-bottom: 8px;
}

.required {
  color: #f87171;
}

.input-group {
  display: flex;
  gap: 0;
}

.protocol-select {
  padding: 14px 12px;
  background: rgba(30, 41, 59, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-right: none;
  border-radius: 10px 0 0 10px;
  color: #38bdf8;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  outline: none;
  min-width: 100px;
}

.protocol-select:focus {
  border-color: #38bdf8;
}

.protocol-select option {
  background: #1e293b;
  color: #f1f5f9;
}

.input-group .form-input {
  flex: 1;
  border-radius: 0 10px 10px 0;
}

.form-input {
  width: 100%;
  padding: 14px 16px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 10px;
  color: #f1f5f9;
  font-size: 14px;
  transition: all 0.2s;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: #38bdf8;
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.1);
}

.form-input::placeholder {
  color: #64748b;
}

.form-input:disabled,
.protocol-select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.url-preview {
  margin-top: 8px;
  font-size: 12px;
  color: #64748b;
}

.url-preview code {
  background: rgba(56, 189, 248, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: monospace;
  color: #38bdf8;
}

.quick-examples {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.examples-label {
  font-size: 12px;
  color: #64748b;
}

.example-btn {
  padding: 4px 10px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 6px;
  color: #94a3b8;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.example-btn:hover {
  background: rgba(56, 189, 248, 0.1);
  border-color: rgba(56, 189, 248, 0.3);
  color: #38bdf8;
}

.error-message {
  margin-top: 12px;
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  color: #f87171;
  font-size: 13px;
}

.info-box {
  display: flex;
  gap: 12px;
  margin-top: 16px;
  padding: 14px;
  background: rgba(30, 41, 59, 0.4);
  border-radius: 10px;
  border: 1px solid rgba(148, 163, 184, 0.1);
}

.info-icon {
  width: 20px;
  height: 20px;
  background: rgba(56, 189, 248, 0.2);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #38bdf8;
  flex-shrink: 0;
}

.info-text {
  flex: 1;
}

.info-text p {
  margin: 0;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}

.info-text p + p {
  margin-top: 6px;
}

.info-text code {
  background: rgba(56, 189, 248, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 11px;
  color: #38bdf8;
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.btn {
  padding: 14px 24px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #38bdf8, #818cf8);
  color: white;
  box-shadow: 0 4px 15px rgba(56, 189, 248, 0.3);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(56, 189, 248, 0.4);
}

.btn-secondary {
  background: rgba(30, 41, 59, 0.6);
  color: #94a3b8;
  border: 1px solid rgba(148, 163, 184, 0.2);
}

.btn-secondary:hover:not(:disabled) {
  background: rgba(30, 41, 59, 0.8);
  color: #f1f5f9;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.setup-footer {
  text-align: center;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
}

.setup-footer p {
  margin: 0;
  font-size: 12px;
  color: #64748b;
}
</style>
