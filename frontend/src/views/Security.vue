<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { getSecurityRules, createSecurityRule, updateSecurityRule, deleteSecurityRule, toggleSecurityRule } from '@/api/security'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const form = reactive({ name: '', type: 'ip_blacklist', content: '', priority: 0, remark: '' })

const typeOptions = [
  { label: 'IP黑名单', value: 'ip_blacklist' },
  { label: 'IP白名单', value: 'ip_whitelist' },
  { label: 'URL黑名单', value: 'url_blacklist' },
  { label: 'User-Agent黑名单', value: 'ua_blacklist' },
  { label: '自定义规则', value: 'custom' }
]

async function fetchData() {
  loading.value = true
  const res = await getSecurityRules({ page: page.value, page_size: 20 })
  list.value = res.data.list
  total.value = res.data.total
  loading.value = false
}

function openDialog(row?: any) {
  editId.value = row?.id || null
  Object.assign(form, { name: row?.name || '', type: row?.type || 'ip_blacklist', content: row?.content || '', priority: row?.priority || 0, remark: row?.remark || '' })
  dialogVisible.value = true
}

async function handleSave() {
  if (editId.value) {
    await updateSecurityRule(editId.value, form)
    ElMessage.success('更新成功')
  } else {
    await createSecurityRule(form)
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchData()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该规则?', '提示')
  await deleteSecurityRule(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function handleToggle(row: any) {
  await toggleSecurityRule(row.id)
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
      <span style="font-size: 18px; font-weight: bold">安全管理</span>
      <el-button type="primary" @click="openDialog()">添加规则</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="type" label="类型" width="140">
        <template #default="{ row }">
          <el-tag>{{ typeOptions.find(t => t.value === row.type)?.label || row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="content" label="内容" show-overflow-tooltip />
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="handleToggle(row)">{{ row.status === 'enabled' ? '禁用' : '启用' }}</el-button>
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top: 16px" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="fetchData" />

    <el-dialog v-model="dialogVisible" :title="editId ? '编辑规则' : '添加规则'" width="600px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type">
            <el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="4" placeholder="每行一条规则" /></el-form-item>
        <el-form-item label="优先级"><el-input-number v-model="form.priority" :min="0" :max="100" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
