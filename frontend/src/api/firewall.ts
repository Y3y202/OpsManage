import request from './request'

export const getFirewallStatus = () => request.get('/security/firewall/status')
export const addFirewallRule = (data: any) => request.post('/security/firewall/rules', data)
export const deleteFirewallRule = (id: number) => request.delete(`/security/firewall/rules/${id}`)
export const getFirewallPorts = () => request.get('/security/firewall/ports')
export const restartFirewall = () => request.post('/security/firewall/restart')
