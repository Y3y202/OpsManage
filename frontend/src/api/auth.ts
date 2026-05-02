import request from './request'

export const login = (data: { username: string; password: string; captcha_id: string; captcha_code: string }) =>
  request.post('/auth/login', data)

export const register = (data: { username: string; password: string; email?: string }) =>
  request.post('/auth/register', data)

export const logout = () => request.post('/auth/logout')

export const getCaptcha = () => request.get('/auth/captcha')

export const getProfile = () => request.get('/profile')

export const changePassword = (data: { old_password: string; new_password: string }) =>
  request.put('/password', data)
