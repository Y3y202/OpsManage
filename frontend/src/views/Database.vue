<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { getDatabases, createDatabase, updateDatabase, deleteDatabase, startDatabase, stopDatabase } from '@/api/database'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const form = reactive({ name: '', type: 'mysql', host: '127.0.0.1', port: 3306, username: '', password: '', version: '' })

const typeOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'postgresql' },
  { label: 'Redis', value: 'redis' }
]

const defaultPorts: Record<string, number> = { mysql: 3306, postgresql: 5432, redis: 6379 }

function onTypeChange(val: string) {
  form.port = defaultPorts[val] || 3306
}

async function fetchData() {
  loading.value = true
  const res = await getDatabases({ page: page.value, page_size: 20 })
  list.value = res.data.list
  total.value = res.data.total
  loading.value = false
}

function openDialog(row?: any) {
  editId.value = row?.id || null
  Object.assign(form, { name: row?.name || '', type: row?.type || 'mysql', host: row?.host || '127.0.0.1', port: row?.port || 3306, username: row?.username || '', password: '', version: row?.version || '' })
  dialogVisible.value = true
}

async function handleSave() {
  if (editId.value) {
    await updateDatabase(editId.value, form)
    ElMessage.success('更新成功')
  } else {
    await createDatabase(form)
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该数据库?', '提示')
  await deleteDatabase(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function handleStart(row: any) {
  await startDatabase(row.id)
  ElMessage.success('已启动')
  fetchData()
}

async function handleStop(row: any) {
  await stopDatabase(row.id)
  ElMessage.success('已停止')
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">数据库管理</span>
      <el-button type="primary" @click="openDialog()">添加数据库</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag>{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="host" label="主机" width="140" />
      <el-table-column prop="port" label="端口" width="80" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'danger'">{{ row.status }}</el-tag>
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

    <el-dialog v-model="dialogVisible" :title="editId ? '编辑数据库' : '添加数据库'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" @change="onTypeChange">
            <el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="主机"><el-input v-model="form.host" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="form.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="版本"><el-input v-model="form.version" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
