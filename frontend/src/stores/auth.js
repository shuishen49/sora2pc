import { defineStore } from 'pinia'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('adminToken') || '',
  }),
  getters: {
    isAuthed: (state) => !!state.token,
  },
  actions: {
    setToken(token) {
      this.token = token || ''
      if (this.token) {
        localStorage.setItem('adminToken', this.token)
      } else {
        localStorage.removeItem('adminToken')
      }
    },
    logout() {
      this.setToken('')
      window.location.href = '/login'
    },
  },
})

