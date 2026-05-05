import request from './request'

export interface ServiceVersionInfo {
  type: string
  name: string
  current_version: string
  installed: boolean
  running: boolean
  available_versions: string[]
  installed_versions: string[]
}

// 获取所有服务版本信息
export function getServiceVersions() {
  return request.get<ServiceVersionInfo[]>('/services/versions')
}

// 获取指定服务版本信息
export function getServiceVersionByType(type: string) {
  return request.get<ServiceVersionInfo>(`/services/versions/${type}`)
}

// 切换服务版本
export function switchServiceVersion(type: string, version: string) {
  return request.post(`/services/${type}/switch`, { version })
}
