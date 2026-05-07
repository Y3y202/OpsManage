<script setup lang="ts">
defineOptions({
  name: 'OpsSetting',
})

import { ref, onMounted } from 'vue'
import { getSettings, updateSettings } from '@/api/ops/setting'
import { ElMessage, ElMessageBox } from 'element-plus'

const settings = ref<Record<string, string>>({})
const loading = ref(false)

async function fetchData() {
  loading.value = true
  try {
    const res = await getSettings()
    settings.value = res.data || {}
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  await updateSettings(settings.value)
  ElMessage.success('保存成功')
  fetchData()
}

async function addSetting() {
  try {
    const { value } = await ElMessageBox.prompt('请输入设置项名称', '添加设置', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /.+/,
      inputErrorMessage: '名称不能为空'
    })
    settings.value[value] = ''
  } catch {
    // cancelled
  }
}

function removeSetting(key: string) {
  delete settings.value[key]
  settings.value = { ...settings.value }
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">系统设置</span>
      <div>
        <el-button @click="addSetting">添加设置项</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </div>
    </div>
    <el-table :data="Object.entries(settings).map(([key, value]) => ({ key, value }))" v-loading="loading" stripe>
      <el-table-column prop="key" label="键" width="250" />
      <el-table-column label="值">
        <template #default="{ row }">
          <el-input v-model="settings[row.key]" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="removeSetting(row.key)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>
<style scoped>
.ops-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
