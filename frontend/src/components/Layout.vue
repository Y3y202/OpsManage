<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const isCollapse = ref(false)

const menuItems = [
  { path: '/dashboard', icon: 'Odometer', title: '仪表盘' },
  { path: '/websites', icon: 'Globe', title: '网站' },
  { path: '/databases', icon: 'Coin', title: '数据库' },
  { path: '/containers', icon: 'Box', title: '容器' },
  { path: '/files', icon: 'FolderOpened', title: '文件' },
  { path: '/tasks', icon: 'Timer', title: '计划任务' },
  { path: '/security', icon: 'Lock', title: '安全' },
  { path: '/logs', icon: 'Document', title: '日志' },
  { path: '/settings', icon: 'Setting', title: '设置' }
]

async function handleLogout() {
  await userStore.logout()
  router.push('/login')
}

onMounted(() => {
  userStore.fetchProfile()
})
</script>

<template>
  <el-container style="height: 100vh">
    <el-aside :width="isCollapse ? '64px' : '200px'" style="background: #304156; transition: width 0.3s">
      <div style="height: 60px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 16px; font-weight: bold">
        {{ isCollapse ? 'OM' : 'OpsManage' }}
      </div>
      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        router
      >
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.title }}</template>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header style="display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #eee; background: #fff">
        <el-icon style="cursor: pointer; font-size: 20px" @click="isCollapse = !isCollapse">
          <Fold v-if="!isCollapse" />
          <Expand v-else />
        </el-icon>
        <div style="display: flex; align-items: center; gap: 12px">
          <span>{{ userStore.user?.nickname || userStore.user?.username }}</span>
          <el-button type="danger" size="small" @click="handleLogout">退出</el-button>
        </div>
      </el-header>
      <el-main style="background: #f0f2f5; padding: 20px">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>
