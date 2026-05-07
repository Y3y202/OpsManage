<script setup lang="ts">
defineOptions({
  name: 'OpsContainer',
})

import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as api from '@/api/ops/container'
import type { ContainerRow, CreateContainerForm, DockerOverview } from '@/api/ops/container'

const activeTab = ref('overview')
const overviewLoading = ref(false)
const overview = ref<DockerOverview>(api.normalizeDockerOverview())

const containerList = ref<ContainerRow[]>([])
const containerTotal = ref(0)
const cPage = ref(1)
const cPageSize = ref(20)
const cLoading = ref(false)
const cDialog = ref(false)
const cSubmitting = ref(false)
const cForm = reactive<CreateContainerForm>({
  name: '',
  image: '',
  ports: '',
  volumes: '',
  env: '',
  command: '',
  restartPolicy: 'unless-stopped',
  network: '',
})

const logDialog = ref(false)
const logTitle = ref('')
const logContent = ref('')
const logLoading = ref(false)

const imageList = ref<any[]>([])
const iLoading = ref(false)
const pullDialog = ref(false)
const pullImageName = ref('')
const pullLoading = ref(false)

const networkList = ref<any[]>([])
const nLoading = ref(false)
const volumeList = ref<any[]>([])
const vLoading = ref(false)

const registryList = ref<any[]>([])
const rLoading = ref(false)
const rDialog = ref(false)
const rForm = reactive({ name: '', url: '', username: '', password: '', is_default: false })

const composeList = ref<any[]>([])
const coLoading = ref(false)
const coDialog = ref(false)
const coForm = reactive({ name: '', path: '' })

const templateList = ref<any[]>([])
const tLoading = ref(false)
const tDialog = ref(false)
const tForm = reactive({ name: '', description: '', content: '' })

const runningRate = computed(() => {
  if (!overview.value.containers_total) {
    return 0
  }
  return Math.round((overview.value.containers_running / overview.value.containers_total) * 100)
})

async function fetchOverview() {
  overviewLoading.value = true
  try {
    overview.value = await api.getDockerOverview()
  }
  finally {
    overviewLoading.value = false
  }
}

async function fetchContainers() {
  cLoading.value = true
  try {
    const result = await api.getContainers({ page: cPage.value, page_size: cPageSize.value })
    containerList.value = result.list
    containerTotal.value = result.total
  }
  finally {
    cLoading.value = false
  }
}

async function fetchImages() {
  iLoading.value = true
  try {
    imageList.value = await api.listImages() as unknown as any[] || []
  }
  finally {
    iLoading.value = false
  }
}

async function fetchNetworks() {
  nLoading.value = true
  try {
    networkList.value = await api.listDockerNetworks() as unknown as any[] || []
  }
  finally {
    nLoading.value = false
  }
}

async function fetchVolumes() {
  vLoading.value = true
  try {
    volumeList.value = await api.listDockerVolumes() as unknown as any[] || []
  }
  finally {
    vLoading.value = false
  }
}

async function fetchRegistries() {
  rLoading.value = true
  try {
    registryList.value = await api.listRegistries() as unknown as any[] || []
  }
  finally {
    rLoading.value = false
  }
}

async function fetchCompose() {
  coLoading.value = true
  try {
    composeList.value = await api.listComposeProjects() as unknown as any[] || []
  }
  finally {
    coLoading.value = false
  }
}

async function fetchTemplates() {
  tLoading.value = true
  try {
    templateList.value = await api.listComposeTemplates() as unknown as any[] || []
  }
  finally {
    tLoading.value = false
  }
}

async function refreshActiveTab() {
  await fetchOverview()
  if (activeTab.value === 'containers') {
    await fetchContainers()
  }
  else if (activeTab.value === 'images') {
    await fetchImages()
  }
  else if (activeTab.value === 'networks') {
    await fetchNetworks()
  }
  else if (activeTab.value === 'volumes') {
    await fetchVolumes()
  }
  else if (activeTab.value === 'registries') {
    await fetchRegistries()
  }
  else if (activeTab.value === 'compose') {
    await fetchCompose()
  }
  else if (activeTab.value === 'templates') {
    await fetchTemplates()
  }
}

function onTabChange(tab: string | number) {
  activeTab.value = String(tab)
  refreshActiveTab()
}

function resetContainerForm() {
  Object.assign(cForm, {
    name: '',
    image: '',
    ports: '',
    volumes: '',
    env: '',
    command: '',
    restartPolicy: 'unless-stopped',
    network: '',
  })
}

function openCDialog() {
  resetContainerForm()
  cDialog.value = true
}

async function handleCreate() {
  if (!cForm.name.trim() || !cForm.image.trim()) {
    ElMessage.warning('请填写容器名称和镜像')
    return
  }
  cSubmitting.value = true
  try {
    await api.createContainer(cForm)
    ElMessage.success('容器创建成功')
    cDialog.value = false
    await Promise.all([fetchContainers(), fetchOverview()])
  }
  finally {
    cSubmitting.value = false
  }
}

async function confirmAction(message: string) {
  await ElMessageBox.confirm(message, '确认操作', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
}

async function handleDeleteC(row: ContainerRow) {
  await confirmAction(`确定强制删除容器「${row.name}」吗？`)
  await api.deleteContainer(row.actionId)
  ElMessage.success('容器已删除')
  await Promise.all([fetchContainers(), fetchOverview()])
}

async function handleStart(row: ContainerRow) {
  await api.startContainer(row.actionId)
  ElMessage.success('启动命令已执行')
  await Promise.all([fetchContainers(), fetchOverview()])
}

async function handleStop(row: ContainerRow) {
  await confirmAction(`确定停止容器「${row.name}」吗？`)
  await api.stopContainer(row.actionId)
  ElMessage.success('停止命令已执行')
  await Promise.all([fetchContainers(), fetchOverview()])
}

async function handleRestart(row: ContainerRow) {
  await api.restartContainer(row.actionId)
  ElMessage.success('重启命令已执行')
  await Promise.all([fetchContainers(), fetchOverview()])
}

async function handleLogs(row: ContainerRow) {
  logDialog.value = true
  logLoading.value = true
  logTitle.value = `${row.name} 日志`
  logContent.value = ''
  try {
    const r: any = await api.getContainerLogs(row.actionId)
    logContent.value = r?.logs || ''
  }
  finally {
    logLoading.value = false
  }
}

function openPullDialog() {
  pullImageName.value = ''
  pullDialog.value = true
}

async function handlePullImage() {
  if (!pullImageName.value.trim()) {
    ElMessage.warning('请输入镜像名称')
    return
  }
  pullLoading.value = true
  try {
    await api.pullImage(pullImageName.value)
    ElMessage.success('镜像拉取完成')
    pullDialog.value = false
    await Promise.all([fetchImages(), fetchOverview()])
  }
  finally {
    pullLoading.value = false
  }
}

async function handleRemoveImage(name: string) {
  await confirmAction(`确定删除镜像「${name}」吗？`)
  await api.removeImage(name)
  ElMessage.success('镜像已删除')
  await Promise.all([fetchImages(), fetchOverview()])
}

async function handleRemoveNetwork(id: string, name?: string) {
  await confirmAction(`确定删除网络「${name || id}」吗？`)
  await api.removeNetwork(id)
  ElMessage.success('网络已删除')
  await Promise.all([fetchNetworks(), fetchOverview()])
}

async function handleRemoveVolume(name: string) {
  await confirmAction(`确定删除存储卷「${name}」吗？`)
  await api.removeVolume(name)
  ElMessage.success('存储卷已删除')
  await Promise.all([fetchVolumes(), fetchOverview()])
}

function openRDialog() {
  Object.assign(rForm, { name: '', url: '', username: '', password: '', is_default: false })
  rDialog.value = true
}
async function handleCreateR() {
  await api.createRegistry(rForm)
  ElMessage.success('镜像仓库已添加')
  rDialog.value = false
  await fetchRegistries()
}
async function handleDeleteR(id: number) {
  await confirmAction('确定删除该镜像仓库吗？')
  await api.deleteRegistry(id)
  ElMessage.success('镜像仓库已删除')
  await fetchRegistries()
}

function openCoDialog() {
  Object.assign(coForm, { name: '', path: '' })
  coDialog.value = true
}
async function handleCreateCo() {
  await api.createComposeProject(coForm)
  ElMessage.success('编排项目已添加')
  coDialog.value = false
  await fetchCompose()
}
async function handleDeleteCo(id: number) {
  await confirmAction('确定删除该编排项目吗？')
  await api.deleteComposeProject(id)
  ElMessage.success('编排项目已删除')
  await fetchCompose()
}
async function handleStartCo(id: number) {
  await api.startComposeProject(id)
  ElMessage.success('编排项目已启动')
  await fetchCompose()
}
async function handleStopCo(id: number) {
  await api.stopComposeProject(id)
  ElMessage.success('编排项目已停止')
  await fetchCompose()
}

function openTDialog() {
  Object.assign(tForm, { name: '', description: '', content: '' })
  tDialog.value = true
}
async function handleCreateT() {
  await api.createComposeTemplate(tForm)
  ElMessage.success('编排模板已添加')
  tDialog.value = false
  await fetchTemplates()
}
async function handleDeleteT(id: number) {
  await confirmAction('确定删除该编排模板吗？')
  await api.deleteComposeTemplate(id)
  ElMessage.success('编排模板已删除')
  await fetchTemplates()
}

async function handlePrune(type: string) {
  await ElMessageBox.confirm(`确定清理 ${type === 'all' ? '全部未使用 Docker 资源' : type}？此操作不可恢复。`, '危险操作', { type: 'warning' })
  const r: any = await api.pruneDocker(type)
  ElMessage.success(r?.msg || '清理完成')
  await refreshActiveTab()
}

onMounted(async () => {
  await fetchOverview()
  await fetchContainers()
})
</script>

<template>
  <div class="docker-page">
    <div class="page-header">
      <div>
        <h2>容器管理</h2>
        <p>管理 Docker 容器、镜像、网络、存储卷、镜像仓库和 Compose 编排。</p>
      </div>
      <div class="header-actions">
        <el-button :loading="overviewLoading" @click="refreshActiveTab">刷新</el-button>
        <el-button type="primary" @click="openCDialog">创建容器</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="top-tabs" @tab-change="onTabChange">
      <el-tab-pane label="概览" name="overview">
        <div class="stat-row">
          <el-card class="stat-card" shadow="never">
            <div class="stat-icon bg-blue"><el-icon><Box /></el-icon></div>
            <div class="stat-body">
              <div class="stat-label">容器运行率</div>
              <div class="stat-big">{{ runningRate }}%</div>
              <div class="stat-sub">运行 {{ overview.containers_running }} / 总计 {{ overview.containers_total }}</div>
            </div>
          </el-card>
          <el-card class="stat-card" shadow="never">
            <div class="stat-icon bg-orange"><el-icon><Picture /></el-icon></div>
            <div class="stat-body"><div class="stat-label">镜像</div><div class="stat-big">{{ overview.images }}</div></div>
          </el-card>
          <el-card class="stat-card" shadow="never">
            <div class="stat-icon bg-indigo"><el-icon><Share /></el-icon></div>
            <div class="stat-body"><div class="stat-label">网络</div><div class="stat-big">{{ overview.networks }}</div></div>
          </el-card>
          <el-card class="stat-card" shadow="never">
            <div class="stat-icon bg-pink"><el-icon><FolderOpened /></el-icon></div>
            <div class="stat-body"><div class="stat-label">存储卷</div><div class="stat-big">{{ overview.volumes }}</div></div>
          </el-card>
        </div>

        <div class="info-row">
          <el-card class="section-card" shadow="never">
            <template #header><div class="section-hdr"><el-icon color="#3370ff"><DataLine /></el-icon><span>磁盘占用</span></div></template>
            <div class="disk-row">
              <div v-for="(item, key) in overview.disk_usage" :key="key" class="disk-item">
                <div class="disk-label">{{ key === 'images' ? '镜像' : key === 'containers' ? '容器' : key === 'volumes' ? '本地存储卷' : '构建缓存' }}</div>
                <div class="disk-size">{{ item.size }}</div>
                <div class="disk-reclaim">可释放 {{ item.reclaimable || '0B' }}</div>
              </div>
            </div>
          </el-card>
          <el-card class="section-card" shadow="never">
            <template #header><div class="section-hdr"><el-icon color="#3370ff"><Setting /></el-icon><span>Docker 配置</span></div></template>
            <div class="config-item"><span>Socket</span><code>{{ overview.socket }}</code></div>
            <div class="config-item"><span>镜像仓库</span><el-tag>{{ overview.registries }}</el-tag></div>
            <div class="config-item"><span>Compose 项目/模板</span><el-tag>{{ overview.compose }} / {{ overview.compose_templates }}</el-tag></div>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="容器" name="containers">
        <div class="toolbar"><strong>容器列表</strong><el-button type="primary" @click="openCDialog">创建容器</el-button></div>
        <el-table :data="containerList" v-loading="cLoading" stripe border>
          <el-table-column label="ID" width="130"><template #default="{ row }"><code>{{ row.shortId }}</code></template></el-table-column>
          <el-table-column prop="name" label="名称" min-width="150" show-overflow-tooltip />
          <el-table-column prop="image" label="镜像" min-width="180" show-overflow-tooltip />
          <el-table-column prop="ports" label="端口" min-width="180" show-overflow-tooltip />
          <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="row.statusType">{{ row.status }}</el-tag></template></el-table-column>
          <el-table-column label="操作" width="360" fixed="right">
            <template #default="{ row }">
              <el-button-group>
                <el-button size="small" type="success" plain :disabled="row.status === 'running'" @click="handleStart(row)">启动</el-button>
                <el-button size="small" type="warning" plain :disabled="row.status !== 'running'" @click="handleStop(row)">停止</el-button>
                <el-button size="small" @click="handleRestart(row)">重启</el-button>
                <el-button size="small" @click="handleLogs(row)">日志</el-button>
                <el-button size="small" type="danger" plain @click="handleDeleteC(row)">删除</el-button>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination v-model:current-page="cPage" class="pagination" :total="containerTotal" :page-size="cPageSize" layout="total, prev, pager, next" @current-change="fetchContainers" />
      </el-tab-pane>

      <el-tab-pane label="镜像" name="images">
        <div class="toolbar"><strong>镜像列表</strong><el-button type="primary" @click="openPullDialog">拉取镜像</el-button></div>
        <el-table :data="imageList" v-loading="iLoading" stripe border empty-text="暂无镜像">
          <el-table-column prop="name" label="镜像名" min-width="240" show-overflow-tooltip />
          <el-table-column prop="id" label="IMAGE ID" min-width="140" />
          <el-table-column prop="size" label="大小" width="120" />
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" plain @click="handleRemoveImage(row.name)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="网络" name="networks">
        <el-table :data="networkList" v-loading="nLoading" stripe border empty-text="暂无网络">
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="id" label="NETWORK ID" min-width="160" />
          <el-table-column prop="driver" label="驱动" width="120" />
          <el-table-column prop="scope" label="范围" width="100" />
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" plain :disabled="['bridge', 'host', 'none'].includes(row.name)" @click="handleRemoveNetwork(row.id, row.name)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="存储卷" name="volumes">
        <el-table :data="volumeList" v-loading="vLoading" stripe border empty-text="暂无存储卷">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="driver" label="驱动" width="120" />
          <el-table-column prop="mountpoint" label="挂载点" min-width="260" show-overflow-tooltip />
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" plain @click="handleRemoveVolume(row.name)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="仓库" name="registries">
        <div class="toolbar"><strong>镜像仓库</strong><el-button type="primary" @click="openRDialog">添加仓库</el-button></div>
        <el-table :data="registryList" v-loading="rLoading" stripe border empty-text="暂无仓库">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="url" label="地址" min-width="240" show-overflow-tooltip />
          <el-table-column prop="username" label="用户名" width="140" />
          <el-table-column label="默认" width="80"><template #default="{ row }"><el-tag v-if="row.is_default" type="success">是</el-tag><span v-else>-</span></template></el-table-column>
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" plain @click="handleDeleteR(row.id)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="编排" name="compose">
        <div class="toolbar"><strong>Compose 项目</strong><el-button type="primary" @click="openCoDialog">添加编排</el-button></div>
        <el-table :data="composeList" v-loading="coLoading" stripe border empty-text="暂无编排项目">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="path" label="Compose 文件" min-width="260" show-overflow-tooltip />
          <el-table-column prop="services" label="服务数" width="100" />
          <el-table-column prop="status" label="状态" width="110"><template #default="{ row }"><el-tag :type="row.status === 'running' ? 'success' : 'info'">{{ row.status || 'unknown' }}</el-tag></template></el-table-column>
          <el-table-column label="操作" width="220"><template #default="{ row }"><el-button size="small" type="success" plain @click="handleStartCo(row.id)">启动</el-button><el-button size="small" type="warning" plain @click="handleStopCo(row.id)">停止</el-button><el-button size="small" type="danger" plain @click="handleDeleteCo(row.id)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="模板" name="templates">
        <div class="toolbar"><strong>Compose 模板</strong><el-button type="primary" @click="openTDialog">添加模板</el-button></div>
        <el-table :data="templateList" v-loading="tLoading" stripe border empty-text="暂无模板">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="description" label="描述" min-width="240" show-overflow-tooltip />
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" plain @click="handleDeleteT(row.id)">删除</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="清理" name="config">
        <el-alert title="清理操作会删除未使用的 Docker 资源，请确认没有业务依赖后再执行。" type="warning" show-icon :closable="false" />
        <div class="prune-actions">
          <el-button type="warning" plain @click="handlePrune('containers')">清理停止的容器</el-button>
          <el-button type="warning" plain @click="handlePrune('images')">清理未使用镜像</el-button>
          <el-button type="warning" plain @click="handlePrune('volumes')">清理未使用卷</el-button>
          <el-button type="danger" plain @click="handlePrune('all')">一键清理全部</el-button>
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="cDialog" title="创建容器" width="640px">
      <el-form :model="cForm" label-width="100px">
        <el-form-item label="名称" required><el-input v-model="cForm.name" placeholder="my-nginx" /></el-form-item>
        <el-form-item label="镜像" required><el-input v-model="cForm.image" placeholder="nginx:latest" /></el-form-item>
        <el-form-item label="端口映射"><el-input v-model="cForm.ports" type="textarea" :rows="2" placeholder="8080:80，多个可换行/逗号/分号分隔" /></el-form-item>
        <el-form-item label="卷挂载"><el-input v-model="cForm.volumes" type="textarea" :rows="2" placeholder="/host/data:/container/data" /></el-form-item>
        <el-form-item label="环境变量"><el-input v-model="cForm.env" type="textarea" :rows="2" placeholder="KEY=VALUE" /></el-form-item>
        <el-form-item label="网络"><el-input v-model="cForm.network" placeholder="bridge 或自定义网络，可留空" /></el-form-item>
        <el-form-item label="重启策略"><el-select v-model="cForm.restartPolicy"><el-option label="不设置" value="" /><el-option label="no" value="no" /><el-option label="always" value="always" /><el-option label="unless-stopped" value="unless-stopped" /><el-option label="on-failure" value="on-failure" /></el-select></el-form-item>
        <el-form-item label="启动命令"><el-input v-model="cForm.command" placeholder="可选，例如 sleep infinity" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="cDialog = false">取消</el-button><el-button type="primary" :loading="cSubmitting" @click="handleCreate">创建并启动</el-button></template>
    </el-dialog>

    <el-dialog v-model="logDialog" :title="logTitle" width="820px">
      <el-skeleton v-if="logLoading" :rows="8" animated />
      <pre v-else class="log-box">{{ logContent || '暂无日志' }}</pre>
    </el-dialog>

    <el-dialog v-model="pullDialog" title="拉取镜像" width="520px">
      <el-input v-model="pullImageName" placeholder="nginx:latest" />
      <template #footer><el-button @click="pullDialog = false">取消</el-button><el-button type="primary" :loading="pullLoading" @click="handlePullImage">拉取</el-button></template>
    </el-dialog>

    <el-dialog v-model="rDialog" title="添加镜像仓库" width="520px">
      <el-form :model="rForm" label-width="90px">
        <el-form-item label="名称"><el-input v-model="rForm.name" placeholder="Docker Hub" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="rForm.url" placeholder="https://registry-1.docker.io" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="rForm.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="rForm.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer><el-button @click="rDialog = false">取消</el-button><el-button type="primary" @click="handleCreateR">添加</el-button></template>
    </el-dialog>

    <el-dialog v-model="coDialog" title="添加 Compose 项目" width="560px">
      <el-form :model="coForm" label-width="100px">
        <el-form-item label="名称"><el-input v-model="coForm.name" placeholder="my-app" /></el-form-item>
        <el-form-item label="文件路径"><el-input v-model="coForm.path" placeholder="/path/to/docker-compose.yml" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="coDialog = false">取消</el-button><el-button type="primary" @click="handleCreateCo">添加</el-button></template>
    </el-dialog>

    <el-dialog v-model="tDialog" title="添加 Compose 模板" width="680px">
      <el-form :model="tForm" label-width="80px">
        <el-form-item label="名称"><el-input v-model="tForm.name" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="tForm.description" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="tForm.content" type="textarea" :rows="12" placeholder="粘贴 docker-compose.yml 内容" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="tDialog = false">取消</el-button><el-button type="primary" @click="handleCreateT">添加</el-button></template>
    </el-dialog>
  </div>
</template>

<style scoped>
.docker-page { display: flex; flex-direction: column; gap: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 4px 4px 0; }
.page-header h2 { margin: 0; font-size: 22px; font-weight: 700; color: var(--el-text-color-primary); }
.page-header p { margin: 6px 0 0; color: var(--el-text-color-secondary); }
.header-actions, .toolbar { display: flex; align-items: center; gap: 12px; }
.toolbar { justify-content: space-between; margin-bottom: 12px; }
.top-tabs { background: var(--el-bg-color); border-radius: 14px; padding: 4px 16px 16px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
.top-tabs :deep(.el-tabs__content) { padding-top: 12px; }
.stat-row { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }
.stat-card :deep(.el-card__body) { display: flex; align-items: center; gap: 16px; }
.stat-icon { width: 48px; height: 48px; border-radius: 14px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 22px; flex-shrink: 0; }
.bg-blue { background: linear-gradient(135deg, #3370ff, #5b8def); }
.bg-orange { background: linear-gradient(135deg, #ff7d00, #ffaa44); }
.bg-indigo { background: linear-gradient(135deg, #4f46e5, #818cf8); }
.bg-pink { background: linear-gradient(135deg, #eb2f96, #f5a3d5); }
.stat-body { min-width: 0; }
.stat-label { font-size: 14px; color: var(--el-text-color-secondary); }
.stat-big { margin-top: 6px; font-size: 30px; font-weight: 700; color: #3370ff; line-height: 1; }
.stat-sub { margin-top: 6px; font-size: 12px; color: var(--el-text-color-secondary); }
.info-row { display: grid; grid-template-columns: 1.2fr 0.8fr; gap: 16px; margin-top: 16px; }
.section-card { border-radius: 14px; }
.section-hdr { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.disk-row { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.disk-item { padding: 12px; border-radius: 10px; background: var(--el-fill-color-lighter); }
.disk-label { color: var(--el-text-color-secondary); font-size: 13px; }
.disk-size { margin-top: 8px; font-size: 18px; font-weight: 700; }
.disk-reclaim { margin-top: 4px; font-size: 12px; color: #ff7d00; }
.config-item { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 12px 0; border-bottom: 1px solid var(--el-border-color-lighter); }
.config-item:last-child { border-bottom: 0; }
.pagination { margin-top: 12px; justify-content: flex-end; }
.prune-actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 16px; }
.log-box { margin: 0; padding: 16px; min-height: 260px; max-height: 560px; overflow: auto; border-radius: 8px; background: #111827; color: #d1d5db; font-size: 13px; line-height: 1.6; }
code { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; }
@media (max-width: 1200px) { .stat-row, .disk-row { grid-template-columns: repeat(2, minmax(0, 1fr)); } .info-row { grid-template-columns: 1fr; } }
@media (max-width: 768px) { .page-header { flex-direction: column; align-items: stretch; } .stat-row, .disk-row { grid-template-columns: 1fr; } }
</style>
