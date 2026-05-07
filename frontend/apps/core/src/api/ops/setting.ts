import request from './request'

export const getSettings = () => request.get('/api/settings')
export const updateSettings = (data: Record<string, string>) => request.put('/api/settings', data)
export const getSettingByKey = (key: string) => request.get(`/settings/${key}`)
