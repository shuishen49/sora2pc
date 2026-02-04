import { ref } from 'vue'

const toasts = ref([])

export const useToast = () => {
  const add = (message, type = 'info', duration = 3000) => {
    const id = Date.now()
    toasts.value.push({ id, message, type })
    setTimeout(() => remove(id), duration)
  }

  const remove = (id) => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  const success = (msg, duration) => add(msg, 'success', duration)
  const error = (msg, duration) => add(msg, 'error', duration)
  const warning = (msg, duration) => add(msg, 'warning', duration)

  return { toasts, add, remove, success, error, warning }
}
