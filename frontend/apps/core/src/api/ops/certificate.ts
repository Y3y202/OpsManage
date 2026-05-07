import request from './request'

// 获取证书列表
export function getCertificates() {
  return request.get('/api/certificates')
}

// 获取证书详情
export function getCertificate(id: number) {
  return request.get(`/certificates/${id}`)
}

// 申请 Let's Encrypt 证书
export function applyLetsencrypt(data: {
  domain: string
  email: string
  standalone: boolean
  web_root?: string
}) {
  return request.post('/api/certificates/apply', data)
}

// 上传自定义证书
export function uploadCertificate(data: {
  name: string
  domain: string
  cert: string
  key: string
  chain?: string
}) {
  return request.post('/api/certificates', data)
}

// 删除证书
export function deleteCertificate(id: number) {
  return request.delete(`/certificates/${id}`)
}

// 续签证书
export function renewCertificate(id: number) {
  return request.post(`/certificates/${id}/renew`)
}

// 切换自动续签
export function toggleAutoRenew(id: number) {
  return request.post(`/certificates/${id}/auto-renew`)
}

// 获取证书文件内容
export function getCertificateContent(id: number, field: string) {
  return request.get(`/certificates/${id}/content/${field}`)
}

// 应用证书到站点
export function applyCertToSite(id: number, website_id: number) {
  return request.post(`/certificates/${id}/apply-site`, { website_id })
}

// 获取可绑定的站点列表
export function getSitesForCert() {
  return request.get('/api/certificates/sites')
}
