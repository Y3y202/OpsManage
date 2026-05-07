import request from './request'

// 容器
export const getContainers = (params?: any) => request.get('/api/containers', { params })
export const createContainer = (data: any) => request.post('/api/containers', data)
export const getContainer = (id: number) => request.get(`/containers/${id}`)
export const deleteContainer = (id: number) => request.delete(`/containers/${id}`)
export const startContainer = (id: number) => request.post(`/containers/${id}/start`)
export const stopContainer = (id: number) => request.post(`/containers/${id}/stop`)
export const restartContainer = (id: number) => request.post(`/containers/${id}/restart`)
export const getContainerLogs = (id: number) => request.get(`/containers/${id}/logs`)

// 镜像
export const listImages = () => request.get('/api/containers/images')
export const pullImage = (image: string) => request.post('/api/containers/images/pull', { image })
export const removeImage = (id: string) => request.delete(`/containers/images/${id}`)

// 总览
export const getDockerOverview = () => request.get('/api/containers/overview')

// 网络
export const listDockerNetworks = () => request.get('/api/containers/networks')
export const removeNetwork = (id: string) => request.delete(`/containers/networks/${id}`)

// 存储卷
export const listDockerVolumes = () => request.get('/api/containers/volumes')
export const removeVolume = (id: string) => request.delete(`/containers/volumes/${id}`)

// 清理
export const pruneDocker = (type?: string) => request.post(`/containers/prune${type ? `?type=${type}` : ''}`)

// 镜像仓库
export const listRegistries = () => request.get('/api/containers/registries')
export const createRegistry = (data: any) => request.post('/api/containers/registries', data)
export const deleteRegistry = (id: number) => request.delete(`/containers/registries/${id}`)

// 编排项目
export const listComposeProjects = () => request.get('/api/containers/compose')
export const createComposeProject = (data: any) => request.post('/api/containers/compose', data)
export const deleteComposeProject = (id: number) => request.delete(`/containers/compose/${id}`)
export const startComposeProject = (id: number) => request.post(`/containers/compose/${id}/start`)
export const stopComposeProject = (id: number) => request.post(`/containers/compose/${id}/stop`)

// 编排模板
export const listComposeTemplates = () => request.get('/api/containers/templates')
export const createComposeTemplate = (data: any) => request.post('/api/containers/templates', data)
export const deleteComposeTemplate = (id: number) => request.delete(`/containers/templates/${id}`)
