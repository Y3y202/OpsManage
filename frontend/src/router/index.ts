import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue')
  },
  {
    path: '/',
    component: () => import('@/components/Layout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue'), meta: { title: '仪表盘' } },
      { path: 'websites', name: 'Website', component: () => import('@/views/Website.vue'), meta: { title: '网站' } },
      { path: 'databases', name: 'Database', component: () => import('@/views/Database.vue'), meta: { title: '数据库' } },
      { path: 'containers', name: 'Container', component: () => import('@/views/Container.vue'), meta: { title: '容器' } },
      { path: 'files', name: 'FileManager', component: () => import('@/views/FileManager.vue'), meta: { title: '文件' } },
      { path: 'tasks', name: 'Task', component: () => import('@/views/Task.vue'), meta: { title: '计划任务' } },
      { path: 'security', name: 'Security', component: () => import('@/views/Security.vue'), meta: { title: '安全' } },
      { path: 'logs', name: 'Log', component: () => import('@/views/Log.vue'), meta: { title: '日志' } },
      { path: 'settings', name: 'Setting', component: () => import('@/views/Setting.vue'), meta: { title: '设置' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) {
    return { name: 'Login' }
  }
  if (to.name === 'Login' && token) {
    return { name: 'Dashboard' }
  }
})

export default router
