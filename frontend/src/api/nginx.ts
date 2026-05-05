import request from './request'

// ========== Nginx 服务管理 ==========

export function getNginxStatus() {
  return request.get('/nginx/status')
}

export function getNginxOverview() {
  return request.get('/nginx/overview')
}

export function installNginx() {
  return request.post('/nginx/install')
}

export function nginxService(action: string) {
  return request.post('/nginx/service', { action })
}

export function testNginxConfig() {
  return request.get('/nginx/test')
}

export function importNginxSites() {
  return request.post('/nginx/import')
}

// ========== Nginx 站点管理 ==========

export function listNginxSites(params: any) {
  return request.get('/nginx/sites', { params })
}

export function createNginxSite(data: any) {
  return request.post('/nginx/sites', data)
}

export function getNginxSite(id: number) {
  return request.get(`/nginx/sites/${id}`)
}

export function updateNginxSite(id: number, data: any) {
  return request.put(`/nginx/sites/${id}`, data)
}

export function deleteNginxSite(id: number) {
  return request.delete(`/nginx/sites/${id}`)
}

export function nginxSiteAction(id: number, action: string) {
  return request.post(`/nginx/sites/${id}/${action}`)
}

export function reloadNginxSite(id: number) {
  return request.post(`/nginx/sites/${id}/reload`)
}

// ========== SSL 管理 ==========

export function manageNginxSSL(id: number, data: any) {
  return request.post(`/nginx/sites/${id}/ssl`, data)
}

// ========== 配置编辑 ==========

export function getNginxSiteConfig(id: number) {
  return request.get(`/nginx/sites/${id}/config`)
}

export function saveNginxSiteConfig(id: number, content: string) {
  return request.put(`/nginx/sites/${id}/config`, { content })
}

// ========== 日志查看 ==========

export function getNginxSiteLogs(id: number, type: string, lines: number) {
  return request.get(`/nginx/sites/${id}/logs`, { params: { type, lines } })
}
