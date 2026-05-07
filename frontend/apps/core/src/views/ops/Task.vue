<script setup lang="ts">
defineOptions({
  name: 'OpsTask',
})

import { ref, onMounted, reactive } from 'vue'
import { getTasks, createTask, updateTask, deleteTask, runTask, toggleTask } from '@/api/ops/task'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const form = reactive({ name: '', command: '', cron_expr: '' })

async function fetchData() {
  loading.value = true
  try {
    const res = await getTasks({ page: page.value, page_size: 20 })
    list.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

function openDialog(row?: any) {
  editId.value = row?.id || null
  Object.assign(form, { name: row?.name || '', command: row?.command || '', cron_expr: row?.cron_expr || '' })
  dialogVisible.value = true
}

async function handleSave() {
  if (editId.value) {
    await updateTask(editId.value, form)
    ElMessage.success('更新成功')
  } else {
    await createTask(form)
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该任务?', '提示')
  await deleteTask(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function handleRun(row: any) {
  await runTask(row.id)
  ElMessage.success('任务已触发执行')
  setTimeout(fetchData, 2000)
}

async function handleToggle(row: any) {
  await toggleTask(row.id)
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">计划任务</span>
      <el-button type="primary" @click="openDialog()">添加任务</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="cron_expr" label="Cron表达式" width="140" />
      <el-table-column prop="command" label="命令" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_result" label="上次结果" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.last_result" :type="row.last_result === 'success' ? 'success' : 'danger'">{{ row.last_result }}</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="last_run" label="上次执行" width="180" />
      <el-table-column prop="next_run" label="下次执行" width="180">
        <template #default="{ row }">
          <span v-if="row.next_run && row.next_run !== '0001-01-01T00:00:00Z'">{{ row.next_run }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="handleRun(row)">执行</el-button>
          <el-button size="small" @click="handleToggle(row)">{{ row.status === 'enabled' ? '禁用' : '启用' }}</el-button>
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top: 16px" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="fetchData" />

    <el-dialog v-model="dialogVisible" :title="editId ? '编辑任务' : '添加任务'" width="600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="Cron表达式"><el-input v-model="form.cron_expr" placeholder="0 */5 * * * *" /></el-form-item>
        <el-form-item label="命令"><el-input v-model="form.command" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
<style scoped>
.ops-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
