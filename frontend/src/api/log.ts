import request from './request'

export const getLogs = (params?: any) => request.get('/logs', { params })
export const getLogSources = () => request.get('/logs/sources')
export const clearLogs = (source?: string) => request.delete('/logs/clear', { params: { source } })
export const getSystemLogs = (type: string, lines?: number) =>
  request.get('/logs/system', { params: { type, lines: lines || 200 } })
