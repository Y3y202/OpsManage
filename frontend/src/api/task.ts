import request from './request'

export const getTasks = (params?: any) => request.get('/tasks', { params })
export const createTask = (data: any) => request.post('/tasks', data)
export const getTask = (id: number) => request.get(`/tasks/${id}`)
export const updateTask = (id: number, data: any) => request.put(`/tasks/${id}`, data)
export const deleteTask = (id: number) => request.delete(`/tasks/${id}`)
export const runTask = (id: number) => request.post(`/tasks/${id}/run`)
export const toggleTask = (id: number) => request.post(`/tasks/${id}/toggle`)
