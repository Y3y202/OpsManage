import request from './request'

export const getDashboard = () => request.get('/dashboard')
export const getSystemInfo = () => request.get('/dashboard/system-info')
export const getSystemStatus = () => request.get('/dashboard/system-status')
