import request from './request'
import { buildCreateContainerPayload, normalizeContainerList, normalizeDockerOverview, sanitizeDockerResourceId } from './container-adapter'
import type { CreateContainerForm } from './container-adapter'

// 容器
export const getContainers = async (params?: any) => normalizeContainerList(await request.get('/api/containers', { params }))
export const createContainer = (data: CreateContainerForm | any) => request.post('/api/containers', buildCreateContainerPayload(data))
export const getContainer = (id: number | string) => request.get(`/api/containers/${sanitizeDockerResourceId(id)}`)
export const deleteContainer = (id: number | string) => request.delete(`/api/containers/${sanitizeDockerResourceId(id)}`)
export const startContainer = (id: number | string) => request.post(`/api/containers/${sanitizeDockerResourceId(id)}/start`)
export const stopContainer = (id: number | string) => request.post(`/api/containers/${sanitizeDockerResourceId(id)}/stop`)
export const restartContainer = (id: number | string) => request.post(`/api/containers/${sanitizeDockerResourceId(id)}/restart`)
export const getContainerLogs = (id: number | string) => request.get(`/api/containers/${sanitizeDockerResourceId(id)}/logs`)

// 镜像
export const listImages = () => request.get('/api/containers/images')
export const pullImage = (image: string) => request.post('/api/containers/images/pull', { image: image.trim() })
export const removeImage = (id: string) => request.delete(`/api/containers/images/${sanitizeDockerResourceId(id)}`)

// 总览
export const getDockerOverview = async () => normalizeDockerOverview(await request.get('/api/containers/overview'))

// 网络
export const listDockerNetworks = () => request.get('/api/containers/networks')
export const removeNetwork = (id: string) => request.delete(`/api/containers/networks/${sanitizeDockerResourceId(id)}`)

// 存储卷
export const listDockerVolumes = () => request.get('/api/containers/volumes')
export const removeVolume = (id: string) => request.delete(`/api/containers/volumes/${sanitizeDockerResourceId(id)}`)

// 清理
export const pruneDocker = (type?: string) => request.post(`/api/containers/prune${type ? `?type=${encodeURIComponent(type)}` : ''}`)

// 镜像仓库
export const listRegistries = () => request.get('/api/containers/registries')
export const createRegistry = (data: any) => request.post('/api/containers/registries', data)
export const deleteRegistry = (id: number) => request.delete(`/api/containers/registries/${id}`)

// 编排项目
export const listComposeProjects = () => request.get('/api/containers/compose')
export const createComposeProject = (data: any) => request.post('/api/containers/compose', data)
export const deleteComposeProject = (id: number) => request.delete(`/api/containers/compose/${id}`)
export const startComposeProject = (id: number) => request.post(`/api/containers/compose/${id}/start`)
export const stopComposeProject = (id: number) => request.post(`/api/containers/compose/${id}/stop`)

// 编排模板
export const listComposeTemplates = () => request.get('/api/containers/templates')
export const createComposeTemplate = (data: any) => request.post('/api/containers/templates', data)
export const deleteComposeTemplate = (id: number) => request.delete(`/api/containers/templates/${id}`)

export { buildCreateContainerPayload, normalizeContainerList, normalizeDockerOverview }
export type { ContainerRow, DockerOverview, CreateContainerForm } from './container-adapter'
