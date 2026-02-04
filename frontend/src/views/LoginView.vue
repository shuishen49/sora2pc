<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { postJson } from '../api/client'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

const submit = async () => {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const { data } = await postJson('/api/login', {
      username: username.value,
      password: password.value,
    })
    if (data?.success && data?.token) {
      auth.setToken(data.token)
      router.push('/')
    } else {
      error.value = data?.message || '登录失败'
    }
  } catch (e) {
    error.value = e?.data?.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="bg-layer" />
    <div class="shell">
      <header class="brand">
        <div class="brand-mark">
          <span class="dot" />
          <span class="dot dot-2" />
        </div>
        <div class="brand-text">
          <span class="eyebrow">Sora · Control Hub</span>
          <h1>Sora2 Console</h1>
          <p>登录统一控制台，管理 Token、配置与生成任务。</p>
        </div>
      </header>

      <main class="card">
        <form class="form" @submit.prevent="submit">
          <div class="field">
            <label>账户</label>
            <input
              v-model="username"
              type="text"
              placeholder="admin"
              required
            />
          </div>
          <div class="field">
            <label>密码</label>
            <input
              v-model="password"
              type="password"
              placeholder="admin"
              required
            />
          </div>
          <button type="submit" :disabled="loading">
            <span v-if="!loading">进入控制台</span>
            <span v-else class="btn-loading">
              <span class="spinner" /> 正在登录…
            </span>
          </button>
          <p v-if="error" class="error">
            {{ error }}
          </p>
          <p class="hint">默认账号密码：admin / admin · 登录后请尽快修改。</p>
        </form>
      </main>
    </div>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at 18% 20%, #020617, #020617 36%, #020617 70%, #020617);
  color: #e5e7eb;
  position: relative;
  overflow: hidden;
}

.bg-layer::before,
.bg-layer::after {
  content: '';
  position: absolute;
  border-radius: 999px;
  filter: blur(60px);
  opacity: 0.9;
}

.bg-layer::before {
  width: 320px;
  height: 320px;
  background: conic-gradient(from 140deg, #38bdf8, #6366f1, #a855f7);
  top: -80px;
  left: -40px;
}

.bg-layer::after {
  width: 360px;
  height: 360px;
  background: radial-gradient(circle at 30% 0%, #22c55e, transparent 60%),
    radial-gradient(circle at 80% 40%, #0ea5e9, transparent 60%);
  bottom: -120px;
  right: -40px;
}

.shell {
  position: relative;
  z-index: 1;
  width: min(960px, 100%);
  padding: 32px 20px;
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.9fr);
  gap: 24px;
}

.brand {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 16px;
}

.brand-mark {
  width: 64px;
  height: 64px;
  border-radius: 22px;
  background: radial-gradient(circle at 25% 0, #38bdf8, transparent 55%),
    radial-gradient(circle at 0 100%, #22c55e, transparent 60%),
    radial-gradient(circle at 100% 0, #6366f1, transparent 60%);
  border: 1px solid rgba(148, 163, 184, 0.4);
  box-shadow:
    0 18px 60px rgba(15, 23, 42, 0.9),
    0 0 0 1px rgba(15, 23, 42, 0.8);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.dot {
  width: 9px;
  height: 9px;
  border-radius: 999px;
  background: #e5e7eb;
  box-shadow: 0 0 18px rgba(248, 250, 252, 0.9);
}

.dot-2 {
  position: absolute;
  right: 16px;
  bottom: 14px;
  width: 6px;
  height: 6px;
  opacity: 0.6;
}

.brand-text h1 {
  margin: 4px 0;
  font-size: 30px;
  font-weight: 600;
  letter-spacing: 0.03em;
}

.brand-text h1 span {
  background: linear-gradient(120deg, #38bdf8, #6366f1, #a855f7);
  -webkit-background-clip: text;
  color: transparent;
}

.eyebrow {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: #9ca3af;
}

.brand-text p {
  margin: 0;
  font-size: 12px;
  color: #9ca3af;
  max-width: 320px;
}

.card {
  border-radius: 24px;
  border: 1px solid rgba(148, 163, 184, 0.6);
  background: radial-gradient(circle at 0 0, rgba(148, 163, 184, 0.2), transparent 55%),
    rgba(15, 23, 42, 0.86);
  backdrop-filter: blur(26px);
  box-shadow:
    0 22px 80px rgba(15, 23, 42, 0.95),
    0 0 0 1px rgba(15, 23, 42, 0.9);
  padding: 22px 20px 20px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 12px;
  font-weight: 500;
  color: #e5e7eb;
}

.field input {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.7);
  background: rgba(15, 23, 42, 0.9);
  color: #e5e7eb;
  padding: 9px 11px;
  font-size: 13px;
  outline: none;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background 0.16s ease;
}

.field input::placeholder {
  color: #6b7280;
}

.field input:focus-visible {
  border-color: #38bdf8;
  box-shadow: 0 0 0 1px rgba(56, 189, 248, 0.4);
  background: radial-gradient(circle at 0 0, rgba(56, 189, 248, 0.18), transparent 55%),
    rgba(15, 23, 42, 0.96);
}

button {
  margin-top: 4px;
  border-radius: 999px;
  border: none;
  outline: none;
  cursor: pointer;
  background: linear-gradient(120deg, #38bdf8, #6366f1, #a855f7);
  color: white;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  box-shadow:
    0 16px 45px rgba(56, 189, 248, 0.35),
    0 0 0 1px rgba(15, 23, 42, 0.85);
  transition:
    transform 0.14s ease,
    box-shadow 0.14s ease,
    filter 0.14s ease,
    opacity 0.14s ease;
}

button:hover {
  transform: translateY(-1px);
  filter: brightness(1.08);
}

button:active {
  transform: translateY(0) scale(0.98);
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.9);
}

button:disabled {
  opacity: 0.65;
  cursor: wait;
  transform: none;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.9);
}

.btn-loading {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.spinner {
  width: 14px;
  height: 14px;
  border-radius: 999px;
  border-width: 2px;
  border-style: solid;
  border-color: rgba(248, 250, 252, 0.35);
  border-top-color: #f9fafb;
  animation: spin 0.8s linear infinite;
}

.error {
  margin: 2px 0 0;
  font-size: 12px;
  color: #fb7185;
}

.hint {
  margin: 6px 0 0;
  font-size: 11px;
  color: #9ca3af;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .shell {
    grid-template-columns: minmax(0, 1fr);
    padding-inline: 16px;
  }

  .brand {
    text-align: left;
  }

  .brand-text p {
    max-width: 100%;
  }
}
</style>

