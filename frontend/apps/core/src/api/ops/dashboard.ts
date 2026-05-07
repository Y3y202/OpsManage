import request from './request'

export const getDashboard = () => request.get('/api/dashboard')
export const getSystemInfo = () => request.get('/api/dashboard/system-info')
export const getSystemStatus = () => request.get('/api/dashboard/system-status')
