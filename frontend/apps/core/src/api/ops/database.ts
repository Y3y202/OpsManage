import request from './request'

export const getDatabases = (params?: any) => request.get('/api/databases', { params })
export const createDatabase = (data: any) => request.post('/api/databases', data)
export const getDatabase = (id: number) => request.get(`/databases/${id}`)
export const updateDatabase = (id: number, data: any) => request.put(`/databases/${id}`, data)
export const deleteDatabase = (id: number) => request.delete(`/databases/${id}`)
export const startDatabase = (id: number) => request.post(`/databases/${id}/start`)
export const stopDatabase = (id: number) => request.post(`/databases/${id}/stop`)
