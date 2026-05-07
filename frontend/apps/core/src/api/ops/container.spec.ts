import { describe, expect, it } from 'vitest'
import { buildCreateContainerPayload, normalizeContainerList, normalizeDockerOverview, sanitizeDockerResourceId } from './container-adapter'

describe('container api adapters', () => {
  it('normalizes list payload from backend response and supports docker container id fallback', () => {
    const result = normalizeContainerList({
      list: [
        { container_id: 'abcdef123456', name: 'nginx', image: 'nginx:latest', status: 'running', ports: '0.0.0.0:8080->80/tcp' },
        { id: 2, container_id: '', name: 'draft', image: 'redis:7', status: 'created' },
      ],
      total: 2,
    })

    expect(result.total).toBe(2)
    expect(result.list[0].actionId).toBe('abcdef123456')
    expect(result.list[0].shortId).toBe('abcdef123456')
    expect(result.list[0].statusType).toBe('success')
    expect(result.list[1].actionId).toBe(2)
    expect(result.list[1].statusType).toBe('warning')
  })

  it('builds create container payload without empty fields and supports command/restart policy', () => {
    expect(buildCreateContainerPayload({
      name: 'web',
      image: 'nginx:latest',
      ports: '8080:80\n',
      volumes: '',
      env: 'NODE_ENV=production',
      command: 'nginx -g daemon off;',
      restartPolicy: 'unless-stopped',
    })).toEqual({
      name: 'web',
      image: 'nginx:latest',
      ports: '8080:80',
      env: 'NODE_ENV=production',
      command: 'nginx -g daemon off;',
      restart_policy: 'unless-stopped',
    })
  })

  it('normalizes overview with missing disk usage entries', () => {
    const overview = normalizeDockerOverview({ containers_total: 3, containers_running: 1 })
    expect(overview.containers_total).toBe(3)
    expect(overview.disk_usage.images.size).toBe('0B')
    expect(overview.socket).toBe('unix:///var/run/docker.sock')
  })

  it('encodes docker resource ids for safe path usage', () => {
    expect(sanitizeDockerResourceId('nginx:latest')).toBe('nginx%3Alatest')
    expect(sanitizeDockerResourceId('sha256:abc/def')).toBe('sha256%3Aabc%2Fdef')
  })
})
