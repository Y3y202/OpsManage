<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { getWebsites, createWebsite, updateWebsite, deleteWebsite, startWebsite, stopWebsite } from '@/api/website'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const form = reactive({ name: '', domain: '', path: '', port: 80, remark: '', waf_enabled: false })

async function fetchData() {
  loading.value = true
  const res = await getWebsites({ page: page.value, page_size: 20 })
  list.value = res.data.list
  total.value = res.data.total
  loading.value = false
}

function openDialog(row?: any) {
  editId.value = row?.id || null
  Object.assign(form, { name: row?.name || '', domain: row?.domain || '', path: row?.path || '', port: row?.port || 80, remark: row?.remark || '', waf_enabled: row?.waf_enabled || false })
  dialogVisible.value = true
}

async function handleSave() {
  if (editId.value) {
    await updateWebsite(editId.value, form)
    ElMessage.success('更新成功')
  } else {
    await createWebsite(form)
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该网站?', '提示')
  await deleteWebsite(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function handleStart(row: any) {
  await startWebsite(row.id)
  ElMessage.success('已启动')
  fetchData()
}

async function handleStop(row: any) {
  await stopWebsite(row.id)
  ElMessage.success('已停止')
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">网站管理</span>
      <el-button type="primary" @click="openDialog()">添加网站</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="domain" label="域名" />
      <el-table-column prop="port" label="端口" width="80" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'danger'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="ssl_enabled" label="SSL" width="80">
        <template #default="{ row }">
          <el-tag :type="row.ssl_enabled ? 'success' : 'info'">{{ row.ssl_enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" :type="row.status === 'running' ? 'warning' : 'success'" @click="row.status === 'running' ? handleStop(row) : handleStart(row)">
            {{ row.status === 'running' ? '停止' : '启动' }}
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top: 16px" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="fetchData" />

    <el-dialog v-model="dialogVisible" :title="editId ? '编辑网站' : '添加网站'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="域名"><el-input v-model="form.domain" /></el-form-item>
        <el-form-item label="路径"><el-input v-model="form.path" placeholder="/var/www/html" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="form.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        <el-form-item label="WAF"><el-switch v-model="form.waf_enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
