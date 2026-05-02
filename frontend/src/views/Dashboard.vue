<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboard, getSystemInfo, getSystemStatus } from '@/api/dashboard'

const dashboard = ref<any>({})
const sysInfo = ref<any>({})
const sysStatus = ref<any>({})

onMounted(async () => {
  const [d, i, s] = await Promise.all([
    getDashboard(),
    getSystemInfo(),
    getSystemStatus()
  ])
  dashboard.value = d.data
  sysInfo.value = i.data
  sysStatus.value = s.data
})

function formatMB(mb: number) {
  if (mb > 1024) return (mb / 1024).toFixed(1) + ' GB'
  return mb + ' MB'
}
</script>

<template>
  <div>
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="8">
        <el-card>
          <div style="display: flex; align-items: center; gap: 12px">
            <el-icon size="40" color="#409eff"><Globe /></el-icon>
            <div>
              <div style="font-size: 28px; font-weight: bold">{{ dashboard.websites || 0 }}</div>
              <div style="color: #999">网站</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div style="display: flex; align-items: center; gap: 12px">
            <el-icon size="40" color="#67c23a"><Coin /></el-icon>
            <div>
              <div style="font-size: 28px; font-weight: bold">{{ dashboard.databases || 0 }}</div>
              <div style="color: #999">数据库</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div style="display: flex; align-items: center; gap: 12px">
            <el-icon size="40" color="#e6a23c"><Box /></el-icon>
            <div>
              <div style="font-size: 28px; font-weight: bold">{{ dashboard.containers || 0 }}</div>
              <div style="color: #999">容器</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card header="系统信息">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="操作系统">{{ sysInfo.os }}</el-descriptions-item>
            <el-descriptions-item label="架构">{{ sysInfo.arch }}</el-descriptions-item>
            <el-descriptions-item label="主机名">{{ sysInfo.hostname }}</el-descriptions-item>
            <el-descriptions-item label="内核">{{ sysInfo.kernel }}</el-descriptions-item>
            <el-descriptions-item label="CPU核心">{{ sysInfo.num_cpu }}</el-descriptions-item>
            <el-descriptions-item label="Go版本">{{ sysInfo.go_version }}</el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ sysInfo.uptime }}</el-descriptions-item>
            <el-descriptions-item label="面板版本">{{ dashboard.panel_version }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="资源使用">
          <div style="margin-bottom: 20px">
            <div style="margin-bottom: 8px">内存: {{ formatMB(sysStatus.memory?.used || 0) }} / {{ formatMB(sysStatus.memory?.total || 0) }}</div>
            <el-progress :percentage="Math.round(sysStatus.memory?.used_percent || 0)" :color="['#67c23a', '#e6a23c', '#f56c6c']" />
          </div>
          <div style="margin-bottom: 20px">
            <div style="margin-bottom: 8px">磁盘: {{ formatMB(sysStatus.disk?.used || 0) }} / {{ formatMB(sysStatus.disk?.total || 0) }}</div>
            <el-progress :percentage="Math.round(sysStatus.disk?.used_percent || 0)" :color="['#67c23a', '#e6a23c', '#f56c6c']" />
          </div>
          <div>
            <div style="margin-bottom: 8px">负载: {{ sysStatus.load?.['1m'] || 0 }} / {{ sysStatus.load?.['5m'] || 0 }} / {{ sysStatus.load?.['15m'] || 0 }}</div>
            <div style="color: #999; font-size: 12px">1分钟 / 5分钟 / 15分钟</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
