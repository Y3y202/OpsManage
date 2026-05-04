<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, shallowRef } from 'vue'
import { getDashboard, getSystemInfo, getSystemStatus } from '@/api/dashboard'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, GaugeChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, GaugeChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const dashboard = ref<any>({})
const sysInfo = ref<any>({})
const sysStatus = ref<any>({})
let timer: ReturnType<typeof setInterval> | null = null

// 历史数据（实时采集）
const cpuHistory = ref<number[]>([])
const memHistory = ref<number[]>([])
const timeLabels = ref<string[]>([])
const maxHistory = 30

function collectData() {
  const now = new Date()
  const label = now.getHours().toString().padStart(2, '0') + ':' + now.getMinutes().toString().padStart(2, '0') + ':' + now.getSeconds().toString().padStart(2, '0')
  timeLabels.value.push(label)
  memHistory.value.push(Math.round(sysStatus.value.memory?.used_percent || 0))
  // 使用后端真实 CPU 使用率
  cpuHistory.value.push(Math.round(sysStatus.value.cpu?.used_percent || 0))
  if (timeLabels.value.length > maxHistory) {
    timeLabels.value.shift()
    cpuHistory.value.shift()
    memHistory.value.shift()
  }
}

async function fetchData() {
  const [d, i, s] = await Promise.all([getDashboard(), getSystemInfo(), getSystemStatus()])
  dashboard.value = d.data
  sysInfo.value = i.data
  sysStatus.value = s.data
  collectData()
}
async function refreshStatus() {
  try {
    const s = await getSystemStatus()
    sysStatus.value = s.data
    collectData()
  } catch { /* */ }
}
onMounted(async () => { await fetchData(); timer = setInterval(refreshStatus, 5000) })
onUnmounted(() => { if (timer) clearInterval(timer) })

function formatGB(mb: number) {
  if (!mb) return '0'
  return (mb / 1024).toFixed(1)
}
function formatMB(mb: number) {
  if (!mb) return '0 MB'
  if (mb > 1024) return (mb / 1024).toFixed(1) + ' GB'
  return mb.toFixed(0) + ' MB'
}

// KPI 卡片
const kpiCards = computed(() => [
  { label: '总服务器数', value: (dashboard.value.websites || 0) + (dashboard.value.databases || 0), sub: `网站 ${dashboard.value.websites || 0} · 数据库 ${dashboard.value.databases || 0}`, color: '#3370ff', bg: 'linear-gradient(135deg, #3370ff 0%, #5b8def 100%)' },
  { label: '正常运行主机', value: (dashboard.value.websites_running || 0) + (dashboard.value.databases_running || 0), sub: '运行中', color: '#00b42a', bg: 'linear-gradient(135deg, #00b42a 0%, #23c343 100%)' },
  { label: '活跃告警数', value: 0, sub: '暂无告警', color: '#ff5733', bg: 'linear-gradient(135deg, #f53f3f 0%, #ff7a70 100%)' },
  { label: '容器实例', value: dashboard.value.containers || 0, sub: `运行中 ${dashboard.value.containers_running || 0}`, color: '#722ed1', bg: 'linear-gradient(135deg, #722ed1 0%, #b37feb 100%)' },
  { label: 'SSH 主机', value: dashboard.value.ssh_accounts || 0, sub: '已配置', color: '#14c9c9', bg: 'linear-gradient(135deg, #14c9c9 0%, #3fdcdc 100%)' },
  { label: '计划任务', value: dashboard.value.tasks || 0, sub: '已配置', color: '#ff7d00', bg: 'linear-gradient(135deg, #ff7d00 0%, #ffaa44 100%)' },
])

// ECharts 配置
const cpuChartOption = computed(() => ({
  tooltip: { trigger: 'axis', backgroundColor: '#1e293b', borderColor: '#334155', textStyle: { color: '#e2e8f0', fontSize: 12 } },
  legend: { data: ['CPU', '内存'], top: 0, right: 0, textStyle: { color: '#64748b', fontSize: 12 } },
  grid: { top: 36, right: 16, bottom: 8, left: 44 },
  xAxis: { type: 'category', data: timeLabels.value, boundaryGap: false, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#94a3b8', fontSize: 10, showMaxLabel: true, showMinLabel: true } },
  yAxis: { type: 'value', min: 0, max: 100, splitLine: { lineStyle: { color: '#f1f5f9', type: 'dashed' } }, axisLabel: { color: '#94a3b8', fontSize: 11, formatter: '{value}%' } },
  series: [
    { name: 'CPU', type: 'line', smooth: true, showSymbol: false, lineStyle: { width: 2.5, color: '#3370ff' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(51,112,255,0.15)' }, { offset: 1, color: 'rgba(51,112,255,0)' }] } }, data: cpuHistory.value },
    { name: '内存', type: 'line', smooth: true, showSymbol: false, lineStyle: { width: 2.5, color: '#00b42a' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(0,180,42,0.15)' }, { offset: 1, color: 'rgba(0,180,42,0)' }] } }, data: memHistory.value },
  ]
}))

const diskGaugeOption = computed(() => {
  const pct = Math.round(sysStatus.value.disk?.used_percent || 0)
  const color = pct > 85 ? '#f53f3f' : pct > 60 ? '#ff7d00' : '#00b42a'
  return {
    series: [{
      type: 'gauge', startAngle: 200, endAngle: -20, min: 0, max: 100,
      pointer: { show: false },
      progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
      axisLine: { lineStyle: { width: 14, color: [[1, '#e5e6eb']] } },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false },
      detail: {
        valueAnimation: true, fontSize: 28, fontWeight: 700, color: '#1d2129',
        formatter: '{value}%',
        offsetCenter: [0, '10%']
      },
      title: { show: true, offsetCenter: [0, '45%'], fontSize: 13, color: '#86909c' },
      data: [{ value: pct, name: '磁盘使用率' }]
    }]
  }
})

const memGaugeOption = computed(() => {
  const pct = Math.round(sysStatus.value.memory?.used_percent || 0)
  const color = pct > 85 ? '#f53f3f' : pct > 60 ? '#ff7d00' : '#00b42a'
  return {
    series: [{
      type: 'gauge', startAngle: 200, endAngle: -20, min: 0, max: 100,
      pointer: { show: false },
      progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
      axisLine: { lineStyle: { width: 14, color: [[1, '#e5e6eb']] } },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false },
      detail: {
        valueAnimation: true, fontSize: 28, fontWeight: 700, color: '#1d2129',
        formatter: '{value}%',
        offsetCenter: [0, '10%']
      },
      title: { show: true, offsetCenter: [0, '45%'], fontSize: 13, color: '#86909c' },
      data: [{ value: pct, name: '内存使用率' }]
    }]
  }
})

// CPU 圆形仪表盘
const cpuGaugeOption = computed(() => {
  const raw = sysStatus.value.cpu?.used_percent
  const pct = raw != null && !isNaN(raw) ? Math.round(raw) : 0
  const color = pct > 85 ? '#f53f3f' : pct > 60 ? '#ff7d00' : '#3370ff'
  return {
    animation: true,
    animationDuration: 1000,
    series: [{
      type: 'gauge', startAngle: 200, endAngle: -20, min: 0, max: 100,
      pointer: { show: false },
      progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
      axisLine: { lineStyle: { width: 14, color: [[1, '#e5e6eb']] } },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false },
      detail: {
        valueAnimation: true, fontSize: 28, fontWeight: 700, color: '#1d2129',
        formatter: '{value}%',
        offsetCenter: [0, '10%']
      },
      title: { show: true, offsetCenter: [0, '45%'], fontSize: 13, color: '#86909c' },
      data: [{ value: pct, name: 'CPU 使用率' }]
    }]
  }
})

// 负载圆形仪表盘（1/5/15 分钟）
const loadGaugeOption = computed(() => {
  const cores = sysInfo.value.num_cpu || 1
  const loads = [
    { val: sysStatus.value.load?.['1m'] || 0, name: '1 分钟' },
    { val: sysStatus.value.load?.['5m'] || 0, name: '5 分钟' },
    { val: sysStatus.value.load?.['15m'] || 0, name: '15 分钟' },
  ]
  return loads.map(item => {
    const pct = Math.round((item.val / cores) * 100)
    const displayPct = Math.min(pct, 100)
    const color = pct > 80 ? '#f53f3f' : pct > 50 ? '#ff7d00' : '#722ed1'
    return {
      animation: true,
      animationDuration: 1000,
      series: [{
        type: 'gauge', startAngle: 200, endAngle: -20, min: 0, max: 100,
        pointer: { show: false },
        progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
        axisLine: { lineStyle: { width: 14, color: [[1, '#e5e6eb']] } },
        axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false },
        detail: {
          valueAnimation: true, fontSize: 24, fontWeight: 700, color: '#1d2129',
          formatter: '{value}%',
          offsetCenter: [0, '10%']
        },
        title: { show: true, offsetCenter: [0, '45%'], fontSize: 13, color: '#86909c' },
        data: [{ value: displayPct, name: item.name }]
      }]
    }
  })
})
</script>

<template>
  <div class="dashboard-pro fade-in-up">
    <!-- KPI 卡片行 -->
    <div class="kpi-row">
      <div v-for="(card, i) in kpiCards" :key="i" class="kpi-card">
        <div class="kpi-gradient" :style="{ background: card.bg }"></div>
        <div class="kpi-body">
          <div class="kpi-label">{{ card.label }}</div>
          <div class="kpi-value" :style="{ color: card.color }">{{ card.value }}</div>
          <div class="kpi-sub">{{ card.sub }}</div>
        </div>
      </div>
    </div>

    <!-- 趋势图 -->
    <div class="chart-single">
      <el-card class="chart-card chart-wide">
        <template #header>
          <div class="card-hdr">
            <span class="card-hdr-title"><el-icon><TrendCharts /></el-icon> CPU / 内存使用率趋势</span>
          </div>
        </template>
        <v-chart :option="cpuChartOption" style="height: 280px" autoresize />
      </el-card>
    </div>

    <!-- 三圆仪表盘行 -->
    <div class="gauge-row">
      <el-card class="chart-card gauge-card">
        <v-chart :option="cpuGaugeOption" style="height: 220px" autoresize />
        <div class="gauge-detail">
          <div><span class="gauge-detail-label">核心数:</span> {{ sysInfo.num_cpu || '-' }} 核</div>
        </div>
      </el-card>
      <el-card class="chart-card gauge-card">
        <v-chart :option="memGaugeOption" style="height: 220px" autoresize />
        <div class="gauge-detail">
          <div><span class="gauge-detail-label">已用:</span> {{ formatMB(sysStatus.memory?.used || 0) }}</div>
          <div><span class="gauge-detail-label">可用:</span> {{ formatMB(sysStatus.memory?.free || 0) }}</div>
          <div><span class="gauge-detail-label">总计:</span> {{ formatMB(sysStatus.memory?.total || 0) }}</div>
        </div>
      </el-card>
      <el-card class="chart-card gauge-card">
        <v-chart :option="diskGaugeOption" style="height: 220px" autoresize />
        <div class="gauge-detail">
          <div><span class="gauge-detail-label">已用:</span> {{ formatMB(sysStatus.disk?.used || 0) }}</div>
          <div><span class="gauge-detail-label">可用:</span> {{ formatMB(sysStatus.disk?.free || 0) }}</div>
          <div><span class="gauge-detail-label">总计:</span> {{ formatMB(sysStatus.disk?.total || 0) }}</div>
        </div>
      </el-card>
    </div>

    <!-- 底部信息行 -->
    <div class="info-row">
      <el-card class="info-card">
        <template #header>
          <div class="card-hdr">
            <span class="card-hdr-title"><el-icon><Monitor /></el-icon> 系统信息</span>
          </div>
        </template>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-key">操作系统</span>
            <span class="info-val">{{ sysInfo.os || '-' }} {{ sysInfo.arch || '' }}</span>
          </div>
          <div class="info-item">
            <span class="info-key">主机名</span>
            <span class="info-val">{{ sysInfo.hostname || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-key">内核版本</span>
            <span class="info-val">{{ sysInfo.kernel || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-key">CPU 核心</span>
            <span class="info-val">{{ sysInfo.num_cpu || '-' }} 核</span>
          </div>
          <div class="info-item">
            <span class="info-key">Go 版本</span>
            <span class="info-val">{{ sysInfo.go_version || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-key">运行时间</span>
            <span class="info-val">{{ sysInfo.uptime || '-' }}</span>
          </div>
        </div>
      </el-card>

      <el-card class="info-card">
        <template #header>
          <div class="card-hdr">
            <span class="card-hdr-title"><el-icon><DataAnalysis /></el-icon> 系统负载</span>
            <span class="card-hdr-sub">1 / 5 / 15 分钟</span>
          </div>
        </template>
        <div class="load-gauge-row">
          <div v-for="(opt, i) in loadGaugeOption" :key="i" class="load-gauge-item">
            <v-chart :option="opt" style="height: 160px" autoresize />
          </div>
        </div>
        <div class="load-bar-row">
          <div class="load-bar-label">CPU 核心数</div>
          <div class="load-bar-val">{{ sysInfo.num_cpu || 0 }} 核</div>
        </div>
        <div class="load-bar-row">
          <div class="load-bar-label">面板版本</div>
          <div class="load-bar-val">{{ dashboard.panel_version || '-' }}</div>
        </div>
        <div class="load-bar-row">
          <div class="load-bar-label">服务器时间</div>
          <div class="load-bar-val">{{ dashboard.server_time || '-' }}</div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.dashboard-pro {
  min-height: calc(100vh - 108px);
}

/* ===== KPI 卡片 ===== */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.kpi-card {
  background: #fff;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  position: relative;
  transition: transform 0.2s, box-shadow 0.2s;
  cursor: default;
}
.kpi-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.08);
}
.kpi-gradient {
  height: 4px;
}
.kpi-body {
  padding: 20px 18px 18px;
}
.kpi-label {
  font-size: 13px;
  color: #86909c;
  margin-bottom: 8px;
  font-weight: 500;
}
.kpi-value {
  font-size: 30px;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 6px;
  font-variant-numeric: tabular-nums;
}
.kpi-sub {
  font-size: 12px;
  color: #86909c;
}

/* ===== 趋势图 ===== */
.chart-single {
  margin-bottom: 20px;
}
.chart-card {
  border-radius: 14px !important;
  border: none !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04) !important;
}
.card-hdr {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-hdr-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
  display: flex;
  align-items: center;
  gap: 6px;
}
.card-hdr-title .el-icon {
  color: #3370ff;
}
.card-hdr-sub {
  font-size: 12px;
  color: #86909c;
}

/* ===== 三圆仪表盘行 ===== */
.gauge-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.gauge-card {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding-bottom: 12px;
}
.gauge-card :deep(.el-card__body) {
  width: 100%;
  flex: 1;
}
.gauge-card :deep(.echarts) {
  width: 100% !important;
}
.gauge-detail {
  display: flex;
  gap: 20px;
  font-size: 12px;
  color: #64748b;
  margin-top: -8px;
}
.gauge-detail-label {
  color: #94a3b8;
}

/* ===== 底部信息行 ===== */
.info-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
}
.info-item {
  display: flex;
  justify-content: space-between;
  padding: 13px 16px;
  border-bottom: 1px solid #f2f3f5;
}
.info-item:nth-child(odd) {
  border-right: 1px solid #f2f3f5;
}
.info-item:nth-last-child(-n+2) {
  border-bottom: none;
}
.info-key {
  font-size: 13px;
  color: #86909c;
}
.info-val {
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
}

/* 负载 */
.load-gauge-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  padding: 8px 0;
}
.load-gauge-item :deep(.echarts) {
  width: 100% !important;
}
.load-bar-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid #f2f3f5;
}
.load-bar-label {
  font-size: 13px;
  color: #86909c;
}
.load-bar-val {
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
}
</style>
