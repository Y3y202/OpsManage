import request from './request'

export const getFirewallStatus = () => request.get('/api/security/firewall/status')
export const addFirewallRule = (data: any) => request.post('/api/security/firewall/rules', data)
export const deleteFirewallRule = (id: number) => request.delete(`/security/firewall/rules/${id}`)
export const getFirewallPorts = () => request.get('/api/security/firewall/ports')
export const restartFirewall = () => request.post('/api/security/firewall/restart')
