import type { EpPropMergeType } from 'element-plus/es/utils/vue/props/types'

export interface DockerDiskItem {
  count?: string
  size: string
  reclaimable?: string
}

export interface DockerOverview {
  containers_total: number
  containers_running: number
  images: number
  networks: number
  volumes: number
  compose: number
  compose_templates: number
  registries: number
  socket: string
  disk_usage: Record<string, DockerDiskItem>
}

export interface ContainerRow {
  id?: number | string
  container_id?: string
  shortId: string
  actionId: number | string
  name: string
  image: string
  status: string
  rawStatus?: string
  ports?: string
  created?: string
  size?: string
  statusType: EpPropMergeType<StringConstructor, 'success' | 'warning' | 'info' | 'danger' | 'primary', unknown>
}

export interface ContainerListResult {
  list: ContainerRow[]
  total: number
}

export interface CreateContainerForm {
  name: string
  image: string
  ports?: string
  volumes?: string
  env?: string
  command?: string
  restartPolicy?: string
  network?: string
}

const defaultDiskItem: DockerDiskItem = { count: '0', size: '0B', reclaimable: '0B' }

function statusType(status?: string): ContainerRow['statusType'] {
  const normalized = (status || '').toLowerCase()
  if (normalized.includes('running') || normalized.includes('up')) {
    return 'success'
  }
  if (normalized.includes('created') || normalized.includes('paused')) {
    return 'warning'
  }
  if (normalized.includes('exited') || normalized.includes('dead') || normalized.includes('error')) {
    return 'danger'
  }
  return 'info'
}

function displayStatus(status?: string, rawStatus?: string) {
  const normalized = (status || rawStatus || '').toLowerCase()
  if (normalized.includes('running') || normalized.includes('up')) {
    return 'running'
  }
  if (normalized.includes('created')) {
    return 'created'
  }
  if (normalized.includes('paused')) {
    return 'paused'
  }
  if (normalized.includes('exited') || normalized.includes('dead')) {
    return 'stopped'
  }
  return status || rawStatus || 'unknown'
}

export function normalizeDockerOverview(input: any = {}): DockerOverview {
  const diskUsage = input.disk_usage || {}
  return {
    containers_total: Number(input.containers_total || 0),
    containers_running: Number(input.containers_running || 0),
    images: Number(input.images || 0),
    networks: Number(input.networks || 0),
    volumes: Number(input.volumes || 0),
    compose: Number(input.compose || 0),
    compose_templates: Number(input.compose_templates || 0),
    registries: Number(input.registries || 0),
    socket: input.socket || 'unix:///var/run/docker.sock',
    disk_usage: {
      images: { ...defaultDiskItem, ...(diskUsage.images || {}) },
      containers: { ...defaultDiskItem, ...(diskUsage.containers || {}) },
      volumes: { ...defaultDiskItem, ...(diskUsage.volumes || {}) },
      build_cache: { ...defaultDiskItem, ...(diskUsage.build_cache || {}) },
    },
  }
}

export function normalizeContainerList(input: any = {}): ContainerListResult {
  const rawList = Array.isArray(input) ? input : Array.isArray(input.list) ? input.list : []
  const list = rawList.map((item: any): ContainerRow => {
    const dockerId = item.container_id || item.containerId || item.docker_id || ''
    const actionId = dockerId || item.id || item.name
    const shortId = dockerId ? String(dockerId).slice(0, 12) : String(item.id || '-')
    const status = displayStatus(item.status, item.raw_status)
    return {
      ...item,
      container_id: dockerId,
      actionId,
      shortId,
      status,
      rawStatus: item.raw_status || item.rawStatus || item.status,
      statusType: statusType(status),
    }
  })
  return { list, total: Number(input.total ?? list.length) }
}

export function buildCreateContainerPayload(form: CreateContainerForm) {
  const payload: Record<string, string> = {
    name: form.name.trim(),
    image: form.image.trim(),
  }
  const optional: Record<string, string | undefined> = {
    ports: form.ports,
    volumes: form.volumes,
    env: form.env,
    command: form.command,
    restart_policy: form.restartPolicy,
    network: form.network,
  }
  Object.entries(optional).forEach(([key, value]) => {
    const trimmed = value?.trim()
    if (trimmed) {
      payload[key] = trimmed
    }
  })
  return payload
}

export function sanitizeDockerResourceId(id: string | number) {
  return encodeURIComponent(String(id))
}
