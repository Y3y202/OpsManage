import request from './request'

// 获取活跃安装任务
export function getActiveTasks() {
  return request.get('/installer/tasks')
}

// 获取任务进度
export function getTaskProgress(taskId: string) {
  return request.get(`/installer/tasks/${taskId}`)
}

// 统一安装接口
export function installService(type: string, data?: any) {
  return request.post(`/installer/install/${type}`, data || {})
}

// Nginx 安装
export function installNginx() {
  return installService('nginx')
}

// MySQL 安装
export function installMySQL(rootPass?: string) {
  return installService('mysql', { root_pass: rootPass })
}

// PostgreSQL 安装
export function installPostgreSQL() {
  return installService('postgresql')
}

// Redis 安装
export function installRedis() {
  return installService('redis')
}

// SSE 进度监听
export function createProgressStream(taskId: string): EventSource {
  const token = localStorage.getItem('token')
  return new EventSource(`/sse/progress/${taskId}`)
}
