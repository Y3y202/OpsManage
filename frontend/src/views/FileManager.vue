<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listFiles, readFile, saveFile, deleteFile, mkdir, renameFile } from '@/api/file'
import { ElMessage, ElMessageBox } from 'element-plus'

const currentPath = ref('/')
const files = ref<any[]>([])
const loading = ref(false)
const editDialogVisible = ref(false)
const editPath = ref('')
const editContent = ref('')
const mkdirDialogVisible = ref(false)
const newDirPath = ref('')
const renameDialogVisible = ref(false)
const renameOldPath = ref('')
const renameNewPath = ref('')

async function fetchData() {
  loading.value = true
  try {
    const res = await listFiles(currentPath.value)
    files.value = res.data.files || []
  } catch {
    files.value = []
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string) {
  currentPath.value = path
  fetchData()
}

function goUp() {
  const parts = currentPath.value.replace(/\/$/, '').split('/')
  parts.pop()
  navigateTo(parts.join('/') || '/')
}

async function handleOpen(row: any) {
  if (row.is_dir) {
    navigateTo(row.path)
  } else {
    try {
      const res = await readFile(row.path)
      editPath.value = row.path
      editContent.value = res.data.content
      editDialogVisible.value = true
    } catch {
      ElMessage.error('无法打开文件')
    }
  }
}

async function handleSave() {
  await saveFile(editPath.value, editContent.value)
  ElMessage.success('保存成功')
  editDialogVisible.value = false
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除 ${row.name}?`, '提示')
  await deleteFile(row.path)
  ElMessage.success('已删除')
  fetchData()
}

function openMkdirDialog() {
  newDirPath.value = currentPath.value + '/new-folder'
  mkdirDialogVisible.value = true
}

async function handleMkdir() {
  await mkdir(newDirPath.value)
  ElMessage.success('目录已创建')
  mkdirDialogVisible.value = false
  fetchData()
}

function openRenameDialog(row: any) {
  renameOldPath.value = row.path
  renameNewPath.value = row.path
  renameDialogVisible.value = true
}

async function handleRename() {
  await renameFile(renameOldPath.value, renameNewPath.value)
  ElMessage.success('重命名成功')
  renameDialogVisible.value = false
  fetchData()
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

onMounted(fetchData)
</script>

<template>
  <el-card>
    <div style="margin-bottom: 16px; display: flex; align-items: center; gap: 8px">
      <span style="font-size: 18px; font-weight: bold">文件管理</span>
      <div style="flex: 1" />
      <el-button @click="goUp">上级目录</el-button>
      <el-button type="primary" @click="openMkdirDialog">新建目录</el-button>
    </div>
    <el-breadcrumb separator="/" style="margin-bottom: 16px">
      <el-breadcrumb-item>
        <a @click="navigateTo('/')">/</a>
      </el-breadcrumb-item>
      <template v-for="(seg, i) in currentPath.split('/').filter(Boolean)" :key="i">
        <el-breadcrumb-item>
          <a @click="navigateTo('/' + currentPath.split('/').filter(Boolean).slice(0, i + 1).join('/'))">{{ seg }}</a>
        </el-breadcrumb-item>
      </template>
    </el-breadcrumb>
    <el-table :data="files" v-loading="loading" stripe @row-dblclick="handleOpen">
      <el-table-column prop="name" label="名称">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 6px; cursor: pointer" @click="handleOpen(row)">
            <el-icon v-if="row.is_dir"><FolderOpened /></el-icon>
            <el-icon v-else><Document /></el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="size" label="大小" width="100">
        <template #default="{ row }">{{ row.is_dir ? '-' : formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mode" label="权限" width="100" />
      <el-table-column prop="mod_time" label="修改时间" width="180" />
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button size="small" @click.stop="openRenameDialog(row)">重命名</el-button>
          <el-button size="small" type="danger" @click.stop="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editDialogVisible" :title="'编辑: ' + editPath" width="800px">
      <el-input v-model="editContent" type="textarea" :rows="20" style="font-family: monospace" />
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="mkdirDialogVisible" title="新建目录" width="400px">
      <el-input v-model="newDirPath" placeholder="目录路径" />
      <template #footer>
        <el-button @click="mkdirDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleMkdir">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="renameDialogVisible" title="重命名" width="400px">
      <el-input v-model="renameNewPath" placeholder="新路径" />
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRename">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
