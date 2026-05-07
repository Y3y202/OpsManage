import request from './request'

// ========== 数据库服务管理 ==========

export function getDBServiceStatus() {
  return request.get('/api/databases/services/status')
}

export function listDBInstances() {
  return request.get('/api/databases/instances')
}

export function createDBInstance(data: any) {
  return request.post('/api/databases/instances', data)
}

export function getDBInstance(id: number) {
  return request.get(`/databases/instances/${id}`)
}

export function dbInstanceAction(id: number, action: string) {
  return request.post(`/databases/instances/${id}/${action}`)
}

export function getDBInstanceConfig(id: number) {
  return request.get(`/databases/instances/${id}/config`)
}

export function saveDBInstanceConfig(id: number, content: string) {
  return request.put(`/databases/instances/${id}/config`, { content })
}

export function getDBInstanceStats(id: number) {
  return request.get(`/databases/instances/${id}/stats`)
}

// ========== 数据库管理 ==========

export function listDBDatabases(instanceId: number) {
  return request.get(`/databases/instances/${instanceId}/databases`)
}

export function createDBDatabase(data: any) {
  return request.post(`/databases/instances/${data.instance_id}/databases`, data)
}

export function deleteDBDatabase(id: number) {
  return request.delete(`/databases/databases/${id}`)
}

export function syncDBDatabases(instanceId: number) {
  return request.post(`/databases/instances/${instanceId}/databases/sync`)
}

// ========== 用户管理 ==========

export function listDBUsers(instanceId: number) {
  return request.get(`/databases/instances/${instanceId}/users`)
}

export function createDBUser(instanceId: number, data: any) {
  return request.post(`/databases/instances/${instanceId}/users`, data)
}

export function updateDBUserPassword(userId: number, password: string) {
  return request.put(`/databases/users/${userId}/password`, { password })
}

export function deleteDBUser(id: number) {
  return request.delete(`/databases/users/${id}`)
}

// ========== 实例密码管理 ==========

export function updateDBInstancePassword(instanceId: number, password: string) {
  return request.put(`/databases/instances/${instanceId}/password`, { password })
}

// ========== 备份管理 ==========

export function listDBBackups(instanceId: number) {
  return request.get(`/databases/instances/${instanceId}/backups`)
}

export function createDBBackup(instanceId: number, dbName: string) {
  return request.post(`/databases/instances/${instanceId}/backups`, { instance_id: instanceId, db_name: dbName })
}

export function restoreDBBackup(id: number) {
  return request.post(`/databases/backups/${id}/restore`)
}
