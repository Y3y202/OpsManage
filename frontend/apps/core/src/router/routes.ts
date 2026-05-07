import type { RouteRecordMainRaw } from '@fantastic-admin/types'
import type { RouteRecordRaw } from 'vue-router'
import pinia from '@/store'

function opsMenuRoutes(): RouteRecordRaw[] {
  return [
    {
      path: 'websites',
      name: 'websites',
      component: () => import('@/views/ops/Website.vue'),
      meta: { title: '网站', icon: 'i-lucide:globe' },
    },
    {
      path: 'certificates',
      name: 'certificates',
      component: () => import('@/views/ops/Certificate.vue'),
      meta: { title: '证书', icon: 'i-lucide:key-round' },
    },
    {
      path: 'databases',
      name: 'databases',
      component: () => import('@/views/ops/Database.vue'),
      meta: { title: '数据库', icon: 'i-lucide:database' },
    },
    {
      path: 'containers',
      name: 'containers',
      component: () => import('@/views/ops/Container.vue'),
      meta: { title: '容器', icon: 'i-lucide:box' },
    },
    {
      path: 'files',
      name: 'files',
      component: () => import('@/views/ops/FileManager.vue'),
      meta: { title: '文件', icon: 'i-lucide:folder-open' },
    },
    {
      path: 'tasks',
      name: 'tasks',
      component: () => import('@/views/ops/Task.vue'),
      meta: { title: '计划任务', icon: 'i-lucide:timer' },
    },
    {
      path: 'security',
      name: 'security',
      component: () => import('@/views/ops/Security.vue'),
      meta: { title: '安全', icon: 'i-lucide:shield-check' },
    },
    {
      path: 'logs',
      name: 'logs',
      component: () => import('@/views/ops/Log.vue'),
      meta: { title: '日志', icon: 'i-lucide:file-text' },
    },
    {
      path: 'settings',
      name: 'settings',
      component: () => import('@/views/ops/Setting.vue'),
      meta: { title: '设置', icon: 'i-lucide:settings' },
    },
    {
      path: 'versions',
      name: 'versions',
      component: () => import('@/views/ops/Version.vue'),
      meta: { title: '版本管理', icon: 'i-lucide:sliders-horizontal' },
    },
  ]
}

// 固定路由（默认路由）
const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login.vue'),
    meta: {
      title: '登录',
    },
  },
  {
    path: '/:all(.*)*',
    name: 'notFound',
    component: () => import('@/views/[...all].vue'),
    meta: {
      title: '找不到页面',
    },
  },
]

// 系统路由
const systemRoutes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/index.vue'),
    meta: {
      breadcrumb: false,
    },
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/ops/Dashboard.vue'),
        meta: {
          title: useAppSettingsStore(pinia).settings.app.home.title,
          icon: 'i-ant-design:dashboard-twotone',
          breadcrumb: false,
        },
      },
      ...opsMenuRoutes(),
      {
        path: 'reload',
        name: 'reload',
        component: () => import('@/views/reload.vue'),
        meta: {
          title: '重新加载中...',
          breadcrumb: false,
          menu: false,
        },
      },
    ],
  },
]

// 动态路由（导航菜单路由）
const asyncRoutes: RouteRecordMainRaw[] = [
  {
    meta: {
      title: '运维管理',
      icon: 'i-lucide:server-cog',
    },
    children: opsMenuRoutes().map(route => ({
      ...route,
      path: `/${route.path}`,
    })),
  },
]

export {
  asyncRoutes,
  constantRoutes,
  systemRoutes,
}
