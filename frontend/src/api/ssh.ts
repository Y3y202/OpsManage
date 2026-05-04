import request from './request'

// SSH 账号管理
export const getSSHAccounts = (params?: any) => request.get('/security/ssh', { params })
export const createSSHAccount = (data: any) => request.post('/security/ssh', data)
export const getSSHAccount = (id: number) => request.get(`/security/ssh/${id}`)
export const getSSHAccountFull = (id: number) => request.get(`/security/ssh/${id}/full`)
export const updateSSHAccount = (id: number, data: any) => request.put(`/security/ssh/${id}`, data)
export const deleteSSHAccount = (id: number) => request.delete(`/security/ssh/${id}`)

// 连接测试
export const testSSHConnection = (id: number) => request.post(`/security/ssh/${id}/test`)

// 凭证管理
export const changeSSHCredential = (id: number, data: any) => request.post(`/security/ssh/${id}/credential`, data)

// 远程操作
export const changeRemotePassword = (id: number, data: any) => request.post(`/security/ssh/${id}/change-password`, data)
export const changeSSHPort = (id: number, data: any) => request.post(`/security/ssh/${id}/change-port`, data)
export const restartSSHD = (id: number) => request.post(`/security/ssh/${id}/restart`)
export const installSSHKey = (id: number) => request.post(`/security/ssh/${id}/install-key`)
export const executeSSHCommand = (id: number, data: any) => request.post(`/security/ssh/${id}/command`, data)

// sshd_config
export const getSSHdConfig = (id: number) => request.get(`/security/ssh/${id}/sshd-config`)
export const saveSSHdConfig = (id: number, data: any) => request.put(`/security/ssh/${id}/sshd-config`, data)

// 密钥生成
export const generateSSHKeyPair = () => request.post('/security/ssh/generate-key')
