import request from './request'

export const getSettings = () => request.get('/settings')
export const updateSettings = (data: Record<string, string>) => request.put('/settings', data)
export const getSettingByKey = (key: string) => request.get(`/settings/${key}`)
