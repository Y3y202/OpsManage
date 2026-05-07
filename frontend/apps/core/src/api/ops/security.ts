import request from './request'

export const getSecurityRules = (params?: any) => request.get('/api/security/rules', { params })
export const createSecurityRule = (data: any) => request.post('/api/security/rules', data)
export const getSecurityRule = (id: number) => request.get(`/security/rules/${id}`)
export const updateSecurityRule = (id: number, data: any) => request.put(`/security/rules/${id}`, data)
export const deleteSecurityRule = (id: number) => request.delete(`/security/rules/${id}`)
export const toggleSecurityRule = (id: number) => request.post(`/security/rules/${id}/toggle`)
