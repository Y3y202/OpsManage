<template>
  <div class="nginx-page">
    <!-- Nginx 服务状态 -->
    <div class="service-status-bar" v-if="nginxStatus">
      <div class="service-info">
        <span class="service-name">
          <el-icon><Monitor /></el-icon>
          Nginx
        </span>
        <el-tag :type="nginxStatus.running ? 'success' : 'danger'" size="small">
          {{ nginxStatus.running ? '运行中' : '未运行' }}
        </el-tag>
        <span class="version" v-if="nginxStatus.version">v{{ nginxStatus.version }}</span>
        <span class="not-installed" v-if="!nginxStatus.installed">未安装</span>
      </div>
      <div class="service-actions">
        <el-button v-if="!nginxStatus.installed" type="primary" size="small" @click="handleInstallNginx" :loading="installing">
          <el-icon><Download /></el-icon> 安装 Nginx
        </el-button>
        <template v-else>
          <el-button size="small" @click="handleService('start')" :disabled="nginxStatus.running">
            <el-icon><VideoPlay /></el-icon> 启动
          </el-button>
          <el-button size="small" @click="handleService('stop')" :disabled="!nginxStatus.running">
            <el-icon><VideoPause /></el-icon> 停止
          </el-button>
          <el-button size="small" @click="handleService('restart')">
            <el-icon><Refresh /></el-icon> 重启
          </el-button>
          <el-button size="small" @click="handleService('reload')">
            <el-icon><RefreshRight /></el-icon> 重载
          </el-button>
        </template>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row" v-if="overview">
      <div class="stat-card">
        <div class="stat-icon blue"><el-icon><Promotion /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.total_sites || 0 }}</div>
          <div class="stat-label">站点总数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon green"><el-icon><CircleCheck /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.running_sites || 0 }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon orange"><el-icon><Lock /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.ssl_sites || 0 }}</div>
          <div class="stat-label">SSL 站点</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon purple"><el-icon><Connection /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.active_conns || 0 }}</div>
          <div class="stat-label">活跃连接</div>
        </div>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="action-bar">
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon> 添加站点
      </el-button>
      <el-button @click="handleImport">
        <el-icon><Upload /></el-icon> 导入已有配置
      </el-button>
      <el-button @click="handleTestConfig">
        <el-icon><CircleCheck /></el-icon> 测试配置
      </el-button>
    </div>

    <!-- 站点列表 -->
    <div class="sites-table">
      <el-table :data="sites" stripe v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="站点名称" min-width="120">
          <template #default="{ row }">
            <span class="site-name">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="domain" label="域名" min-width="150">
          <template #default="{ row }">
            <a :href="'http://' + row.domain" target="_blank" class="domain-link">
              {{ row.domain }}
              <el-icon><Link /></el-icon>
            </a>
          </template>
        </el-table-column>
        <el-table-column prop="root" label="根目录" min-width="180" show-overflow-tooltip />
        <el-table-column prop="proxy_type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.proxy_type === 'proxy' ? 'warning' : row.proxy_type === 'php' ? 'success' : 'info'">
              {{ row.proxy_type === 'proxy' ? '代理' : row.proxy_type === 'php' ? 'PHP' : '静态' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ssl" label="SSL" width="60">
          <template #default="{ row }">
            <el-icon v-if="row.ssl" class="ssl-icon green"><Lock /></el-icon>
            <el-icon v-else class="ssl-icon gray"><Unlock /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'running' ? 'success' : 'danger'" size="small">
              {{ row.status === 'running' ? '运行' : '停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button size="small" @click="handleEditSite(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button size="small" @click="handleSSL(row)">
                <el-icon><Lock /></el-icon>
              </el-button>
              <el-button size="small" @click="handleConfig(row)">
                <el-icon><Document /></el-icon>
              </el-button>
              <el-button size="small" @click="handleLogs(row)">
                <el-icon><List /></el-icon>
              </el-button>
              <el-button size="small" :type="row.status === 'running' ? 'warning' : 'success'" @click="handleToggleSite(row)">
                <el-icon v-if="row.status === 'running'"><VideoPause /></el-icon>
                <el-icon v-else><VideoPlay /></el-icon>
              </el-button>
              <el-button size="small" type="danger" @click="handleDeleteSite(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建/编辑站点对话框 -->
    <el-dialog v-model="siteDialogVisible" :title="editingSite ? '编辑站点' : '添加站点'" width="600px" destroy-on-close>
      <el-form :model="siteForm" label-width="100px">
        <el-form-item label="站点名称" required>
          <el-input v-model="siteForm.name" placeholder="例如：my-blog" />
        </el-form-item>
        <el-form-item label="域名" required>
          <el-input v-model="siteForm.domain" placeholder="例如：www.example.com" />
        </el-form-item>
        <el-form-item label="根目录" required>
          <el-input v-model="siteForm.root" placeholder="/var/www/html">
            <template #append>
              <el-button @click="siteForm.root = '/var/www/' + siteForm.name">自动</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="siteForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="站点类型">
          <el-radio-group v-model="siteForm.proxy_type">
            <el-radio-button value="static">静态网站</el-radio-button>
            <el-radio-button value="proxy">反向代理</el-radio-button>
            <el-radio-button value="php">PHP 网站</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="代理地址" v-if="siteForm.proxy_type === 'proxy'">
          <el-input v-model="siteForm.proxy_pass" placeholder="例如：http://127.0.0.1:3000" />
        </el-form-item>
        <el-form-item label="Gzip 压缩">
          <el-switch v-model="siteForm.gzip" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="siteForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="siteDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveSite" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- SSL 管理对话框 -->
    <el-dialog v-model="sslDialogVisible" title="SSL 证书管理" width="500px" destroy-on-close>
      <div v-if="currentSite">
        <div class="ssl-status">
          <el-tag :type="currentSite.ssl ? 'success' : 'info'" size="large">
            {{ currentSite.ssl ? 'SSL 已启用' : 'SSL 未启用' }}
          </el-tag>
        </div>
        <el-form :model="sslForm" label-width="100px" style="margin-top: 20px">
          <el-form-item label="申请方式">
            <el-radio-group v-model="sslForm.method">
              <el-radio value="acme">Let's Encrypt (自动)</el-radio>
              <el-radio value="manual">手动指定证书</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="邮箱" v-if="sslForm.method === 'acme'">
            <el-input v-model="sslForm.email" placeholder="用于接收证书过期通知" />
          </el-form-item>
          <el-form-item label="证书路径" v-if="sslForm.method === 'manual'">
            <el-input v-model="sslForm.cert_path" placeholder="/path/to/fullchain.pem" />
          </el-form-item>
          <el-form-item label="密钥路径" v-if="sslForm.method === 'manual'">
            <el-input v-model="sslForm.key_path" placeholder="/path/to/privkey.pem" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button v-if="currentSite?.ssl" @click="handleSSLAction('disable')">关闭 SSL</el-button>
        <el-button v-if="currentSite?.ssl" @click="handleSSLAction('renew')">续签证书</el-button>
        <el-button type="primary" @click="handleSSLAction('enable')" :loading="sslLoading">
          {{ currentSite?.ssl ? '更新证书' : '启用 SSL' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 配置编辑器对话框 -->
    <el-dialog v-model="configDialogVisible" title="Nginx 配置编辑" width="800px" destroy-on-close>
      <div v-if="currentSite" class="config-editor">
        <div class="config-info">
          <span>配置文件: {{ configData.config_file }}</span>
        </div>
        <el-input
          v-model="configData.content"
          type="textarea"
          :rows="20"
          class="config-textarea"
          placeholder="Nginx 配置内容"
        />
      </div>
      <template #footer>
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveConfig" :loading="saving">保存并重载</el-button>
      </template>
    </el-dialog>

    <!-- 日志查看器对话框 -->
    <el-dialog v-model="logDialogVisible" title="站点日志" width="900px" destroy-on-close>
      <div v-if="currentSite" class="log-viewer">
        <div class="log-toolbar">
          <el-radio-group v-model="logType" @change="loadLogs">
            <el-radio-button value="access">访问日志</el-radio-button>
            <el-radio-button value="error">错误日志</el-radio-button>
          </el-radio-group>
          <el-input-number v-model="logLines" :min="50" :max="1000" :step="50" @change="loadLogs" />
          <el-button @click="loadLogs" :icon="Refresh">刷新</el-button>
        </div>
        <pre class="log-content">{{ logContent }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Monitor, Download, VideoPlay, VideoPause, Refresh, RefreshRight,
  CircleCheck, Lock, Unlock, Connection, Plus, Upload,
  Edit, Document, List, Delete, Link, Promotion
} from '@element-plus/icons-vue'
import {
  getNginxStatus, getNginxOverview, installNginx, nginxService,
  testNginxConfig, importNginxSites,
  listNginxSites, createNginxSite, updateNginxSite, deleteNginxSite,
  nginxSiteAction, manageNginxSSL, getNginxSiteConfig, saveNginxSiteConfig,
  getNginxSiteLogs
} from '@/api/nginx'

const loading = ref(false)
const saving = ref(false)
const installing = ref(false)
const nginxStatus = ref<any>(null)
const overview = ref<any>(null)
const sites = ref<any[]>([])

// 站点对话框
const siteDialogVisible = ref(false)
const editingSite = ref<any>(null)
const siteForm = ref({
  name: '',
  domain: '',
  root: '',
  port: 80,
  proxy_type: 'static',
  proxy_pass: '',
  gzip: true,
  remark: ''
})

// SSL 对话框
const sslDialogVisible = ref(false)
const sslLoading = ref(false)
const sslForm = ref({
  method: 'acme',
  email: '',
  cert_path: '',
  key_path: ''
})

// 配置编辑器
const configDialogVisible = ref(false)
const configData = ref({ config_file: '', content: '' })

// 日志查看器
const logDialogVisible = ref(false)
const logType = ref('access')
const logLines = ref(100)
const logContent = ref('')

const currentSite = ref<any>(null)

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const [statusRes, overviewRes, sitesRes] = await Promise.all([
      getNginxStatus(),
      getNginxOverview(),
      listNginxSites({ page: 1, page_size: 100 })
    ])
    nginxStatus.value = (statusRes as any).data
    overview.value = (overviewRes as any).data
    sites.value = (sitesRes as any).data?.items || (sitesRes as any).data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function handleInstallNginx() {
  installing.value = true
  try {
    await installNginx()
    ElMessage.success('正在安装 Nginx...')
    setTimeout(loadData, 3000)
  } catch (e: any) {
    ElMessage.error(e.message || '安装失败')
  } finally {
    installing.value = false
  }
}

async function handleService(action: string) {
  try {
    await nginxService(action)
    ElMessage.success('操作成功')
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleTestConfig() {
  try {
    const res: any = await testNginxConfig()
    if (res.data?.valid) {
      ElMessage.success('Nginx 配置测试通过')
    } else {
      ElMessage.error('配置测试失败: ' + (res.data?.output || ''))
    }
  } catch (e: any) {
    ElMessage.error(e.message || '测试失败')
  }
}

async function handleImport() {
  try {
    const res: any = await importNginxSites()
    ElMessage.success(`已导入 ${res.data?.imported || 0} 个站点`)
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '导入失败')
  }
}

function showCreateDialog() {
  editingSite.value = null
  siteForm.value = {
    name: '',
    domain: '',
    root: '',
    port: 80,
    proxy_type: 'static',
    proxy_pass: '',
    gzip: true,
    remark: ''
  }
  siteDialogVisible.value = true
}

function handleEditSite(site: any) {
  editingSite.value = site
  siteForm.value = {
    name: site.name,
    domain: site.domain,
    root: site.root,
    port: site.port,
    proxy_type: site.proxy_type,
    proxy_pass: site.proxy_pass || '',
    gzip: site.gzip,
    remark: site.remark || ''
  }
  siteDialogVisible.value = true
}

async function handleSaveSite() {
  saving.value = true
  try {
    if (editingSite.value) {
      await updateNginxSite(editingSite.value.id, siteForm.value)
      ElMessage.success('更新成功')
    } else {
      await createNginxSite(siteForm.value)
      ElMessage.success('创建成功')
    }
    siteDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDeleteSite(site: any) {
  await ElMessageBox.confirm(`确定删除站点 "${site.name}" (${site.domain})?`, '确认删除', { type: 'warning' })
  try {
    await deleteNginxSite(site.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

async function handleToggleSite(site: any) {
  const action = site.status === 'running' ? 'disable' : 'enable'
  try {
    await nginxSiteAction(site.id, action)
    ElMessage.success('操作成功')
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

function handleSSL(site: any) {
  currentSite.value = site
  sslForm.value = {
    method: 'acme',
    email: '',
    cert_path: site.ssl_cert || '',
    key_path: site.ssl_key || ''
  }
  sslDialogVisible.value = true
}

async function handleSSLAction(action: string) {
  sslLoading.value = true
  try {
    const data: any = { action }
    if (action === 'enable') {
      if (sslForm.value.method === 'acme') {
        data.email = sslForm.value.email
      } else {
        data.cert_path = sslForm.value.cert_path
        data.key_path = sslForm.value.key_path
      }
    }
    await manageNginxSSL(currentSite.value.id, data)
    ElMessage.success('操作成功')
    sslDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    sslLoading.value = false
  }
}

async function handleConfig(site: any) {
  currentSite.value = site
  try {
    const res: any = await getNginxSiteConfig(site.id)
    configData.value = res.data || { config_file: '', content: '' }
    configDialogVisible.value = true
  } catch (e: any) {
    ElMessage.error(e.message || '加载配置失败')
  }
}

async function handleSaveConfig() {
  saving.value = true
  try {
    await saveNginxSiteConfig(currentSite.value.id, configData.value.content)
    ElMessage.success('配置已保存并重载')
    configDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleLogs(site: any) {
  currentSite.value = site
  logType.value = 'access'
  logDialogVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  try {
    const res: any = await getNginxSiteLogs(currentSite.value.id, logType.value, logLines.value)
    logContent.value = res.data?.logs || '暂无日志'
  } catch (e: any) {
    logContent.value = '加载日志失败: ' + e.message
  }
}
</script>

<style scoped>
.nginx-page {
  padding: 0;
}

.service-status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px 20px;
  margin-bottom: 16px;
}

.service-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.service-name {
  font-weight: 600;
  font-size: 15px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.version {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.not-installed {
  color: var(--el-color-warning);
  font-size: 13px;
}

.service-actions {
  display: flex;
  gap: 8px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.stat-card {
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: white;
}

.stat-icon.blue { background: linear-gradient(135deg, #667eea, #764ba2); }
.stat-icon.green { background: linear-gradient(135deg, #11998e, #38ef7d); }
.stat-icon.orange { background: linear-gradient(135deg, #f093fb, #f5576c); }
.stat-icon.purple { background: linear-gradient(135deg, #4facfe, #00f2fe); }

.stat-value {
  font-size: 22px;
  font-weight: 700;
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.action-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.ssl-icon.green { color: var(--el-color-success); }
.ssl-icon.gray { color: var(--el-text-color-placeholder); }

.site-name {
  font-weight: 500;
}

.domain-link {
  color: var(--el-color-primary);
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 4px;
}

.domain-link:hover {
  text-decoration: underline;
}

.ssl-status {
  text-align: center;
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

.log-viewer {
  .log-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
  }
  .log-content {
    background: #1e1e1e;
    color: #d4d4d4;
    padding: 16px;
    border-radius: 6px;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 12px;
    line-height: 1.6;
    max-height: 500px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
}
</style>
