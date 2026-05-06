<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getServiceVersions, switchServiceVersion, type ServiceVersionInfo } from '@/api/version'
import { ElMessage, ElMessageBox } from 'element-plus'

const services = ref<ServiceVersionInfo[]>([])
const loading = ref(false)
const switching = ref<Record<string, boolean>>({})

// 进度对话框
const progressVisible = ref(false)
const progressTaskId = ref('')
const progressPercent = ref(0)
const progressMessage = ref('')
const progressLogs = ref<string[]>([])
const progressStatus = ref<'running' | 'success' | 'failed'>('running')
let progressTimer: ReturnType<typeof setInterval> | null = null

const serviceIcons: Record<string, string> = {
  nginx: '🌐',
  mysql: '🐬',
  postgresql: '🐘',
  redis: '🔴',
}

const serviceColors: Record<string, string> = {
  nginx: '#009639',
  mysql: '#4479A1',
  postgresql: '#336791',
  redis: '#DC382D',
}

async function fetchVersions() {
  loading.value = true
  try {
    const res = await getServiceVersions()
    services.value = res.data || []
  } finally {
    loading.value = false
  }
}

async function handleSwitch(svc: ServiceVersionInfo, version: string) {
  try {
    await ElMessageBox.confirm(
      `确定要将 ${svc.name} 从 ${svc.current_version} 切换到 ${version} 吗？<br><br>` +
      `<span style="color:#e6a23c">⚠️ 切换过程中服务会短暂中断，请确保没有关键业务在运行。</span>`,
      '版本切换确认',
      {
        confirmButtonText: '确定切换',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true,
      }
    )

    switching.value[svc.type] = true
    const res = await switchServiceVersion(svc.type, version)
    if (res.data?.task_id) {
      progressTaskId.value = res.data.task_id
      progressVisible.value = true
      progressPercent.value = 0
      progressMessage.value = '准备中...'
      progressLogs.value = []
      progressStatus.value = 'running'
      startProgressPolling()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.message || '操作失败')
    }
  } finally {
    switching.value[svc.type] = false
  }
}

function startProgressPolling() {
  if (progressTimer) clearInterval(progressTimer)
  progressTimer = setInterval(async () => {
    try {
      const res = await fetch(`/api/installer/tasks/${progressTaskId.value}`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
      })
      const data = await res.json()
      if (data.code === 200 && data.data) {
        progressPercent.value = data.data.progress || 0
        progressMessage.value = data.data.message || ''
        progressLogs.value = data.data.logs || []
        progressStatus.value = data.data.status

        if (data.data.status === 'success' || data.data.status === 'failed') {
          if (progressTimer) {
            clearInterval(progressTimer)
            progressTimer = null
          }
          if (data.data.status === 'success') {
            ElMessage.success('版本切换成功！')
            fetchVersions()
          } else {
            ElMessage.error('版本切换失败: ' + data.data.message)
          }
        }
      }
    } catch { /* */ }
  }, 1000)
}

function closeProgress() {
  progressVisible.value = false
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

function isCurrentVersion(svc: ServiceVersionInfo, version: string): boolean {
  const current = svc.current_version
  return current.startsWith(version) || current === version
}

function getVersionTagType(svc: ServiceVersionInfo, version: string): string {
  if (isCurrentVersion(svc, version)) return 'success'
  if (svc.installed_versions.some(v => v === version)) return 'warning'
  return ''
}

function getVersionTagText(svc: ServiceVersionInfo, version: string): string {
  if (isCurrentVersion(svc, version)) return '当前'
  if (svc.installed_versions.some(v => v === version)) return '已安装'
  return '可安装'
}

onMounted(fetchVersions)
onUnmounted(() => {
  if (progressTimer) clearInterval(progressTimer)
})
</script>

<template>
  <div class="version-page">
    <div class="page-header">
      <div class="page-title">
        <span style="font-size: 20px; font-weight: bold">🔧 服务版本管理</span>
        <span class="page-desc">管理 Nginx、MySQL、PostgreSQL、Redis 的版本切换</span>
      </div>
      <el-button @click="fetchVersions" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <div class="service-grid" v-loading="loading">
      <div class="service-card" v-for="svc in services" :key="svc.type">
        <div class="card-header" :style="{ borderBottomColor: serviceColors[svc.type] }">
          <div class="svc-icon" :style="{ background: serviceColors[svc.type] }">
            {{ serviceIcons[svc.type] }}
          </div>
          <div class="svc-info">
            <div class="svc-name">{{ svc.name }}</div>
            <div class="svc-version" v-if="svc.installed">
              v{{ svc.current_version }}
              <el-tag :type="svc.running ? 'success' : 'warning'" size="small" style="margin-left: 8px">
                {{ svc.running ? '运行中' : '已停止' }}
              </el-tag>
            </div>
            <div class="svc-version" v-else>
              <el-tag type="info" size="small">未安装</el-tag>
            </div>
          </div>
        </div>

        <div class="card-body" v-if="svc.installed">
          <div class="section-title">可用版本</div>
          <div class="version-list">
            <div
              v-for="ver in svc.available_versions"
              :key="ver"
              class="version-item"
              :class="{ active: isCurrentVersion(svc, ver) }"
            >
              <div class="version-info">
                <span class="ver-number">{{ ver }}</span>
                <el-tag :type="getVersionTagType(svc, ver)" size="small" effect="plain">
                  {{ getVersionTagText(svc, ver) }}
                </el-tag>
              </div>
              <el-button
                v-if="!isCurrentVersion(svc, ver)"
                type="primary"
                size="small"
                :loading="switching[svc.type]"
                @click="handleSwitch(svc, ver)"
              >
                🔄 切换到 {{ ver }}
              </el-button>
              <el-tag v-else type="success" size="small" effect="dark">当前版本</el-tag>
            </div>
          </div>

          <div class="section-title" v-if="svc.installed_versions.length > 0">已安装版本</div>
          <div class="installed-list" v-if="svc.installed_versions.length > 0">
            <el-tag
              v-for="ver in svc.installed_versions"
              :key="ver"
              :type="isCurrentVersion(svc, ver) ? 'success' : 'warning'"
              size="small"
              effect="light"
              style="margin: 2px 4px"
            >
              {{ ver }} {{ isCurrentVersion(svc, ver) ? '(当前)' : '' }}
            </el-tag>
          </div>
        </div>

        <div class="card-body empty" v-else>
          <el-empty description="服务未安装" :image-size="60">
            <el-button type="primary" size="small">前往安装</el-button>
          </el-empty>
        </div>
      </div>
    </div>

    <!-- 进度对话框 -->
    <el-dialog
      v-model="progressVisible"
      title="版本切换进度"
      width="600px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="closeProgress"
    >
      <div class="progress-container">
        <el-progress
          :percentage="progressPercent"
          :status="progressStatus === 'success' ? 'success' : progressStatus === 'failed' ? 'exception' : undefined"
          :stroke-width="10"
          striped
          striped-flow
        />
        <div class="progress-message">{{ progressMessage }}</div>

        <div class="progress-logs" ref="logContainer">
          <div v-for="(log, i) in progressLogs" :key="i" class="log-line">
            {{ log }}
          </div>
        </div>
      </div>

      <template #footer>
        <el-button
          v-if="progressStatus === 'success' || progressStatus === 'failed'"
          type="primary"
          @click="closeProgress"
        >
          关闭
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.version-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-desc {
  font-size: 13px;
  color: #86909c;
}

.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: 20px;
}

.service-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  transition: box-shadow 0.2s;
}

.service-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  border-bottom: 3px solid;
}

.svc-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: #fff;
  flex-shrink: 0;
}

.svc-info {
  flex: 1;
  min-width: 0;
}

.svc-name {
  font-size: 17px;
  font-weight: 600;
  color: #1d2129;
}

.svc-version {
  font-size: 13px;
  color: #86909c;
  margin-top: 4px;
  display: flex;
  align-items: center;
}

.card-body {
  padding: 16px 20px;
}

.card-body.empty {
  padding: 20px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: #4e5969;
  margin-bottom: 10px;
  margin-top: 12px;
}

.section-title:first-child {
  margin-top: 0;
}

.version-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.version-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: #f7f8fa;
  border-radius: 8px;
  transition: background 0.15s;
}

.version-item:hover {
  background: #f0f2f5;
}

.version-item.active {
  background: #e8f7ee;
  border: 1px solid #b7eb8f;
}

.version-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ver-number {
  font-size: 14px;
  font-weight: 600;
  color: #1d2129;
}

.installed-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px 0;
}

.progress-container {
  padding: 8px 0;
}

.progress-message {
  text-align: center;
  margin: 12px 0;
  color: #4e5969;
  font-size: 14px;
}

.progress-logs {
  max-height: 300px;
  overflow-y: auto;
  background: #1e1e1e;
  border-radius: 8px;
  padding: 12px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  margin-top: 12px;
}

.log-line {
  color: #d4d4d4;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-line:contains('✅') {
  color: #4ec9b0;
}

.log-line:contains('❌') {
  color: #f48771;
}
</style>
