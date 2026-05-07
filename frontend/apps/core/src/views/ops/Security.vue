<script setup lang="ts">
defineOptions({
  name: 'OpsSecurity',
})

import { ref, onMounted, reactive } from 'vue'
import { getSecurityRules, createSecurityRule, updateSecurityRule, deleteSecurityRule, toggleSecurityRule } from '@/api/ops/security'
import {
  getSSHAccounts, createSSHAccount, updateSSHAccount, deleteSSHAccount,
  testSSHConnection, getSSHAccountFull, changeSSHCredential,
  changeRemotePassword, changeSSHPort, restartSSHD, installSSHKey,
  executeSSHCommand, getSSHdConfig, saveSSHdConfig, generateSSHKeyPair
} from '@/api/ops/ssh'
import { getFirewallStatus, addFirewallRule, deleteFirewallRule, getFirewallPorts, restartFirewall } from '@/api/ops/firewall'
import { ElMessage, ElMessageBox } from 'element-plus'

// ======== 安全规则 ========
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
  try {
    const res = await getSecurityRules({ page: page.value, page_size: 20 })
    list.value = res.data.list
    total.value = res.data.total
  } finally { loading.value = false }
}
function openDialog(row?: any) {
  editId.value = row?.id || null
  Object.assign(form, { name: row?.name || '', type: row?.type || 'ip_blacklist', content: row?.content || '', priority: row?.priority || 0, remark: row?.remark || '' })
  dialogVisible.value = true
}
async function handleSave() {
  if (editId.value) { await updateSecurityRule(editId.value, form); ElMessage.success('更新成功') }
  else { await createSecurityRule(form); ElMessage.success('创建成功') }
  dialogVisible.value = false; fetchData()
}
async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定删除该规则?', '提示')
  await deleteSecurityRule(row.id); ElMessage.success('删除成功'); fetchData()
}
async function handleToggle(row: any) { await toggleSecurityRule(row.id); fetchData() }

// ======== SSH 管理 ========
const sshList = ref([])
const sshTotal = ref(0)
const sshLoading = ref(false)
const sshPage = ref(1)
const sshDialogVisible = ref(false)
const sshEditId = ref<number | null>(null)
const sshForm = reactive({
  name: '', host: '', port: 22, username: '',
  password: '', auth_method: 'password', private_key: '', description: ''
})

// 凭证修改
const credDialogVisible = ref(false)
const credForm = reactive({ auth_method: 'password', password: '', private_key: '' })
const credTargetId = ref<number | null>(null)

// 远程密码修改
const pwdDialogVisible = ref(false)
const pwdForm = reactive({ new_password: '', confirm_password: '' })
const pwdTargetId = ref<number | null>(null)
const pwdTargetName = ref('')

// 端口修改
const portDialogVisible = ref(false)
const portForm = reactive({ new_port: 22 })
const portTargetId = ref<number | null>(null)
const portTargetName = ref('')

// 命令执行
const cmdDialogVisible = ref(false)
const cmdForm = reactive({ command: '' })
const cmdResult = ref('')
const cmdTargetId = ref<number | null>(null)
const cmdTargetName = ref('')
const cmdLoading = ref(false)

// 密钥生成
const keyDialogVisible = ref(false)
const keyResult = reactive({ private_key: '', public_key: '' })

// sshd_config 编辑
const configDialogVisible = ref(false)
const configContent = ref('')
const configTargetId = ref<number | null>(null)
const configTargetName = ref('')
const configLoading = ref(false)

// 测试结果
const testResultDialog = ref(false)
const testResultData = ref<any>({})

async function fetchSSHData() {
  sshLoading.value = true
  try {
    const res = await getSSHAccounts({ page: sshPage.value, page_size: 20 })
    sshList.value = res.data.list
    sshTotal.value = res.data.total
  } finally { sshLoading.value = false }
}

function openSSHDialog(row?: any) {
  sshEditId.value = row?.id || null
  if (row) {
    getSSHAccountFull(row.id).then(res => {
      const d = res.data
      Object.assign(sshForm, {
        name: d.name || '', host: d.host || '', port: d.port || 22,
        username: d.username || '', password: d.password || '',
        auth_method: d.auth_method || 'password', private_key: d.private_key || '',
        description: d.description || ''
      })
    })
  } else {
    Object.assign(sshForm, {
      name: '', host: '', port: 22, username: '',
      password: '', auth_method: 'password', private_key: '', description: ''
    })
  }
  sshDialogVisible.value = true
}

async function handleSSHSave() {
  if (sshEditId.value) {
    await updateSSHAccount(sshEditId.value, sshForm)
    ElMessage.success('更新成功')
  } else {
    await createSSHAccount(sshForm)
    ElMessage.success('创建成功')
  }
  sshDialogVisible.value = false
  fetchSSHData()
}

async function handleSSHDelete(row: any) {
  await ElMessageBox.confirm('确定删除该 SSH 账号?', '提示')
  await deleteSSHAccount(row.id)
  ElMessage.success('删除成功')
  fetchSSHData()
}

async function handleSSHTest(row: any) {
  try {
    const res = await testSSHConnection(row.id)
    testResultData.value = res.data
    testResultDialog.value = true
  } catch { ElMessage.error('测试失败') }
}

// 凭证修改
function openCredDialog(row: any) {
  credTargetId.value = row.id
  credForm.auth_method = row.auth_method || 'password'
  credForm.password = ''
  credForm.private_key = ''
  credDialogVisible.value = true
}
async function handleCredSave() {
  if (!credTargetId.value) return
  const data: any = { auth_method: credForm.auth_method }
  if (credForm.auth_method === 'password') data.password = credForm.password
  if (credForm.auth_method === 'key') data.private_key = credForm.private_key
  await changeSSHCredential(credTargetId.value, data)
  ElMessage.success('凭证已更新')
  credDialogVisible.value = false
  fetchSSHData()
}

// 远程密码修改
function openPwdDialog(row: any) {
  pwdTargetId.value = row.id
  pwdTargetName.value = row.name
  pwdForm.new_password = ''
  pwdForm.confirm_password = ''
  pwdDialogVisible.value = true
}
async function handlePwdSave() {
  if (pwdForm.new_password !== pwdForm.confirm_password) {
    ElMessage.error('两次密码不一致'); return
  }
  if (pwdForm.new_password.length < 6) {
    ElMessage.error('密码至少 6 位'); return
  }
  await changeRemotePassword(pwdTargetId.value!, { new_password: pwdForm.new_password })
  ElMessage.success('远程密码已修改')
  pwdDialogVisible.value = false
  fetchSSHData()
}

// 端口修改
function openPortDialog(row: any) {
  portTargetId.value = row.id
  portTargetName.value = row.name
  portForm.new_port = row.port
  portDialogVisible.value = true
}
async function handlePortSave() {
  await ElMessageBox.confirm(`确定将远程 SSH 端口改为 ${portForm.new_port}？修改后需用新端口连接！`, '警告', { type: 'warning' })
  await changeSSHPort(portTargetId.value!, { new_port: portForm.new_port })
  ElMessage.success('远程端口已修改')
  portDialogVisible.value = false
  fetchSSHData()
}

// 重启 SSH
async function handleRestart(row: any) {
  await ElMessageBox.confirm('确定重启远程 SSH 服务?', '提示', { type: 'warning' })
  await restartSSHD(row.id)
  ElMessage.success('SSH 服务已重启')
}

// 一键安装密钥
async function handleInstallKey(row: any) {
  await ElMessageBox.confirm('将生成新密钥对并部署到远程服务器，之后切换为密钥认证。继续?', '提示')
  const res = await installSSHKey(row.id)
  ElMessage.success(res.data.msg)
  keyResult.private_key = ''
  keyResult.public_key = res.data.public_key || ''
  keyDialogVisible.value = true
  fetchSSHData()
}

// 命令执行
function openCmdDialog(row: any) {
  cmdTargetId.value = row.id
  cmdTargetName.value = row.name
  cmdForm.command = ''
  cmdResult.value = ''
  cmdDialogVisible.value = true
}
async function handleCmdRun() {
  if (!cmdForm.command.trim()) return
  cmdLoading.value = true
  try {
    const res = await executeSSHCommand(cmdTargetId.value!, { command: cmdForm.command })
    const d = res.data
    cmdResult.value = (d.stdout || '') + (d.stderr ? '\n[STDERR]\n' + d.stderr : '') + (d.exit_ok ? '' : '\n[退出码: 非零]')
  } catch { cmdResult.value = '执行失败' }
  cmdLoading.value = false
}

// 密钥生成
async function handleGenKey() {
  const res = await generateSSHKeyPair()
  keyResult.private_key = res.data.private_key
  keyResult.public_key = res.data.public_key
  keyDialogVisible.value = true
}

// sshd_config
async function openConfigDialog(row: any) {
  configTargetId.value = row.id
  configTargetName.value = row.name
  configLoading.value = true
  configDialogVisible.value = true
  try {
    const res = await getSSHdConfig(row.id)
    configContent.value = res.data.content || ''
  } catch { configContent.value = '读取失败' }
  configLoading.value = false
}
async function handleConfigSave() {
  await saveSSHdConfig(configTargetId.value!, { content: configContent.value })
  ElMessage.success('配置已保存')
  configDialogVisible.value = false
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}

// ======== 防火墙管理 ========
const fwStatus = ref<any>({ enabled: false, backend: 'none', rules: [], chains: {} })
const fwPorts = ref<any[]>([])
const fwLoading = ref(false)
const fwDialogVisible = ref(false)
const fwForm = reactive({ protocol: 'tcp', dst_port: '', src_ip: '', target: 'ACCEPT', comment: '' })

async function fetchFirewallData() {
  fwLoading.value = true
  try {
    const [s, p] = await Promise.all([getFirewallStatus(), getFirewallPorts()])
    fwStatus.value = s.data
    fwPorts.value = p.data || []
  } finally { fwLoading.value = false }
}
function openFwDialog() {
  Object.assign(fwForm, { protocol: 'tcp', dst_port: '', src_ip: '', target: 'ACCEPT', comment: '' })
  fwDialogVisible.value = true
}
async function handleFwSave() {
  if (!fwForm.dst_port && !fwForm.src_ip) {
    ElMessage.error('端口或IP至少填写一项'); return
  }
  await addFirewallRule(fwForm)
  ElMessage.success('规则已添加')
  fwDialogVisible.value = false
  fetchFirewallData()
}
async function handleFwDelete(rule: any) {
  await ElMessageBox.confirm('确定删除该防火墙规则?', '提示')
  await deleteFirewallRule(rule.id)
  ElMessage.success('规则已删除')
  fetchFirewallData()
}
async function handleFwRestart() {
  await ElMessageBox.confirm('确定重启防火墙?', '提示', { type: 'warning' })
  await restartFirewall()
  ElMessage.success('防火墙已重启')
  fetchFirewallData()
}
function fwTargetLabel(target: string) {
  if (target === 'ACCEPT' || target === 'allow') return '允许'
  if (target === 'DROP' || target === 'deny') return '拒绝'
  if (target === 'REJECT') return '拒绝(返回)'
  return target
}
function fwTargetType(target: string) {
  if (target === 'ACCEPT' || target === 'allow') return 'success'
  if (target === 'DROP' || target === 'deny' || target === 'REJECT') return 'danger'
  return 'info'
}

onMounted(() => { fetchData(); fetchSSHData(); fetchFirewallData() })
</script>

<template>
  <div>
    <el-tabs type="border-card">
      <!-- 安全规则 Tab -->
      <el-tab-pane label="安全规则">
        <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
          <span style="font-size: 18px; font-weight: bold">安全管理</span>
          <el-button type="primary" @click="openDialog()">添加规则</el-button>
        </div>
        <el-table :data="list" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="type" label="类型" width="140">
            <template #default="{ row }"><el-tag>{{ typeOptions.find(t => t.value === row.type)?.label || row.type }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="content" label="内容" show-overflow-tooltip />
          <el-table-column prop="priority" label="优先级" width="80" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }"><el-tag :type="row.status === 'enabled' ? 'success' : 'info'">{{ row.status }}</el-tag></template>
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
            <el-form-item label="类型"><el-select v-model="form.type"><el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" /></el-select></el-form-item>
            <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="4" placeholder="每行一条规则" /></el-form-item>
            <el-form-item label="优先级"><el-input-number v-model="form.priority" :min="0" :max="100" /></el-form-item>
            <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
          </el-form>
          <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSave">确定</el-button></template>
        </el-dialog>
      </el-tab-pane>

      <!-- SSH 管理 Tab -->
      <el-tab-pane label="SSH 管理">
        <div style="margin-bottom: 16px; display: flex; justify-content: space-between">
          <span style="font-size: 18px; font-weight: bold">SSH 账号管理</span>
          <div>
            <el-button @click="handleGenKey" style="margin-right: 8px">🔑 生成密钥对</el-button>
            <el-button type="primary" @click="openSSHDialog()">添加 SSH 账号</el-button>
          </div>
        </div>
        <el-table :data="sshList" v-loading="sshLoading" stripe>
          <el-table-column prop="id" label="ID" width="50" />
          <el-table-column prop="name" label="名称" min-width="100" />
          <el-table-column label="连接信息" min-width="160">
            <template #default="{ row }">{{ row.username }}@{{ row.host }}:{{ row.port }}</template>
          </el-table-column>
          <el-table-column prop="auth_method" label="认证" width="80">
            <template #default="{ row }">
              <el-tag :type="row.auth_method === 'key' ? 'success' : 'primary'" size="small">
                {{ row.auth_method === 'key' ? '密钥' : '密码' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="70">
            <template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="description" label="备注" show-overflow-tooltip />
          <el-table-column label="操作" width="420">
            <template #default="{ row }">
              <el-button-group size="small">
                <el-button type="success" plain @click="handleSSHTest(row)">测试</el-button>
                <el-button @click="openSSHDialog(row)">编辑</el-button>
                <el-button @click="openCredDialog(row)">凭证</el-button>
                <el-button @click="openCmdDialog(row)">命令</el-button>
              </el-button-group>
              <el-dropdown trigger="click" style="margin-left: 4px">
                <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="handleInstallKey(row)">🔑 一键安装密钥</el-dropdown-item>
                    <el-dropdown-item @click="openPwdDialog(row)">🔒 修改远程密码</el-dropdown-item>
                    <el-dropdown-item @click="openPortDialog(row)">🔌 修改 SSH 端口</el-dropdown-item>
                    <el-dropdown-item @click="openConfigDialog(row)">📝 编辑 sshd_config</el-dropdown-item>
                    <el-dropdown-item @click="handleRestart(row)">🔄 重启 SSH 服务</el-dropdown-item>
                    <el-dropdown-item divided @click="handleSSHDelete(row)" style="color: #f56c6c">🗑️ 删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination style="margin-top: 16px" v-model:current-page="sshPage" :total="sshTotal" :page-size="20" layout="prev, pager, next" @current-change="fetchSSHData" />

        <!-- 添加/编辑 SSH 账号 -->
        <el-dialog v-model="sshDialogVisible" :title="sshEditId ? '编辑 SSH 账号' : '添加 SSH 账号'" width="600px">
          <el-form :model="sshForm" label-width="100px">
            <el-form-item label="名称"><el-input v-model="sshForm.name" placeholder="如：生产服务器" /></el-form-item>
            <el-form-item label="主机"><el-input v-model="sshForm.host" placeholder="IP 或域名" /></el-form-item>
            <el-form-item label="端口"><el-input-number v-model="sshForm.port" :min="1" :max="65535" /></el-form-item>
            <el-form-item label="用户名"><el-input v-model="sshForm.username" /></el-form-item>
            <el-form-item label="认证方式">
              <el-radio-group v-model="sshForm.auth_method">
                <el-radio value="password">密码</el-radio>
                <el-radio value="key">密钥</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="sshForm.auth_method === 'password'" label="密码">
              <el-input v-model="sshForm.password" type="password" show-password />
            </el-form-item>
            <el-form-item v-if="sshForm.auth_method === 'key'" label="私钥">
              <el-input v-model="sshForm.private_key" type="textarea" :rows="5" placeholder="粘贴 PEM 格式私钥" />
            </el-form-item>
            <el-form-item label="备注"><el-input v-model="sshForm.description" /></el-form-item>
          </el-form>
          <template #footer><el-button @click="sshDialogVisible = false">取消</el-button><el-button type="primary" @click="handleSSHSave">确定</el-button></template>
        </el-dialog>

        <!-- 修改凭证 -->
        <el-dialog v-model="credDialogVisible" title="修改 SSH 凭证" width="500px">
          <el-form :model="credForm" label-width="100px">
            <el-form-item label="认证方式">
              <el-radio-group v-model="credForm.auth_method">
                <el-radio value="password">密码</el-radio>
                <el-radio value="key">密钥</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="credForm.auth_method === 'password'" label="新密码">
              <el-input v-model="credForm.password" type="password" show-password placeholder="输入新密码" />
            </el-form-item>
            <el-form-item v-if="credForm.auth_method === 'key'" label="私钥">
              <el-input v-model="credForm.private_key" type="textarea" :rows="5" placeholder="粘贴 PEM 私钥" />
            </el-form-item>
          </el-form>
          <template #footer><el-button @click="credDialogVisible = false">取消</el-button><el-button type="primary" @click="handleCredSave">保存</el-button></template>
        </el-dialog>

        <!-- 修改远程密码 -->
        <el-dialog v-model="pwdDialogVisible" :title="`修改远程密码 - ${pwdTargetName}`" width="450px">
          <el-form :model="pwdForm" label-width="100px">
            <el-form-item label="新密码"><el-input v-model="pwdForm.new_password" type="password" show-password /></el-form-item>
            <el-form-item label="确认密码"><el-input v-model="pwdForm.confirm_password" type="password" show-password /></el-form-item>
          </el-form>
          <template #footer><el-button @click="pwdDialogVisible = false">取消</el-button><el-button type="primary" @click="handlePwdSave">修改</el-button></template>
        </el-dialog>

        <!-- 修改 SSH 端口 -->
        <el-dialog v-model="portDialogVisible" :title="`修改 SSH 端口 - ${portTargetName}`" width="400px">
          <el-form :model="portForm" label-width="100px">
            <el-form-item label="新端口"><el-input-number v-model="portForm.new_port" :min="1" :max="65535" /></el-form-item>
          </el-form>
          <el-alert title="将远程修改 /etc/ssh/sshd_config 并重启 sshd 服务" type="warning" :closable="false" />
          <template #footer><el-button @click="portDialogVisible = false">取消</el-button><el-button type="primary" @click="handlePortSave">修改</el-button></template>
        </el-dialog>

        <!-- 执行命令 -->
        <el-dialog v-model="cmdDialogVisible" :title="`执行命令 - ${cmdTargetName}`" width="700px">
          <el-input v-model="cmdForm.command" placeholder="输入命令，如: ls -la /etc" @keyup.enter="handleCmdRun">
            <template #append><el-button @click="handleCmdRun" :loading="cmdLoading">执行</el-button></template>
          </el-input>
          <pre v-if="cmdResult" style="background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; margin-top: 12px; max-height: 300px; overflow: auto; font-size: 13px; white-space: pre-wrap">{{ cmdResult }}</pre>
        </el-dialog>

        <!-- 密钥生成结果 -->
        <el-dialog v-model="keyDialogVisible" title="SSH 密钥对" width="700px">
          <el-form label-width="80px">
            <el-form-item v-if="keyResult.public_key" label="公钥">
              <el-input :model-value="keyResult.public_key" type="textarea" :rows="3" readonly />
              <el-button size="small" @click="copyText(keyResult.public_key)" style="margin-top: 4px">复制公钥</el-button>
            </el-form-item>
            <el-form-item v-if="keyResult.private_key" label="私钥">
              <el-input :model-value="keyResult.private_key" type="textarea" :rows="8" readonly />
              <el-button size="small" type="warning" @click="copyText(keyResult.private_key)" style="margin-top: 4px">复制私钥</el-button>
            </el-form-item>
          </el-form>
          <el-alert v-if="keyResult.private_key" title="请妥善保管私钥，关闭后将无法再次查看" type="warning" :closable="false" />
        </el-dialog>

        <!-- 测试结果 -->
        <el-dialog v-model="testResultDialog" title="连接测试结果" width="500px">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="主机">{{ testResultData.host }}:{{ testResultData.port }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="testResultData.status === 'connected' ? 'success' : 'danger'">
                {{ testResultData.status === 'connected' ? '✅ 连接成功' : '❌ 连接失败' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="详情">{{ testResultData.msg }}</el-descriptions-item>
            <el-descriptions-item v-if="testResultData.remote_info" label="远程信息">
              <pre style="margin: 0; white-space: pre-wrap; font-size: 12px">{{ testResultData.remote_info }}</pre>
            </el-descriptions-item>
          </el-descriptions>
        </el-dialog>

        <!-- sshd_config 编辑 -->
        <el-dialog v-model="configDialogVisible" :title="`编辑 sshd_config - ${configTargetName}`" width="800px">
          <el-alert title="修改 sshd_config 后需要重启 SSH 服务才能生效，系统会自动备份原配置" type="warning" :closable="false" style="margin-bottom: 12px" />
          <el-input v-model="configContent" type="textarea" :rows="20" :loading="configLoading" style="font-family: monospace" />
          <template #footer>
            <el-button @click="configDialogVisible = false">取消</el-button>
            <el-button type="primary" @click="handleConfigSave">保存配置</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- 防火墙管理 Tab -->
      <el-tab-pane label="防火墙">
        <!-- 状态栏 -->
        <div style="margin-bottom: 16px; display: flex; justify-content: space-between; align-items: center">
          <div style="display: flex; align-items: center; gap: 16px">
            <span style="font-size: 18px; font-weight: bold">防火墙管理</span>
            <el-tag :type="fwStatus.enabled ? 'success' : 'info'" size="large">
              {{ fwStatus.enabled ? '已启用' : '未启用' }}
            </el-tag>
            <el-tag type="info">{{ fwStatus.backend?.toUpperCase() || 'UNKNOWN' }}</el-tag>
            <el-tag v-if="fwStatus.fail2ban_bans > 0" type="danger" size="large">
              🛡️ Fail2Ban 封禁 {{ fwStatus.fail2ban_bans }} 个 IP
            </el-tag>
          </div>
          <div>
            <el-button @click="handleFwRestart" type="warning" plain style="margin-right: 8px">🔄 重启防火墙</el-button>
            <el-button type="primary" @click="openFwDialog">➕ 添加规则</el-button>
          </div>
        </div>

        <!-- 防火墙规则 -->
        <el-card style="margin-bottom: 16px" shadow="never">
          <template #header><span style="font-weight: 600">🛡️ 防火墙规则</span></template>
          <el-table :data="fwStatus.rules" v-loading="fwLoading" stripe empty-text="暂无规则">
            <el-table-column prop="id" label="#" width="60" />
            <el-table-column prop="chain" label="链" width="100">
              <template #default="{ row }">
                <el-tag size="small" :type="row.chain === 'INPUT' ? 'primary' : row.chain === 'OUTPUT' ? 'success' : 'warning'">
                  {{ row.chain }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="protocol" label="协议" width="80">
              <template #default="{ row }">{{ row.protocol?.toUpperCase() || 'ALL' }}</template>
            </el-table-column>
            <el-table-column prop="src_ip" label="来源 IP" min-width="130">
              <template #default="{ row }">{{ row.src_ip || '任意' }}</template>
            </el-table-column>
            <el-table-column prop="dst_port" label="目标端口" width="100">
              <template #default="{ row }">{{ row.dst_port || '任意' }}</template>
            </el-table-column>
            <el-table-column prop="target" label="动作" width="120">
              <template #default="{ row }">
                <el-tag :type="fwTargetType(row.target)" size="small">{{ fwTargetLabel(row.target) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="handleFwDelete(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 监听端口 -->
        <el-card shadow="never">
          <template #header><span style="font-weight: 600">🔌 已监听端口</span></template>
          <el-table :data="fwPorts" stripe empty-text="暂无监听端口">
            <el-table-column prop="proto" label="协议" width="80">
              <template #default="{ row }">
                <el-tag size="small" :type="row.proto?.includes('tcp') ? 'primary' : 'warning'">
                  {{ row.proto?.toUpperCase() }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="local" label="监听地址" min-width="180" />
            <el-table-column prop="process" label="进程" min-width="150">
              <template #default="{ row }">
                <span style="font-family: monospace; font-size: 12px">{{ row.process || '-' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 添加规则弹窗 -->
        <el-dialog v-model="fwDialogVisible" title="添加防火墙规则" width="500px">
          <el-form :model="fwForm" label-width="100px">
            <el-form-item label="协议">
              <el-select v-model="fwForm.protocol">
                <el-option label="TCP" value="tcp" />
                <el-option label="UDP" value="udp" />
                <el-option label="全部" value="all" />
              </el-select>
            </el-form-item>
            <el-form-item label="目标端口">
              <el-input v-model="fwForm.dst_port" placeholder="如: 80, 443, 8000-9000" />
            </el-form-item>
            <el-form-item label="来源 IP">
              <el-input v-model="fwForm.src_ip" placeholder="如: 192.168.1.0/24（留空=任意）" />
            </el-form-item>
            <el-form-item label="动作">
              <el-select v-model="fwForm.target">
                <el-option label="允许 (ACCEPT)" value="ACCEPT" />
                <el-option label="拒绝 (DROP)" value="DROP" />
                <el-option label="拒绝并返回 (REJECT)" value="REJECT" />
              </el-select>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="fwForm.comment" placeholder="规则说明" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="fwDialogVisible = false">取消</el-button>
            <el-button type="primary" @click="handleFwSave">添加</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>
<style scoped>
.ops-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
