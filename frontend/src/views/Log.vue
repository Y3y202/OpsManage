<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getLogs, getLogSources, clearLogs, getSystemLogs, getSSHLogs } from '@/api/log'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('app')
const logs = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const filterLevel = ref('')
const filterSource = ref('')
const filterKeyword = ref('')
const sources = ref<string[]>([])

const sysLogType = ref('syslog')
const sysLogContent = ref('')
const sysLogLoading = ref(false)

const sysLogTypes = [
  { label: '系统日志', value: 'syslog' },
  { label: '认证日志', value: 'auth' },
  { label: 'Nginx访问', value: 'nginx' },
  { label: 'Nginx错误', value: 'nginx_error' }
]

// SSH 登录日志
const sshLogs = ref<any[]>([])
const sshTotal = ref(0)
const sshLoading = ref(false)
const sshPage = ref(1)
const sshFilterEvent = ref('')
const sshFilterKeyword = ref('')

const sshEventOptions = [
  { label: '登录成功', value: 'accepted' },
  { label: '登录失败', value: 'failed' },
  { label: '连接关闭', value: 'closed' },
  { label: '会话开启', value: 'session_open' },
  { label: '会话关闭', value: 'session_close' }
]

function sshEventType(event: string) {
  switch (event) {
    case 'accepted': return 'success'
    case 'failed': return 'danger'
    case 'closed': return 'warning'
    case 'session_open': return ''
    case 'session_close': return 'info'
    default: return 'info'
  }
}

function sshEventLabel(event: string) {
  const found = sshEventOptions.find(e => e.value === event)
  return found ? found.label : event
}

async function fetchLogs() {
  loading.value = true
  const params: any = { page: page.value, page_size: 20 }
  if (filterLevel.value) params.level = filterLevel.value
  if (filterSource.value) params.source = filterSource.value
  if (filterKeyword.value) params.keyword = filterKeyword.value
  const res = await getLogs(params)
  logs.value = res.data.list
  total.value = res.data.total
  loading.value = false
}

async function fetchSources() {
  const res = await getLogSources()
  sources.value = res.data || []
}

async function fetchSystemLogs() {
  sysLogLoading.value = true
  try {
    const res = await getSystemLogs(sysLogType.value)
    sysLogContent.value = res.data.content || '无内容'
  } catch {
    sysLogContent.value = '读取失败'
  }
  sysLogLoading.value = false
}

async function fetchSSHLogs() {
  sshLoading.value = true
  try {
    const params: any = { page: sshPage.value, page_size: 20, lines: 1000 }
    if (sshFilterEvent.value) params.event = sshFilterEvent.value
    if (sshFilterKeyword.value) params.keyword = sshFilterKeyword.value
    const res = await getSSHLogs(params)
    sshLogs.value = res.data.list
    sshTotal.value = res.data.total
  } catch {
    sshLogs.value = []
    sshTotal.value = 0
  }
  sshLoading.value = false
}

async function handleClear() {
  await ElMessageBox.confirm('确定清空所有日志?', '提示')
  await clearLogs(filterSource.value || undefined)
  ElMessage.success('已清空')
  fetchLogs()
}

function levelType(level: string) {
  if (level === 'error') return 'danger'
  if (level === 'warn') return 'warning'
  return 'info'
}

onMounted(() => {
  fetchLogs()
  fetchSources()
  fetchSSHLogs()
})
</script>

<template>
  <el-card>
    <el-tabs v-model="activeTab">
      <el-tab-pane label="应用日志" name="app">
        <div style="margin-bottom: 16px; display: flex; gap: 8px; align-items: center">
          <el-select v-model="filterLevel" placeholder="级别" clearable style="width: 120px" @change="fetchLogs">
            <el-option label="info" value="info" />
            <el-option label="warn" value="warn" />
            <el-option label="error" value="error" />
          </el-select>
          <el-select v-model="filterSource" placeholder="来源" clearable style="width: 140px" @change="fetchLogs">
            <el-option v-for="s in sources" :key="s" :label="s" :value="s" />
          </el-select>
          <el-input v-model="filterKeyword" placeholder="关键词" clearable style="width: 200px" @keyup.enter="fetchLogs" />
          <el-button @click="fetchLogs">搜索</el-button>
          <div style="flex: 1" />
          <el-button type="danger" @click="handleClear">清空日志</el-button>
        </div>
        <el-table :data="logs" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="level" label="级别" width="80">
            <template #default="{ row }">
              <el-tag :type="levelType(row.level)">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="120" />
          <el-table-column prop="message" label="消息" show-overflow-tooltip />
          <el-table-column prop="created_at" label="时间" width="180" />
        </el-table>
        <el-pagination style="margin-top: 16px" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="fetchLogs" />
      </el-tab-pane>

      <el-tab-pane label="SSH 登录日志" name="ssh">
        <div style="margin-bottom: 16px; display: flex; gap: 8px; align-items: center">
          <el-select v-model="sshFilterEvent" placeholder="事件类型" clearable style="width: 140px" @change="sshPage=1;fetchSSHLogs()">
            <el-option v-for="t in sshEventOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
          <el-input v-model="sshFilterKeyword" placeholder="搜索 用户/IP" clearable style="width: 200px" @keyup.enter="sshPage=1;fetchSSHLogs()" />
          <el-button @click="sshPage=1;fetchSSHLogs()">搜索</el-button>
          <el-button @click="sshFilterEvent='';sshFilterKeyword='';sshPage=1;fetchSSHLogs()">重置</el-button>
        </div>
        <el-table :data="sshLogs" v-loading="sshLoading" stripe>
          <el-table-column prop="time" label="时间" width="240" />
          <el-table-column prop="event" label="事件" width="120">
            <template #default="{ row }">
              <el-tag :type="sshEventType(row.event)">{{ sshEventLabel(row.event) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="user" label="用户" width="100" />
          <el-table-column prop="ip" label="IP 地址" width="160" />
          <el-table-column prop="port" label="端口" width="80" />
          <el-table-column prop="message" label="原始日志" show-overflow-tooltip />
        </el-table>
        <el-pagination style="margin-top: 16px" v-model:current-page="sshPage" :total="sshTotal" :page-size="20" layout="prev, pager, next" @current-change="fetchSSHLogs" />
      </el-tab-pane>

      <el-tab-pane label="系统日志" name="system">
        <div style="margin-bottom: 16px; display: flex; gap: 8px">
          <el-select v-model="sysLogType" style="width: 160px">
            <el-option v-for="t in sysLogTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
          <el-button type="primary" @click="fetchSystemLogs" :loading="sysLogLoading">加载</el-button>
        </div>
        <pre style="background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 4px; max-height: 500px; overflow: auto; font-size: 13px; white-space: pre-wrap">{{ sysLogContent }}</pre>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>
