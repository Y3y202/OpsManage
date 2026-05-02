import request from './request'

export const getContainers = (params?: any) => request.get('/containers', { params })
export const createContainer = (data: any) => request.post('/containers', data)
export const getContainer = (id: number) => request.get(`/containers/${id}`)
export const deleteContainer = (id: number) => request.delete(`/containers/${id}`)
export const startContainer = (id: number) => request.post(`/containers/${id}/start`)
export const stopContainer = (id: number) => request.post(`/containers/${id}/stop`)
export const restartContainer = (id: number) => request.post(`/containers/${id}/restart`)
export const getContainerLogs = (id: number) => request.get(`/containers/${id}/logs`)
export const getImages = () => request.get('/containers/images')
export const pullImage = (image: string) => request.post('/containers/images/pull', { image })
