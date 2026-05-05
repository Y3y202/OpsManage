import request from './request'

// 获取活跃安装任务
export function getActiveTasks() {
  return request.get('/installer/tasks')
}

// 获取任务进度
export function getTaskProgress(taskId: string) {
  return request.get(`/installer/tasks/${taskId}`)
}

// 获取可用版本列表
export function getAvailableVersions(type: string) {
  return request.get(`/installer/versions/${type}`)
}

// 统一安装接口
export function installService(type: string, data?: any) {
  return request.post(`/installer/install/${type}`, data || {})
}

// Nginx 安装
export function installNginx(version?: string) {
  return installService('nginx', { version })
}

// MySQL 安装
export function installMySQL(version?: string, rootPass?: string) {
  return installService('mysql', { version, root_pass: rootPass })
}

// PostgreSQL 安装
export function installPostgreSQL(version?: string) {
  return installService('postgresql', { version })
}

// Redis 安装
export function installRedis(version?: string) {
  return installService('redis', { version })
}

// SSE 进度监听
export function createProgressStream(taskId: string): EventSource {
  return new EventSource(`/sse/progress/${taskId}`)
}

// 创建带认证的 SSE 连接
export function connectProgressStream(taskId: string, onMessage: (data: any) => void, onError?: (err: any) => void) {
  const eventSource = new EventSource(`/sse/progress/${taskId}`)

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onMessage(data)
    } catch (e) {
      console.error('Parse SSE data error:', e)
    }
  }

  eventSource.onerror = (err) => {
    console.error('SSE error:', err)
    if (onError) onError(err)
    eventSource.close()
  }

  return eventSource
}
