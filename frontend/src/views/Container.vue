<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { getContainers, createContainer, deleteContainer, startContainer, stopContainer, restartContainer, getContainerLogs } from '@/api/container'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const dialogVisible = ref(false)
const logDialogVisible = ref(false)
const logContent = ref('')
const form = reactive({ name: '', image: '', ports: '', volumes: '', env: '' })

async function fetchData() {
  loading.value = true
  const res = await getContainers({ page: page.value, page_size: 20 })
  list.value = res.data.list
  total.value = res.data.total
  loading.value = false
}

function openDialog() {
  Object.assign(form, { name: '', image: '', ports: '', volumes: '', env: '' })
  dialogVisible.value = true
}

async function handleCreate() {
  await createContainer(form)
  ElMessage.success('创建成功')
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该容器?', '提示')
  await deleteContainer(row.id)
  ElMessage.success('已删除')
  fetchData()
}

async function handleStart(row: any) {
  await startContainer(row.id)
  ElMessage.success('已启动')
  fetchData()
}

async function handleStop(row: any) {
  await stopContainer(row.id)
  ElMessage.success('已停止')
  fetchData()
}

async function handleRestart(row: any) {
  await restartContainer(row.id)
  ElMessage.success('已重启')
  fetchData()
}

async function handleLogs(row: any) {
  const res = await getContainerLogs(row.id)
  logContent.value = res.data.logs
  logDialogVisible.value = true
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">容器管理</span>
      <el-button type="primary" @click="openDialog()">创建容器</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="image" label="镜像" />
      <el-table-column prop="ports" label="端口映射" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="300">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="handleStart(row)">启动</el-button>
          <el-button size="small" type="warning" @click="handleStop(row)">停止</el-button>
          <el-button size="small" @click="handleRestart(row)">重启</el-button>
          <el-button size="small" @click="handleLogs(row)">日志</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top: 16px" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="fetchData" />

    <el-dialog v-model="dialogVisible" title="创建容器" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="镜像"><el-input v-model="form.image" placeholder="nginx:latest" /></el-form-item>
        <el-form-item label="端口"><el-input v-model="form.ports" placeholder="8080:80" /></el-form-item>
        <el-form-item label="卷"><el-input v-model="form.volumes" placeholder="/data:/data" /></el-form-item>
        <el-form-item label="环境变量"><el-input v-model="form.env" placeholder="KEY=VALUE" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="logDialogVisible" title="容器日志" width="700px">
      <pre style="background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 4px; max-height: 400px; overflow: auto; font-size: 13px">{{ logContent }}</pre>
    </el-dialog>
  </el-card>
</template>
