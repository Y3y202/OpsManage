import request from './request'

export const getLogs = (params?: any) => request.get('/api/logs', { params })
export const getLogSources = () => request.get('/api/logs/sources')
export const clearLogs = (source?: string) => request.delete('/api/logs/clear', { params: { source } })
export const getSystemLogs = (type: string, lines?: number) =>
  request.get('/api/logs/system', { params: { type, lines: lines || 200 } })

// SSH 登录日志
export const getSSHLogs = (params?: any) => request.get('/api/logs/ssh', { params })
