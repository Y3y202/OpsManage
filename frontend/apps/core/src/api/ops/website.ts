import request from './request'

export const getWebsites = (params?: any) => request.get('/api/websites', { params })
export const createWebsite = (data: any) => request.post('/api/websites', data)
export const getWebsite = (id: number) => request.get(`/websites/${id}`)
export const updateWebsite = (id: number, data: any) => request.put(`/websites/${id}`, data)
export const deleteWebsite = (id: number) => request.delete(`/websites/${id}`)
export const startWebsite = (id: number) => request.post(`/websites/${id}/start`)
export const stopWebsite = (id: number) => request.post(`/websites/${id}/stop`)
