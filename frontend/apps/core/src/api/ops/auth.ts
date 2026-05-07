import request from './request'

export const login = (data: { username: string; password: string; captcha_id: string; captcha_code: string }) =>
  request.post('/api/auth/login', data)

export const register = (data: { username: string; password: string; email?: string }) =>
  request.post('/api/auth/register', data)

export const logout = () => request.post('/api/auth/logout')

export const getCaptcha = () => request.get('/api/auth/captcha')

export const getProfile = () => request.get('/api/profile')

export const changePassword = (data: { old_password: string; new_password: string }) =>
  request.put('/api/password', data)
