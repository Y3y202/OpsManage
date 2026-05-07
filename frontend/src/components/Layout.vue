<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const isCollapse = ref(false)

interface MenuItem {
  path: string
  icon: string
  title: string
  children?: { path: string; icon: string; title: string }[]
}

const menuItems: MenuItem[] = [
  { path: '/dashboard', icon: 'Odometer', title: '仪表盘' },
  { path: '', icon: 'Globe', title: '网站', children: [
    { path: '/websites', icon: 'Globe', title: '网站' },
    { path: '/certificates', icon: 'Key', title: '证书' }
  ]},
  { path: '/databases', icon: 'Coin', title: '数据库' },
  { path: '/containers', icon: 'Box', title: '容器' },
  { path: '/files', icon: 'FolderOpened', title: '文件' },
  { path: '/tasks', icon: 'Timer', title: '计划任务' },
  { path: '/security', icon: 'Lock', title: '安全' },
  { path: '/logs', icon: 'Document', title: '日志' },
  { path: '/settings', icon: 'Setting', title: '设置' },
  { path: '/versions', icon: 'Operation', title: '版本管理' }
]

// 面包屑标题
const currentTitle = computed(() => {
  for (const item of menuItems) {
    if (item.path === route.path) return item.title
    if (item.children) {
      const child = item.children.find(c => c.path === route.path)
      if (child) return child.title
    }
  }
  return ''
})

async function handleLogout() {
  await userStore.logout()
  router.push('/login')
}

function handleMenuSelect(index: string) {
  if (index && index !== route.path) {
    router.push(index)
  }
}

onMounted(() => { userStore.fetchProfile() })
</script>

<template>
  <el-container class="layout-root">
    <el-aside :width="isCollapse ? '72px' : '240px'" class="layout-sidebar">
      <div class="sidebar-brand">
        <svg width="32" height="32" viewBox="0 0 40 40" fill="none" class="brand-icon">
          <rect width="40" height="40" rx="12" fill="url(#g)"/>
          <path d="M12 20L18 14L26 22L20 28L12 20Z" fill="white" opacity="0.9"/>
          <path d="M18 20L22 16L28 22L22 28L18 20Z" fill="white" opacity="0.6"/>
          <defs><linearGradient id="g" x1="0" y1="0" x2="40" y2="40"><stop stop-color="#4f8cff"/><stop offset="1" stop-color="#6c5ce7"/></linearGradient></defs>
        </svg>
        <span v-if="!isCollapse" class="brand-text">OpsManage</span>
      </div>

      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        :collapse-transition="false"
        :default-openeds="['网站-group']"
        class="sidebar-menu"
        @select="handleMenuSelect"
      >
        <template v-for="item in menuItems" :key="item.path || item.title">
          <!-- 有子菜单 -->
          <el-sub-menu
            v-if="item.children && item.children.length"
            :index="item.title + '-group'"
            class="menu-item"
          >
            <template #title>
              <el-icon><component :is="item.icon" /></el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item
              v-for="child in item.children"
              :key="child.path"
              :index="child.path"
              @click="handleMenuSelect(child.path)"
            >
              <el-icon><component :is="child.icon" /></el-icon>
              <template #title>{{ child.title }}</template>
            </el-menu-item>
          </el-sub-menu>
          <!-- 普通菜单项 -->
          <el-menu-item
            v-else
            :index="item.path"
            class="menu-item"
            @click="handleMenuSelect(item.path)"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </template>
      </el-menu>

      <div class="sidebar-footer">
        <el-button
          :icon="isCollapse ? 'DArrowRight' : 'DArrowLeft'"
          text
          class="collapse-btn"
          @click="isCollapse = !isCollapse"
        />
      </div>
    </el-aside>

    <el-container>
      <el-header class="layout-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown trigger="click" @command="handleLogout">
            <div class="user-avatar-trigger">
              <div class="avatar-circle">
                {{ (userStore.user?.nickname || userStore.user?.username || 'U').charAt(0).toUpperCase() }}
              </div>
              <span class="username">{{ userStore.user?.nickname || userStore.user?.username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>
                  <div style="line-height: 1.4">
                    <div style="font-weight: 600">{{ userStore.user?.nickname || userStore.user?.username }}</div>
                    <div style="color: #86909c; font-size: 12px">{{ userStore.user?.role === 'admin' ? '管理员' : '用户' }}</div>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon> 退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout-root {
  height: 100vh;
}

/* ===== 侧栏 ===== */
.layout-sidebar {
  background: linear-gradient(180deg, #1a1f36 0%, #12152a 100%);
  display: flex;
  flex-direction: column;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  position: relative;
}
.layout-sidebar::after {
  content: '';
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 1px;
  background: rgba(255, 255, 255, 0.06);
}

.sidebar-brand {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 16px;
  flex-shrink: 0;
}
.brand-icon {
  flex-shrink: 0;
}
.brand-text {
  font-size: 18px;
  font-weight: 700;
  background: linear-gradient(135deg, #4f8cff, #a78bfa);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  white-space: nowrap;
}

.sidebar-menu {
  flex: 1;
  border: none !important;
  background: transparent !important;
  padding: 8px;
}

:deep(.menu-item) {
  border-radius: 10px !important;
  margin: 2px 0 !important;
  height: 44px !important;
  line-height: 44px !important;
  color: rgba(255, 255, 255, 0.6) !important;
  transition: all 0.2s ease;
}
/* 二级菜单项不限制高度 */
:deep(.menu-item.el-sub-menu) {
  height: auto !important;
  line-height: normal !important;
}
:deep(.menu-item.el-sub-menu > .el-sub-menu__title) {
  height: 44px !important;
  line-height: 44px !important;
  border-radius: 10px !important;
  color: rgba(255, 255, 255, 0.6) !important;
}
:deep(.menu-item.el-sub-menu > .el-sub-menu__title:hover) {
  background: rgba(255, 255, 255, 0.06) !important;
  color: #fff !important;
}
:deep(.menu-item.el-sub-menu > .el-sub-menu__title .el-sub-menu__icon-arrow) {
  color: rgba(255, 255, 255, 0.4) !important;
}
/* 子菜单项样式 */
:deep(.el-sub-menu .el-menu-item) {
  border-radius: 10px !important;
  margin: 2px 0 !important;
  height: 40px !important;
  line-height: 40px !important;
  padding-left: 48px !important;
  color: rgba(255, 255, 255, 0.5) !important;
}
:deep(.el-sub-menu .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.06) !important;
  color: #fff !important;
}
:deep(.el-sub-menu .el-menu-item.is-active) {
  background: linear-gradient(135deg, rgba(79, 140, 255, 0.2), rgba(108, 92, 231, 0.15)) !important;
  color: #fff !important;
  box-shadow: inset 3px 0 0 0 #4f8cff;
}
:deep(.menu-item:hover) {
  background: rgba(255, 255, 255, 0.06) !important;
  color: #fff !important;
}
:deep(.menu-item.is-active) {
  background: linear-gradient(135deg, rgba(79, 140, 255, 0.2), rgba(108, 92, 231, 0.15)) !important;
  color: #fff !important;
  box-shadow: inset 3px 0 0 0 #4f8cff;
}
:deep(.menu-item .el-icon) {
  font-size: 18px;
  margin-right: 4px;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}
.collapse-btn {
  width: 100%;
  color: rgba(255, 255, 255, 0.4) !important;
  justify-content: center;
}
.collapse-btn:hover {
  color: #fff !important;
}

/* ===== 顶栏 ===== */
.layout-header {
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  border-bottom: 1px solid var(--om-border);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
}
.header-left {
  display: flex;
  align-items: center;
}
.user-avatar-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: background 0.2s;
}
.user-avatar-trigger:hover {
  background: #f2f3f5;
}
.avatar-circle {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: linear-gradient(135deg, #4f8cff, #6c5ce7);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
}
.username {
  font-size: 14px;
  font-weight: 500;
  color: var(--om-text);
}

/* ===== 主内容 ===== */
.layout-main {
  background: var(--om-bg);
  padding: 24px;
  overflow-y: auto;
}
</style>
