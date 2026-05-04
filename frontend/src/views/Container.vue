<script setup lang="ts">
import { ref, onMounted, reactive, nextTick } from 'vue'
import * as api from '@/api/container'
import { ElMessage, ElMessageBox } from 'element-plus'

const overview = ref<any>({})
const activeTab = ref('overview')

// 容器
const containerList = ref([]); const containerTotal = ref(0); const cPage = ref(1); const cLoading = ref(false)
const cDialog = ref(false); const cForm = reactive({ name: '', image: '', ports: '', volumes: '', env: '' })
const logDialog = ref(false); const logContent = ref('')

// 镜像
const imageList = ref([]); const iLoading = ref(false)

// 网络
const networkList = ref([]); const nLoading = ref(false)

// 存储卷
const volumeList = ref([]); const vLoading = ref(false)

// 仓库
const registryList = ref([]); const rDialog = ref(false)
const rForm = reactive({ name: '', url: '', username: '', password: '' })

// 编排
const composeList = ref([]); const coDialog = ref(false)
const coForm = reactive({ name: '', path: '' })

// 编排模板
const templateList = ref([]); const tDialog = ref(false)
const tForm = reactive({ name: '', description: '', content: '' })

async function fetchOverview() {
  const o: any = await api.getDockerOverview()
  await nextTick()
  overview.value = o.data
}

async function fetchContainers() {
  cLoading.value = true
  try {
    const r: any = await api.getContainers({ page: cPage.value, page_size: 20 })
    containerList.value = r.data.list; containerTotal.value = r.data.total
  } finally { cLoading.value = false }
}
async function fetchImages() {
  iLoading.value = true
  try { imageList.value = (await api.listImages() as any).data || [] } finally { iLoading.value = false }
}
async function fetchNetworks() {
  nLoading.value = true
  try { networkList.value = (await api.listDockerNetworks() as any).data || [] } finally { nLoading.value = false }
}
async function fetchVolumes() {
  vLoading.value = true
  try { volumeList.value = (await api.listDockerVolumes() as any).data || [] } finally { vLoading.value = false }
}
async function fetchRegistries() {
  registryList.value = (await api.listRegistries() as any).data || []
}
async function fetchCompose() {
  composeList.value = (await api.listComposeProjects() as any).data || []
}
async function fetchTemplates() {
  templateList.value = (await api.listComposeTemplates() as any).data || []
}

function onTabChange(tab: string) {
  if (tab === 'containers') fetchContainers()
  else if (tab === 'images') fetchImages()
  else if (tab === 'networks') fetchNetworks()
  else if (tab === 'volumes') fetchVolumes()
  else if (tab === 'registries') fetchRegistries()
  else if (tab === 'compose') fetchCompose()
  else if (tab === 'templates') fetchTemplates()
}

// 容器操作
function openCDialog() { Object.assign(cForm, { name: '', image: '', ports: '', volumes: '', env: '' }); cDialog.value = true }
async function handleCreate() { await api.createContainer(cForm); ElMessage.success('创建成功'); cDialog.value = false; fetchContainers(); fetchOverview() }
async function handleDeleteC(row: any) { await ElMessageBox.confirm('确定删除该容器?'); await api.deleteContainer(row.id); ElMessage.success('已删除'); fetchContainers(); fetchOverview() }
async function handleStart(row: any) { await api.startContainer(row.id); ElMessage.success('已启动'); fetchContainers() }
async function handleStop(row: any) { await api.stopContainer(row.id); ElMessage.success('已停止'); fetchContainers() }
async function handleRestart(row: any) { await api.restartContainer(row.id); ElMessage.success('已重启'); fetchContainers() }
async function handleLogs(row: any) { const r: any = await api.getContainerLogs(row.id); logContent.value = r.data.logs; logDialog.value = true }

// 镜像操作
async function handleRemoveImage(name: string) { await ElMessageBox.confirm('确定删除该镜像?'); await api.removeImage(name); ElMessage.success('已删除'); fetchImages(); fetchOverview() }

// 网络操作
async function handleRemoveNetwork(id: string) { await ElMessageBox.confirm('确定删除该网络?'); await api.removeNetwork(id); ElMessage.success('已删除'); fetchNetworks(); fetchOverview() }

// 存储卷操作
async function handleRemoveVolume(name: string) { await ElMessageBox.confirm('确定删除该存储卷?'); await api.removeVolume(name); ElMessage.success('已删除'); fetchVolumes(); fetchOverview() }

// 仓库操作
function openRDialog() { Object.assign(rForm, { name: '', url: '', username: '', password: '' }); rDialog.value = true }
async function handleCreateR() { await api.createRegistry(rForm); ElMessage.success('已添加'); rDialog.value = false; fetchRegistries(); fetchOverview() }
async function handleDeleteR(id: number) { await ElMessageBox.confirm('确定删除该仓库?'); await api.deleteRegistry(id); ElMessage.success('已删除'); fetchRegistries(); fetchOverview() }

// 编排操作
function openCoDialog() { Object.assign(coForm, { name: '', path: '' }); coDialog.value = true }
async function handleCreateCo() { await api.createComposeProject(coForm); ElMessage.success('已添加'); coDialog.value = false; fetchCompose(); fetchOverview() }
async function handleDeleteCo(id: number) { await ElMessageBox.confirm('确定删除该编排?'); await api.deleteComposeProject(id); ElMessage.success('已删除'); fetchCompose(); fetchOverview() }
async function handleStartCo(id: number) { await api.startComposeProject(id); ElMessage.success('已启动'); fetchCompose(); fetchOverview() }
async function handleStopCo(id: number) { await api.stopComposeProject(id); ElMessage.success('已停止'); fetchCompose(); fetchOverview() }

// 模板操作
function openTDialog() { Object.assign(tForm, { name: '', description: '', content: '' }); tDialog.value = true }
async function handleCreateT() { await api.createComposeTemplate(tForm); ElMessage.success('已添加'); tDialog.value = false; fetchTemplates(); fetchOverview() }
async function handleDeleteT(id: number) { await ElMessageBox.confirm('确定删除该模板?'); await api.deleteComposeTemplate(id); ElMessage.success('已删除'); fetchTemplates(); fetchOverview() }

// 清理
async function handlePrune(type: string) {
  await ElMessageBox.confirm(`确定清理 ${type === 'all' ? '全部' : type}？此操作不可恢复`, '警告', { type: 'warning' })
  const r: any = await api.pruneDocker(type)
  ElMessage.success(r.data.msg || '清理完成')
  fetchOverview()
}

onMounted(async () => { await fetchOverview(); fetchContainers() })
</script>

<template>
  <div class="docker-page fade-in-up">
    <!-- Tab 导航（顶部） -->
    <el-tabs v-model="activeTab" @tab-change="onTabChange" class="top-tabs">
      <!-- 概览 Tab -->
      <el-tab-pane label="概览" name="overview">
        <div class="stat-row">
          <div class="stat-card">
            <div class="stat-icon bg-blue"><el-icon size="22"><Box /></el-icon></div>
            <div class="stat-body">
              <div class="stat-label">容器</div>
              <div class="stat-sub"><span>所有 <b>{{ overview.containers_total || 0 }}</b></span><span>已启动 <b style="color:#00b42a">{{ overview.containers_running || 0 }}</b></span></div>
              <div class="stat-big">{{ overview.containers_total || 0 }}</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon bg-purple"><el-icon size="22"><Files /></el-icon></div>
            <div class="stat-body"><div class="stat-label">编排</div><div class="stat-big">{{ overview.compose || 0 }}</div></div>
          </div>
          <div class="stat-card">
            <div class="stat-icon bg-cyan"><el-icon size="22"><Document /></el-icon></div>
            <div class="stat-body"><div class="stat-label">编排模版</div><div class="stat-big">{{ overview.compose_templates || 0 }}</div></div>
          </div>
          <div class="stat-card">
            <div class="stat-icon bg-orange"><el-icon size="22"><Picture /></el-icon></div>
            <div class="stat-body"><div class="stat-label">镜像</div><div class="stat-big">{{ overview.images || 0 }}</div></div>
          </div>
        </div>
        <div class="stat-row">
          <div class="stat-card">
            <div class="stat-icon bg-teal"><el-icon size="22"><Connection /></el-icon></div>
            <div class="stat-body"><div class="stat-label">镜像仓库</div><div class="stat-big">{{ overview.registries || 0 }}</div></div>
          </div>
          <div class="stat-card">
            <div class="stat-icon bg-indigo"><el-icon size="22"><Share /></el-icon></div>
            <div class="stat-body"><div class="stat-label">网络</div><div class="stat-big">{{ overview.networks || 0 }}</div></div>
          </div>
          <div class="stat-card">
            <div class="stat-icon bg-pink"><el-icon size="22"><FolderOpened /></el-icon></div>
            <div class="stat-body"><div class="stat-label">存储卷</div><div class="stat-big">{{ overview.volumes || 0 }}</div></div>
          </div>
          <div class="stat-card stat-placeholder"></div>
        </div>
        <div class="info-row">
          <el-card class="section-card" shadow="never">
            <template #header><div class="section-hdr"><el-icon color="#3370ff"><DataLine /></el-icon><span class="section-title">磁盘占用</span></div></template>
            <div class="disk-row">
              <div class="disk-item" v-for="(item, key) in overview.disk_usage" :key="key">
                <div class="disk-label"><span class="disk-type">{{ key === 'images' ? '镜像' : key === 'containers' ? '容器' : key === 'volumes' ? '本地存储卷' : '构建缓存' }}</span>
                  <span class="disk-reclaim" v-if="item.reclaimable && !item.reclaimable.startsWith('0')">可释放 {{ item.reclaimable }}</span>
                </div>
                <div class="disk-size">{{ item.size || '0B' }}</div>
              </div>
            </div>
          </el-card>
          <el-card class="section-card" shadow="never">
            <template #header><div class="section-hdr"><el-icon color="#3370ff"><Setting /></el-icon><span class="section-title">配置</span></div></template>
            <div class="config-row">
              <div class="config-item"><span class="config-key">Socket路径</span><span class="config-val mono">{{ overview.socket || 'unix:///var/run/docker.sock' }}</span></div>
              <div class="config-item"><span class="config-key">镜像加速</span><el-link type="primary" :underline="false">去修改</el-link></div>
            </div>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- 容器 -->
      <el-tab-pane label="容器" name="containers">
        <div style="margin-bottom:12px; display:flex; justify-content:space-between">
          <span style="font-weight:600">容器列表</span>
          <el-button type="primary" size="small" @click="openCDialog">创建容器</el-button>
        </div>
        <el-table :data="containerList" v-loading="cLoading" stripe size="small">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="image" label="镜像" min-width="160" />
          <el-table-column prop="ports" label="端口" min-width="100" />
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }"><el-tag :type="row.status === 'running' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="280">
            <template #default="{ row }">
              <el-button-group size="small">
                <el-button type="success" plain @click="handleStart(row)">启动</el-button>
                <el-button type="warning" plain @click="handleStop(row)">停止</el-button>
                <el-button @click="handleRestart(row)">重启</el-button>
                <el-button @click="handleLogs(row)">日志</el-button>
                <el-button type="danger" plain @click="handleDeleteC(row)">删除</el-button>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination style="margin-top:12px" v-model:current-page="cPage" :total="containerTotal" :page-size="20" layout="prev,pager,next" @current-change="fetchContainers" />
      </el-tab-pane>

      <!-- 编排 -->
      <el-tab-pane label="编排" name="compose">
        <div style="margin-bottom:12px; display:flex; justify-content:space-between">
          <span style="font-weight:600">编排项目</span>
          <el-button type="primary" size="small" @click="openCoDialog">添加编排</el-button>
        </div>
        <el-table :data="composeList" stripe size="small" empty-text="暂无编排项目">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="path" label="路径" min-width="200" />
          <el-table-column prop="services" label="服务数" width="80" />
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }"><el-tag :type="row.status === 'running' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button size="small" type="success" plain @click="handleStartCo(row.id)">启动</el-button>
              <el-button size="small" type="warning" plain @click="handleStopCo(row.id)">停止</el-button>
              <el-button size="small" type="danger" plain @click="handleDeleteCo(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 编排模版 -->
      <el-tab-pane label="编排模版" name="templates">
        <div style="margin-bottom:12px; display:flex; justify-content:space-between">
          <span style="font-weight:600">编排模板</span>
          <el-button type="primary" size="small" @click="openTDialog">添加模板</el-button>
        </div>
        <el-table :data="templateList" stripe size="small" empty-text="暂无模板">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="description" label="描述" min-width="200" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" plain @click="handleDeleteT(row.id)">删除</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 镜像 -->
      <el-tab-pane label="镜像" name="images">
        <div style="margin-bottom:12px; display:flex; justify-content:space-between">
          <span style="font-weight:600">镜像列表</span>
        </div>
        <el-table :data="imageList" v-loading="iLoading" stripe size="small" empty-text="暂无镜像">
          <el-table-column prop="name" label="镜像名" min-width="200" />
          <el-table-column prop="id" label="IMAGE ID" min-width="120" />
          <el-table-column prop="size" label="大小" width="120" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" plain @click="handleRemoveImage(row.name)">删除</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 网络 -->
      <el-tab-pane label="网络" name="networks">
        <el-table :data="networkList" v-loading="nLoading" stripe size="small" empty-text="暂无网络">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="id" label="NETWORK ID" min-width="140" />
          <el-table-column prop="driver" label="驱动" width="100" />
          <el-table-column prop="scope" label="范围" width="80" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" plain @click="handleRemoveNetwork(row.id)">删除</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 存储卷 -->
      <el-tab-pane label="存储卷" name="volumes">
        <el-table :data="volumeList" v-loading="vLoading" stripe size="small" empty-text="暂无存储卷">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="driver" label="驱动" width="100" />
          <el-table-column prop="mountpoint" label="挂载点" min-width="200" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" plain @click="handleRemoveVolume(row.name)">删除</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 仓库 -->
      <el-tab-pane label="仓库" name="registries">
        <div style="margin-bottom:12px; display:flex; justify-content:space-between">
          <span style="font-weight:600">镜像仓库</span>
          <el-button type="primary" size="small" @click="openRDialog">添加仓库</el-button>
        </div>
        <el-table :data="registryList" stripe size="small" empty-text="暂无仓库">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="url" label="地址" min-width="200" />
          <el-table-column prop="username" label="用户名" width="120" />
          <el-table-column prop="is_default" label="默认" width="60">
            <template #default="{ row }"><el-tag v-if="row.is_default" type="success" size="small">是</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" plain @click="handleDeleteR(row.id)">删除</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 配置 -->
      <el-tab-pane label="配置" name="config">
        <div style="font-weight:600; margin-bottom:12px">Docker 清理</div>
        <div style="display:flex; gap:12px; flex-wrap:wrap">
          <el-button type="warning" plain @click="handlePrune('containers')">清理停止的容器</el-button>
          <el-button type="warning" plain @click="handlePrune('images')">清理未使用镜像</el-button>
          <el-button type="warning" plain @click="handlePrune('volumes')">清理未使用卷</el-button>
          <el-button type="danger" plain @click="handlePrune('all')">一键清理全部</el-button>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 弹窗们 -->
    <el-dialog v-model="cDialog" title="创建容器" width="500px">
      <el-form :model="cForm" label-width="80px">
        <el-form-item label="名称"><el-input v-model="cForm.name" /></el-form-item>
        <el-form-item label="镜像"><el-input v-model="cForm.image" placeholder="nginx:latest" /></el-form-item>
        <el-form-item label="端口"><el-input v-model="cForm.ports" placeholder="8080:80" /></el-form-item>
        <el-form-item label="卷"><el-input v-model="cForm.volumes" placeholder="/data:/data" /></el-form-item>
        <el-form-item label="环境变量"><el-input v-model="cForm.env" placeholder="KEY=VALUE" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="cDialog=false">取消</el-button><el-button type="primary" @click="handleCreate">创建</el-button></template>
    </el-dialog>
    <el-dialog v-model="logDialog" title="容器日志" width="700px">
      <pre style="background:#1e1e1e;color:#d4d4d4;padding:16px;border-radius:4px;max-height:400px;overflow:auto;font-size:13px">{{ logContent }}</pre>
    </el-dialog>
    <el-dialog v-model="rDialog" title="添加镜像仓库" width="500px">
      <el-form :model="rForm" label-width="80px">
        <el-form-item label="名称"><el-input v-model="rForm.name" placeholder="Docker Hub" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="rForm.url" placeholder="https://registry-1.docker.io" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="rForm.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="rForm.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer><el-button @click="rDialog=false">取消</el-button><el-button type="primary" @click="handleCreateR">添加</el-button></template>
    </el-dialog>
    <el-dialog v-model="coDialog" title="添加编排项目" width="500px">
      <el-form :model="coForm" label-width="80px">
        <el-form-item label="名称"><el-input v-model="coForm.name" placeholder="my-app" /></el-form-item>
        <el-form-item label="路径"><el-input v-model="coForm.path" placeholder="/path/to/docker-compose.yml" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="coDialog=false">取消</el-button><el-button type="primary" @click="handleCreateCo">添加</el-button></template>
    </el-dialog>
    <el-dialog v-model="tDialog" title="添加编排模板" width="600px">
      <el-form :model="tForm" label-width="80px">
        <el-form-item label="名称"><el-input v-model="tForm.name" placeholder="Nginx + PHP" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="tForm.description" placeholder="模板说明" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="tForm.content" type="textarea" :rows="10" placeholder="粘贴 docker-compose.yml 内容" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="tDialog=false">取消</el-button><el-button type="primary" @click="handleCreateT">添加</el-button></template>
    </el-dialog>
  </div>
</template>

<style scoped>
.docker-page { display: flex; flex-direction: column; gap: 0; }
.top-tabs { background: #fff; border-radius: 14px; padding: 4px 16px 0; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
.top-tabs :deep(.el-tabs__content) { padding: 16px 4px; }
.top-tabs :deep(.el-tabs__item) { font-size: 15px; font-weight: 600; height: 48px; }
.top-tabs :deep(.el-tabs__item.is-active) { color: #3370ff; }
.stat-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.stat-placeholder { visibility: hidden; }
.stat-card { background: #fff; border-radius: 14px; padding: 20px; display: flex; align-items: center; gap: 16px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); transition: transform 0.2s; }
.stat-card:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,0,0,0.06); }
.stat-icon { width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center; color: #fff; flex-shrink: 0; }
.bg-blue { background: linear-gradient(135deg, #3370ff, #5b8def); }
.bg-purple { background: linear-gradient(135deg, #722ed1, #b37feb); }
.bg-cyan { background: linear-gradient(135deg, #14c9c9, #3fdcdc); }
.bg-orange { background: linear-gradient(135deg, #ff7d00, #ffaa44); }
.bg-teal { background: linear-gradient(135deg, #0fc6c6, #36d9e0); }
.bg-indigo { background: linear-gradient(135deg, #4f46e5, #818cf8); }
.bg-pink { background: linear-gradient(135deg, #eb2f96, #f5a3d5); }
.stat-body { flex: 1; min-width: 0; }
.stat-label { font-size: 14px; font-weight: 600; color: #1d2129; margin-bottom: 4px; }
.stat-sub { font-size: 12px; color: #86909c; display: flex; gap: 12px; margin-bottom: 2px; }
.stat-sub b { color: #1d2129; }
.stat-big { font-size: 28px; font-weight: 700; color: #3370ff; line-height: 1; font-variant-numeric: tabular-nums; }
.section-card { border-radius: 14px !important; border: none !important; box-shadow: 0 2px 8px rgba(0,0,0,0.04) !important; }
.section-hdr { display: flex; align-items: center; gap: 8px; }
.section-title { font-size: 15px; font-weight: 600; color: #1d2129; }
.info-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.disk-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0; }
.disk-item { padding: 16px 20px; border-right: 1px solid #f2f3f5; }
.disk-item:last-child { border-right: none; }
.disk-label { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.disk-type { font-size: 13px; color: #86909c; }
.disk-reclaim { font-size: 12px; color: #ff7d00; }
.disk-size { font-size: 18px; font-weight: 700; color: #1d2129; }
.config-row { display: flex; flex-direction: column; }
.config-item { display: flex; justify-content: space-between; align-items: center; padding: 14px 0; border-bottom: 1px solid #f2f3f5; }
.config-item:last-child { border-bottom: none; }
.config-key { font-size: 14px; color: #86909c; }
.config-val { font-size: 14px; color: #1d2129; font-weight: 500; }
.mono { font-family: 'JetBrains Mono', 'Fira Code', monospace; }
</style>
