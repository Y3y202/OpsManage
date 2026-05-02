import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getProfile, logout as logoutApi } from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const user = ref<any>(null)

  async function fetchProfile() {
    const res = await getProfile()
    user.value = res.data
  }

  async function logout() {
    try {
      await logoutApi()
    } catch {}
    localStorage.removeItem('token')
    user.value = null
  }

  return { user, fetchProfile, logout }
})
