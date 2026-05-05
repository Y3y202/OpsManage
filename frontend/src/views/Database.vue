<template>
  <div class="database-page">
    <!-- 数据库服务概览 -->
    <div class="services-overview">
      <div class="service-card" v-for="svc in services" :key="svc.type">
        <div class="service-header">
          <div class="service-icon" :class="svc.type">
            <el-icon v-if="svc.type === 'mysql'"><Coin /></el-icon>
            <el-icon v-else-if="svc.type === 'postgresql'"><DataLine /></el-icon>
            <el-icon v-else><TrendCharts /></el-icon>
          </div>
          <div class="service-title">
            <div class="service-name">{{ svc.name }}</div>
            <div class="service-version" v-if="svc.version">v{{ svc.version }}</div>
          </div>
          <el-tag :type="svc.running ? 'success' : svc.installed ? 'warning' : 'info'" size="small">
            {{ svc.running ? '运行中' : svc.installed ? '已安装' : '未安装' }}
          </el-tag>
        </div>
        <div class="service-actions">
          <el-button v-if="!svc.installed" type="primary" size="small" @click="handleInstallService(svc)">
            <el-icon><Download /></el-icon> 安装
          </el-button>
          <template v-else>
            <el-button size="small" @click="handleServiceAction(svc, 'start')" :disabled="svc.running">
              <el-icon><VideoPlay /></el-icon>
            </el-button>
            <el-button size="small" @click="handleServiceAction(svc, 'stop')" :disabled="!svc.running">
              <el-icon><VideoPause /></el-icon>
            </el-button>
            <el-button size="small" @click="handleServiceAction(svc, 'restart')">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </template>
          <el-button size="small" @click="handleConfig(svc)">
            <el-icon><Setting /></el-icon>
          </el-button>
        </div>
        <div class="service-meta">
          <span>端口: {{ svc.port }}</span>
        </div>
      </div>
    </div>

    <!-- Tab 导航 -->
    <el-tabs v-model="activeTab" type="border-card" class="db-tabs">
      <el-tab-pane label="数据库实例" name="instances">
        <div class="tab-toolbar">
          <el-button type="primary" size="small" @click="showCreateInstance">
            <el-icon><Plus /></el-icon> 添加实例
          </el-button>
          <el-button size="small" @click="loadInstances">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
        <el-table :data="instances" stripe v-loading="loadingInstances" style="width: 100%">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="实例名称" min-width="120" />
          <el-table-column prop="type" label="类型" width="100">
            <template #default="{ row }">
              <el-tag :type="row.type === 'mysql' ? '' : row.type === 'postgresql' ? 'success' : 'warning'" size="small">
                {{ row.type.toUpperCase() }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="version" label="版本" width="80" />
          <el-table-column prop="host" label="主机" width="120" />
          <el-table-column prop="port" label="端口" width="80" />
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 'running' ? 'success' : 'danger'" size="small">
                {{ row.status === 'running' ? '运行' : '停止' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="240" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="selectInstance(row)">
                <el-icon><Monitor /></el-icon> 管理
              </el-button>
              <el-button size="small" :type="row.status === 'running' ? 'warning' : 'success'" @click="handleInstanceAction(row)">
                <el-icon v-if="row.status === 'running'"><VideoPause /></el-icon>
                <el-icon v-else><VideoPlay /></el-icon>
              </el-button>
              <el-button size="small" type="danger" @click="handleDeleteInstance(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 当选中实例后的子标签 -->
      <template v-if="selectedInstance">
        <el-tab-pane :label="`数据库 (${selectedInstance.name})`" name="databases">
          <div class="tab-toolbar">
            <el-button type="primary" size="small" @click="showCreateDatabase">
              <el-icon><Plus /></el-icon> 新建数据库
            </el-button>
            <el-button size="small" @click="handleSyncDatabases">
              <el-icon><Refresh /></el-icon> 同步
            </el-button>
          </div>
          <el-table :data="databases" stripe v-loading="loadingDatabases" style="width: 100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="数据库名" min-width="150">
              <template #default="{ row }">
                <span class="db-name">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="charset" label="字符集" width="100" />
            <el-table-column prop="collation" label="排序规则" width="150" show-overflow-tooltip />
            <el-table-column prop="size" label="大小" width="100">
              <template #default="{ row }">
                {{ formatSize(row.size) }}
              </template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="handleBackup(row)">
                  <el-icon><Download /></el-icon> 备份
                </el-button>
                <el-button size="small" type="danger" @click="handleDeleteDatabase(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="`用户 (${selectedInstance.name})`" name="users">
          <div class="tab-toolbar">
            <el-button type="primary" size="small" @click="showCreateUser">
              <el-icon><Plus /></el-icon> 新建用户
            </el-button>
            <el-button size="small" @click="loadUsers">
              <el-icon><Refresh /></el-icon> 刷新
            </el-button>
          </div>
          <el-table :data="users" stripe v-loading="loadingUsers" style="width: 100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="username" label="用户名" min-width="120" />
            <el-table-column prop="host" label="主机" width="120" />
            <el-table-column prop="db_name" label="授权数据库" width="150" />
            <el-table-column prop="privileges" label="权限" width="120" />
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button size="small" type="danger" @click="handleDeleteUser(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="`备份 (${selectedInstance.name})`" name="backups">
          <div class="tab-toolbar">
            <el-button size="small" @click="loadBackups">
              <el-icon><Refresh /></el-icon> 刷新
            </el-button>
          </div>
          <el-table :data="backups" stripe v-loading="loadingBackups" style="width: 100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="db_name" label="数据库" width="150" />
            <el-table-column prop="file_path" label="文件路径" min-width="250" show-overflow-tooltip />
            <el-table-column prop="size" label="大小" width="100">
              <template #default="{ row }">
                {{ formatSize(row.size) }}
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="170" />
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button size="small" @click="handleRestore(row)" :disabled="row.status !== 'success'">
                  <el-icon><RefreshLeft /></el-icon> 恢复
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </template>
    </el-tabs>

    <!-- 创建实例对话框 -->
    <el-dialog v-model="instanceDialogVisible" title="添加数据库实例" width="500px" destroy-on-close>
      <el-form :model="instanceForm" label-width="100px">
        <el-form-item label="实例名称" required>
          <el-input v-model="instanceForm.name" placeholder="例如：本地 MySQL" />
        </el-form-item>
        <el-form-item label="数据库类型" required>
          <el-select v-model="instanceForm.type" @change="onTypeChange">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="Redis" value="redis" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="instanceForm.version" placeholder="自动检测" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="instanceForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="Root 密码">
          <el-input v-model="instanceForm.root_pass" type="password" show-password placeholder="数据库 root 密码" />
        </el-form-item>
        <el-form-item label="安装方式">
          <el-radio-group v-model="instanceForm.install_way">
            <el-radio-button value="apt">系统包管理</el-radio-button>
            <el-radio-button value="docker">Docker</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="instanceDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateInstance" :loading="saving">创建</el-button>
      </template>
    </el-dialog>

    <!-- 创建数据库对话框 -->
    <el-dialog v-model="databaseDialogVisible" title="新建数据库" width="450px" destroy-on-close>
      <el-form :model="databaseForm" label-width="80px">
        <el-form-item label="数据库名" required>
          <el-input v-model="databaseForm.name" placeholder="例如：my_app" />
        </el-form-item>
        <el-form-item label="字符集">
          <el-select v-model="databaseForm.charset">
            <el-option label="utf8mb4" value="utf8mb4" />
            <el-option label="utf8" value="utf8" />
            <el-option label="latin1" value="latin1" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="databaseForm.remark" placeholder="备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="databaseDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateDatabase" :loading="saving">创建</el-button>
      </template>
    </el-dialog>

    <!-- 创建用户对话框 -->
    <el-dialog v-model="userDialogVisible" title="新建数据库用户" width="500px" destroy-on-close>
      <el-form :model="userForm" label-width="100px">
        <el-form-item label="用户名" required>
          <el-input v-model="userForm.username" placeholder="例如：app_user" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="userForm.password" type="password" show-password placeholder="用户密码" />
        </el-form-item>
        <el-form-item label="允许主机">
          <el-input v-model="userForm.host" placeholder="默认: % (所有)" />
        </el-form-item>
        <el-form-item label="授权数据库">
          <el-select v-model="userForm.db_name" clearable placeholder="选择数据库">
            <el-option v-for="db in databases" :key="db.id" :label="db.name" :value="db.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="权限">
          <el-select v-model="userForm.privileges">
            <el-option label="ALL (全部权限)" value="ALL" />
            <el-option label="SELECT, INSERT, UPDATE" value="SELECT, INSERT, UPDATE" />
            <el-option label="SELECT (只读)" value="SELECT" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateUser" :loading="saving">创建</el-button>
      </template>
    </el-dialog>

    <!-- 配置编辑对话框 -->
    <el-dialog v-model="configDialogVisible" title="数据库配置" width="700px" destroy-on-close>
      <div class="config-editor">
        <div class="config-info">配置文件: {{ configData.config_path }}</div>
        <el-input
          v-model="configData.content"
          type="textarea"
          :rows="20"
          class="config-textarea"
          placeholder="配置内容"
        />
      </div>
      <template #footer>
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveConfig" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Coin, DataLine, TrendCharts, Download, VideoPlay, VideoPause,
  Refresh, Setting, Plus, Monitor, Delete, RefreshLeft
} from '@element-plus/icons-vue'
import {
  getDBServiceStatus, listDBInstances, createDBInstance, dbInstanceAction,
  getDBInstanceConfig, saveDBInstanceConfig,
  listDBDatabases, createDBDatabase, deleteDBDatabase, syncDBDatabases,
  listDBUsers, createDBUser, deleteDBUser,
  listDBBackups, createDBBackup, restoreDBBackup
} from '@/api/database_new'

const activeTab = ref('instances')
const services = ref<any[]>([])
const instances = ref<any[]>([])
const databases = ref<any[]>([])
const users = ref<any[]>([])
const backups = ref<any[]>([])
const selectedInstance = ref<any>(null)

const loadingInstances = ref(false)
const loadingDatabases = ref(false)
const loadingUsers = ref(false)
const loadingBackups = ref(false)
const saving = ref(false)

// 对话框
const instanceDialogVisible = ref(false)
const databaseDialogVisible = ref(false)
const userDialogVisible = ref(false)
const configDialogVisible = ref(false)

const instanceForm = ref({
  name: '',
  type: 'mysql',
  version: '',
  port: 3306,
  root_pass: '',
  install_way: 'apt'
})

const databaseForm = ref({
  name: '',
  charset: 'utf8mb4',
  remark: ''
})

const userForm = ref({
  username: '',
  password: '',
  host: '%',
  db_name: '',
  privileges: 'ALL'
})

const configData = ref({ config_path: '', content: '' })
const configService = ref<any>(null)

onMounted(() => {
  loadServices()
  loadInstances()
})

async function loadServices() {
  try {
    const res: any = await getDBServiceStatus()
    services.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载服务状态失败')
  }
}

async function loadInstances() {
  loadingInstances.value = true
  try {
    const res: any = await listDBInstances()
    instances.value = res.data?.items || res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载实例失败')
  } finally {
    loadingInstances.value = false
  }
}

function onTypeChange(type: string) {
  const ports: any = { mysql: 3306, postgresql: 5432, redis: 6379 }
  instanceForm.value.port = ports[type] || 3306
}

function showCreateInstance() {
  instanceForm.value = {
    name: '',
    type: 'mysql',
    version: '',
    port: 3306,
    root_pass: '',
    install_way: 'apt'
  }
  instanceDialogVisible.value = true
}

async function handleCreateInstance() {
  saving.value = true
  try {
    await createDBInstance(instanceForm.value)
    ElMessage.success('创建成功')
    instanceDialogVisible.value = false
    loadInstances()
    loadServices()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    saving.value = false
  }
}

async function handleInstallService(svc: any) {
  try {
    // 查找或创建实例
    let instance = instances.value.find(i => i.type === svc.type)
    if (!instance) {
      await createDBInstance({
        name: svc.name,
        type: svc.type,
        install_way: 'apt'
      })
      await loadInstances()
      instance = instances.value.find(i => i.type === svc.type)
    }
    if (instance) {
      await dbInstanceAction(instance.id, 'install')
      ElMessage.success('正在安装...')
      setTimeout(() => {
        loadServices()
        loadInstances()
      }, 5000)
    }
  } catch (e: any) {
    ElMessage.error(e.message || '安装失败')
  }
}

async function handleServiceAction(svc: any, action: string) {
  try {
    const instance = instances.value.find(i => i.type === svc.type)
    if (instance) {
      await dbInstanceAction(instance.id, action)
      ElMessage.success('操作成功')
      loadServices()
      loadInstances()
    } else {
      ElMessage.warning('请先添加实例')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleConfig(svc: any) {
  configService.value = svc
  const instance = instances.value.find(i => i.type === svc.type)
  if (instance) {
    try {
      const res: any = await getDBInstanceConfig(instance.id)
      configData.value = res.data || { config_path: '', content: '' }
      configDialogVisible.value = true
    } catch (e: any) {
      ElMessage.error(e.message || '加载配置失败')
    }
  } else {
    ElMessage.warning('请先添加实例')
  }
}

async function handleSaveConfig() {
  const instance = instances.value.find(i => i.type === configService.value.type)
  if (!instance) return
  saving.value = true
  try {
    await saveDBInstanceConfig(instance.id, configData.value.content)
    ElMessage.success('配置已保存')
    configDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function selectInstance(instance: any) {
  selectedInstance.value = instance
  activeTab.value = 'databases'
  loadDatabases()
  loadUsers()
  loadBackups()
}

async function handleInstanceAction(instance: any) {
  const action = instance.status === 'running' ? 'stop' : 'start'
  try {
    await dbInstanceAction(instance.id, action)
    ElMessage.success('操作成功')
    loadInstances()
    loadServices()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDeleteInstance(instance: any) {
  await ElMessageBox.confirm(`确定删除实例 "${instance.name}"?`, '确认删除', { type: 'warning' })
  // 暂不支持删除实例
  ElMessage.info('暂不支持删除实例')
}

// 数据库管理
async function loadDatabases() {
  if (!selectedInstance.value) return
  loadingDatabases.value = true
  try {
    const res: any = await listDBDatabases(selectedInstance.value.id)
    databases.value = res.data?.items || res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载数据库失败')
  } finally {
    loadingDatabases.value = false
  }
}

function showCreateDatabase() {
  databaseForm.value = { name: '', charset: 'utf8mb4', remark: '' }
  databaseDialogVisible.value = true
}

async function handleCreateDatabase() {
  if (!selectedInstance.value) return
  saving.value = true
  try {
    await createDBDatabase({
      instance_id: selectedInstance.value.id,
      ...databaseForm.value
    })
    ElMessage.success('创建成功')
    databaseDialogVisible.value = false
    loadDatabases()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    saving.value = false
  }
}

async function handleDeleteDatabase(db: any) {
  await ElMessageBox.confirm(`确定删除数据库 "${db.name}"? 此操作不可恢复！`, '确认删除', { type: 'warning' })
  try {
    await deleteDBDatabase(db.id)
    ElMessage.success('删除成功')
    loadDatabases()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

async function handleSyncDatabases() {
  if (!selectedInstance.value) return
  try {
    await syncDBDatabases(selectedInstance.value.id)
    ElMessage.success('同步成功')
    loadDatabases()
  } catch (e: any) {
    ElMessage.error(e.message || '同步失败')
  }
}

// 用户管理
async function loadUsers() {
  if (!selectedInstance.value) return
  loadingUsers.value = true
  try {
    const res: any = await listDBUsers(selectedInstance.value.id)
    users.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载用户失败')
  } finally {
    loadingUsers.value = false
  }
}

function showCreateUser() {
  userForm.value = { username: '', password: '', host: '%', db_name: '', privileges: 'ALL' }
  userDialogVisible.value = true
}

async function handleCreateUser() {
  if (!selectedInstance.value) return
  saving.value = true
  try {
    await createDBUser(selectedInstance.value.id, userForm.value)
    ElMessage.success('创建成功')
    userDialogVisible.value = false
    loadUsers()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    saving.value = false
  }
}

async function handleDeleteUser(user: any) {
  await ElMessageBox.confirm(`确定删除用户 "${user.username}"?`, '确认删除', { type: 'warning' })
  try {
    await deleteDBUser(user.id)
    ElMessage.success('删除成功')
    loadUsers()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

// 备份管理
async function loadBackups() {
  if (!selectedInstance.value) return
  loadingBackups.value = true
  try {
    const res: any = await listDBBackups(selectedInstance.value.id)
    backups.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载备份失败')
  } finally {
    loadingBackups.value = false
  }
}

async function handleBackup(db: any) {
  if (!selectedInstance.value) return
  try {
    await createDBBackup(selectedInstance.value.id, db.name)
    ElMessage.success('正在备份...')
    setTimeout(loadBackups, 3000)
  } catch (e: any) {
    ElMessage.error(e.message || '备份失败')
  }
}

async function handleRestore(backup: any) {
  await ElMessageBox.confirm(`确定恢复数据库 "${backup.db_name}"? 当前数据将被覆盖！`, '确认恢复', { type: 'warning' })
  try {
    await restoreDBBackup(backup.id)
    ElMessage.success('正在恢复...')
  } catch (e: any) {
    ElMessage.error(e.message || '恢复失败')
  }
}

function formatSize(bytes: number) {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}
</script>

<style scoped>
.database-page {
  padding: 0;
}

.services-overview {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.service-card {
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  padding: 16px;
}

.service-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.service-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: white;
}

.service-icon.mysql { background: linear-gradient(135deg, #00758f, #f29111); }
.service-icon.postgresql { background: linear-gradient(135deg, #336791, #fff); color: #336791; }
.service-icon.redis { background: linear-gradient(135deg, #dc382d, #a41e11); }

.service-title {
  flex: 1;
}

.service-name {
  font-weight: 600;
  font-size: 15px;
}

.service-version {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.service-actions {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}

.service-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.db-tabs {
  margin-top: 8px;
}

.tab-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.db-name {
  font-weight: 500;
  font-family: 'JetBrains Mono', monospace;
}

.config-editor {
  .config-info {
    margin-bottom: 8px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  .config-textarea :deep(textarea) {
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 13px;
    line-height: 1.5;
  }
}
</style>
