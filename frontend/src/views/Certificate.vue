<template>
  <div class="certificate-page">
    <!-- 顶部操作栏 -->
    <div class="page-header">
      <div class="header-left">
        <h2>证书管理</h2>
        <span class="subtitle">管理 SSL/TLS 证书，申请 Let's Encrypt 免费证书</span>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showApplyDialog = true">
          <el-icon><Plus /></el-icon>申请 Let's Encrypt 证书
        </el-button>
        <el-button @click="showUploadDialog = true">
          <el-icon><Upload /></el-icon>上传自定义证书
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-cards">
      <el-card shadow="never" class="stat-card">
        <div class="stat-icon valid"><el-icon><CircleCheck /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ validCount }}</div>
          <div class="stat-label">有效证书</div>
        </div>
      </el-card>
      <el-card shadow="never" class="stat-card">
        <div class="stat-icon warning"><el-icon><Warning /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ aboutToExpireCount }}</div>
          <div class="stat-label">即将过期</div>
        </div>
      </el-card>
      <el-card shadow="never" class="stat-card">
        <div class="stat-icon expired"><el-icon><CircleClose /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ expiredCount }}</div>
          <div class="stat-label">已过期</div>
        </div>
      </el-card>
      <el-card shadow="never" class="stat-card">
        <div class="stat-icon total"><el-icon><Files /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ certificates.length }}</div>
          <div class="stat-label">证书总数</div>
        </div>
      </el-card>
    </div>

    <!-- 证书列表 -->
    <el-card shadow="never" class="table-card">
      <el-table :data="certificates" v-loading="loading" stripe>
        <el-table-column prop="name" label="证书名称" min-width="150">
          <template #default="{ row }">
            <div class="cert-name">
              <el-icon :class="['cert-icon', row.type]"><Key /></el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="domain" label="域名" min-width="160" />
        <el-table-column prop="type" label="类型" width="130">
          <template #default="{ row }">
            <el-tag :type="row.type === 'custom' ? 'warning' : 'success'" size="small">
              {{ row.type === 'letsencrypt' ? "Let's Encrypt" : row.type === 'letsencrypt-dns' ? 'DNS 泛域名' : '自定义' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="issuer" label="颁发者" min-width="150">
          <template #default="{ row }">
            <span class="issuer">{{ row.issuer || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="有效期" width="200">
          <template #default="{ row }">
            <div class="validity">
              <div>{{ formatDate(row.not_before) }}</div>
              <div class="date-sep">至</div>
              <div :class="{ 'expired-text': row.status === 'expired', 'warning-text': row.status === 'about_to_expire' }">
                {{ formatDate(row.not_after) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button-group size="small">
              <el-button type="primary" link @click="viewCertDetail(row)">详情</el-button>
              <el-button
                v-if="row.type === 'letsencrypt' || row.type === 'letsencrypt-dns'"
                type="success"
                link
                @click="handleRenew(row)"
              >续签</el-button>
              <el-button
                v-if="row.type === 'letsencrypt' || row.type === 'letsencrypt-dns'"
                :type="row.auto_renew ? 'success' : 'info'"
                link
                @click="handleToggleAutoRenew(row)"
              >{{ row.auto_renew ? '✓ 自动续签' : '自动续签' }}</el-button>
              <el-button type="warning" link @click="showApplyToSite(row)">绑定站点</el-button>
              <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 申请 Let's Encrypt 证书对话框 -->
    <el-dialog v-model="showApplyDialog" title="申请 Let's Encrypt 证书" width="560px" destroy-on-close>
      <el-form :model="applyForm" label-width="100px" ref="applyFormRef" :rules="applyRules">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="applyForm.domain" placeholder="example.com 或 *.example.com（泛域名）" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="applyForm.email" placeholder="admin@example.com" />
        </el-form-item>
        <el-form-item label="验证方式">
          <el-radio-group v-model="applyForm.challenge" @change="onChallengeChange">
            <el-radio value="http">HTTP 验证（仅普通域名）</el-radio>
            <el-radio value="dns">DNS 验证（支持泛域名）</el-radio>
          </el-radio-group>
        </el-form-item>
        <!-- HTTP 验证选项 -->
        <template v-if="applyForm.challenge === 'http'">
          <el-form-item label="模式">
            <el-radio-group v-model="applyForm.standalone">
              <el-radio :value="false">Webroot（推荐，无需停止 Nginx）</el-radio>
              <el-radio :value="true">Standalone（需要 80 端口空闲）</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="!applyForm.standalone" label="Web 目录">
            <el-input v-model="applyForm.web_root" placeholder="/var/www/html" />
          </el-form-item>
        </template>
        <!-- DNS 验证选项 -->
        <template v-if="applyForm.challenge === 'dns'">
          <el-form-item label="DNS 提供商">
            <el-select v-model="applyForm.dns_provider" placeholder="选择域名所在的云服务商" style="width: 100%">
              <el-option label="阿里云（Alibaba Cloud）" value="aliyun" />
              <el-option label="腾讯云（Tencent Cloud）" value="tencent" />
              <el-option label="华为云（Huawei Cloud）" value="huawei" />
              <el-option label="Cloudflare" value="cloudflare" />
            </el-select>
          </el-form-item>
          <el-form-item :label="dnsKeyLabel">
            <el-input v-model="applyForm.dns_key" :placeholder="dnsKeyPlaceholder" show-password />
          </el-form-item>
          <el-form-item :label="dnsSecretLabel">
            <el-input v-model="applyForm.dns_secret" :placeholder="dnsSecretPlaceholder" show-password />
          </el-form-item>
          <el-form-item v-if="applyForm.dns_provider === 'huawei'" label="项目 ID">
            <el-input v-model="applyForm.hw_project_id" placeholder="华为云项目 ID" />
          </el-form-item>
          <el-alert type="info" :closable="false" style="margin-bottom: 12px;">
            <template #title>
              <div style="font-size: 12px; line-height: 1.6;">
                <b>泛域名格式：</b>输入 <code>*.example.com</code> 即可为主域及所有子域申请一张证书<br/>
                <b>密钥获取：</b>
                <span v-if="applyForm.dns_provider === 'aliyun'">阿里云控制台 → 右上角头像 → AccessKey 管理</span>
                <span v-else-if="applyForm.dns_provider === 'tencent'">腾讯云控制台 → 访问管理 → API 密钥管理</span>
                <span v-else-if="applyForm.dns_provider === 'huawei'">华为云控制台 → 我的凭证 → 访问密钥</span>
                <span v-else-if="applyForm.dns_provider === 'cloudflare'">Cloudflare Dashboard → My Profile → API Tokens</span>
                <span v-else>选择 DNS 提供商后显示获取方式</span>
              </div>
            </template>
          </el-alert>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="showApplyDialog = false">取消</el-button>
        <el-button type="primary" @click="handleApply" :loading="applying">
          申请证书
        </el-button>
      </template>
    </el-dialog>

    <!-- 上传自定义证书对话框 -->
    <el-dialog v-model="showUploadDialog" title="上传自定义证书" width="600px" destroy-on-close>
      <el-form :model="uploadForm" label-width="100px" ref="uploadFormRef" :rules="uploadRules">
        <el-form-item label="证书名称" prop="name">
          <el-input v-model="uploadForm.name" placeholder="my-site-cert" />
        </el-form-item>
        <el-form-item label="域名" prop="domain">
          <el-input v-model="uploadForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="证书内容" prop="cert">
          <el-input
            v-model="uploadForm.cert"
            type="textarea"
            :rows="6"
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
          />
        </el-form-item>
        <el-form-item label="私钥内容" prop="key">
          <el-input
            v-model="uploadForm.key"
            type="textarea"
            :rows="6"
            placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
          />
        </el-form-item>
        <el-form-item label="证书链（可选）">
          <el-input
            v-model="uploadForm.chain"
            type="textarea"
            :rows="4"
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showUploadDialog = false">取消</el-button>
        <el-button type="primary" @click="handleUpload" :loading="uploading">
          上传证书
        </el-button>
      </template>
    </el-dialog>

    <!-- 证书详情对话框 -->
    <el-dialog v-model="showDetailDialog" title="证书详情" width="700px" destroy-on-close>
      <template v-if="currentCert">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="证书名称">{{ currentCert.name }}</el-descriptions-item>
          <el-descriptions-item label="域名">{{ currentCert.domain }}</el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag :type="currentCert.type === 'custom' ? 'warning' : 'success'" size="small">
              {{ currentCert.type === 'letsencrypt' ? "Let's Encrypt" : currentCert.type === 'letsencrypt-dns' ? 'DNS 泛域名' : '自定义' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTagType(currentCert.status)" size="small">
              {{ statusText(currentCert.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="颁发者" :span="2">{{ currentCert.issuer || '-' }}</el-descriptions-item>
          <el-descriptions-item label="主题" :span="2">{{ currentCert.subject || '-' }}</el-descriptions-item>
          <el-descriptions-item label="SAN 域名" :span="2">{{ currentCert.sans || '-' }}</el-descriptions-item>
          <el-descriptions-item label="生效时间">{{ formatDate(currentCert.not_before) }}</el-descriptions-item>
          <el-descriptions-item label="过期时间">{{ formatDate(currentCert.not_after) }}</el-descriptions-item>
          <el-descriptions-item label="证书路径" :span="2">{{ currentCert.cert_path }}</el-descriptions-item>
          <el-descriptions-item label="私钥路径" :span="2">{{ currentCert.key_path }}</el-descriptions-item>
        </el-descriptions>

        <div style="margin-top: 16px;">
          <el-tabs v-model="contentTab">
            <el-tab-pane label="证书内容" name="cert">
              <el-input type="textarea" :rows="10" :model-value="certContent" readonly />
            </el-tab-pane>
            <el-tab-pane label="私钥内容" name="key">
              <el-input type="textarea" :rows="10" :model-value="keyContent" readonly />
            </el-tab-pane>
          </el-tabs>
        </div>
      </template>
    </el-dialog>

    <!-- 绑定站点对话框 -->
    <el-dialog v-model="showSiteDialog" title="应用证书到站点" width="450px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="目标站点">
          <el-select v-model="selectedSiteId" placeholder="选择站点" style="width: 100%">
            <el-option
              v-for="site in sites"
              :key="site.id"
              :label="`${site.name} (${site.domain})`"
              :value="site.id"
            >
              <span style="float: left">{{ site.name }}</span>
              <span style="float: right; color: var(--el-text-color-secondary); font-size: 12px;">
                {{ site.domain }} {{ site.ssl ? '(已有SSL)' : '' }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSiteDialog = false">取消</el-button>
        <el-button type="primary" @click="handleApplyToSite" :loading="applyingToSite">
          应用证书
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Upload, CircleCheck, Warning, CircleClose,
  Files, Key
} from '@element-plus/icons-vue'
import {
  getCertificates, applyLetsencrypt, uploadCertificate,
  deleteCertificate, renewCertificate, getCertificateContent,
  applyCertToSite, getSitesForCert, toggleAutoRenew
} from '@/api/certificate'

// 使用 any 类型以匹配项目风格
type Certificate = any
type SiteForCert = any

const loading = ref(false)
const certificates = ref<Certificate[]>([])
const sites = ref<SiteForCert[]>([])

// 统计
const validCount = computed(() => certificates.value.filter(c => c.status === 'valid').length)
const aboutToExpireCount = computed(() => certificates.value.filter(c => c.status === 'about_to_expire').length)
const expiredCount = computed(() => certificates.value.filter(c => c.status === 'expired').length)

// 对话框
const showApplyDialog = ref(false)
const showUploadDialog = ref(false)
const showDetailDialog = ref(false)
const showSiteDialog = ref(false)

// 申请表单
const applyFormRef = ref()
const applying = ref(false)
const applyForm = ref({
  domain: '',
  email: '',
  challenge: 'http',
  // HTTP 验证
  standalone: false,
  web_root: '/var/www/html',
  // DNS 验证
  dns_provider: '',
  dns_key: '',
  dns_secret: '',
  hw_project_id: ''
})
const applyRules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }]
}

// DNS 提供商标签
const dnsKeyLabel = computed(() => {
  const map: any = { aliyun: 'AccessKey ID', tencent: 'SecretId', huawei: 'Access Key ID', cloudflare: 'API Key' }
  return map[applyForm.value.dns_provider] || 'AccessKey'
})
const dnsSecretLabel = computed(() => {
  const map: any = { aliyun: 'AccessKey Secret', tencent: 'SecretKey', huawei: 'Secret Access Key', cloudflare: 'Email' }
  return map[applyForm.value.dns_provider] || 'Secret'
})
const dnsKeyPlaceholder = computed(() => {
  const map: any = { aliyun: 'LTAI5t...', tencent: 'AKID...', huawei: 'XGHT...', cloudflare: 'a1b2c3...' }
  return map[applyForm.value.dns_provider] || '请输入 AccessKey'
})
const dnsSecretPlaceholder = computed(() => {
  const map: any = { aliyun: 'AccessKey Secret', tencent: 'SecretKey', huawei: 'Secret Access Key', cloudflare: 'user@example.com' }
  return map[applyForm.value.dns_provider] || '请输入 Secret'
})

function onChallengeChange() {
  // 切换验证方式时清理 DNS 相关字段
  applyForm.value.dns_provider = ''
  applyForm.value.dns_key = ''
  applyForm.value.dns_secret = ''
  applyForm.value.hw_project_id = ''
}

// 上传表单
const uploadFormRef = ref()
const uploading = ref(false)
const uploadForm = ref({
  name: '',
  domain: '',
  cert: '',
  key: '',
  chain: ''
})
const uploadRules = {
  name: [{ required: true, message: '请输入证书名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  cert: [{ required: true, message: '请输入证书内容', trigger: 'blur' }],
  key: [{ required: true, message: '请输入私钥内容', trigger: 'blur' }]
}

// 详情
const currentCert = ref<Certificate | null>(null)
const contentTab = ref('cert')
const certContent = ref('')
const keyContent = ref('')

// 绑定站点
const selectedSiteId = ref<number | null>(null)
const applyingToSite = ref(false)

// 加载证书列表
async function loadCertificates() {
  loading.value = true
  try {
    const res: any = await getCertificates()
    certificates.value = res.data || []
  } catch {
    ElMessage.error('加载证书列表失败')
  } finally {
    loading.value = false
  }
}

// 申请证书
async function handleApply() {
  try {
    await applyFormRef.value.validate()
  } catch { return }

  applying.value = true
  try {
    await applyLetsencrypt(applyForm.value)
    ElMessage.success('证书申请成功')
    showApplyDialog.value = false
    applyForm.value = { domain: '', email: '', challenge: 'http', standalone: false, web_root: '/var/www/html', dns_provider: '', dns_key: '', dns_secret: '', hw_project_id: '' }
    loadCertificates()
  } catch (e: any) {
    ElMessage.error(e?.message || '证书申请失败')
  } finally {
    applying.value = false
  }
}

// 上传证书
async function handleUpload() {
  try {
    await uploadFormRef.value.validate()
  } catch { return }

  uploading.value = true
  try {
    await uploadCertificate(uploadForm.value)
    ElMessage.success('证书上传成功')
    showUploadDialog.value = false
    uploadForm.value = { name: '', domain: '', cert: '', key: '', chain: '' }
    loadCertificates()
  } catch (e: any) {
    ElMessage.error(e?.message || '证书上传失败')
  } finally {
    uploading.value = false
  }
}

// 查看详情
async function viewCertDetail(cert: Certificate) {
  currentCert.value = cert
  contentTab.value = 'cert'
  certContent.value = '加载中...'
  keyContent.value = '加载中...'
  showDetailDialog.value = true

  try {
    const [certRes, keyRes]: any = await Promise.all([
      getCertificateContent(cert.id, 'cert'),
      getCertificateContent(cert.id, 'key')
    ])
    certContent.value = certRes.data || '无内容'
    keyContent.value = keyRes.data || '无内容'
  } catch {
    certContent.value = '加载失败'
    keyContent.value = '加载失败'
  }
}

// 续签
async function handleRenew(cert: Certificate) {
  try {
    await ElMessageBox.confirm(
      `确定要续签证书 "${cert.name}" 吗？`,
      '确认续签',
      { type: 'warning' }
    )
  } catch { return }

  try {
    await renewCertificate(cert.id)
    ElMessage.success('证书续签成功')
    loadCertificates()
  } catch (e: any) {
    ElMessage.error(e?.message || '续签失败')
  }
}

// 切换自动续签
async function handleToggleAutoRenew(cert: Certificate) {
  try {
    const res = await toggleAutoRenew(cert.id)
    ElMessage.success(res.data?.message || '操作成功')
    loadCertificates()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
}

// 删除
async function handleDelete(cert: Certificate) {
  try {
    await ElMessageBox.confirm(
      `确定要删除证书 "${cert.name}" 吗？删除后不可恢复。`,
      '确认删除',
      { type: 'warning' }
    )
  } catch { return }

  try {
    await deleteCertificate(cert.id)
    ElMessage.success('证书已删除')
    loadCertificates()
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

// 显示绑定站点对话框
async function showApplyToSite(cert: Certificate) {
  currentCert.value = cert
  selectedSiteId.value = null
  try {
    const res: any = await getSitesForCert()
    sites.value = res.data || []
    showSiteDialog.value = true
  } catch {
    ElMessage.error('加载站点列表失败')
  }
}

// 应用到站点
async function handleApplyToSite() {
  if (!currentCert.value || !selectedSiteId.value) {
    ElMessage.warning('请选择站点')
    return
  }

  applyingToSite.value = true
  try {
    await applyCertToSite(currentCert.value.id, selectedSiteId.value)
    ElMessage.success('证书已应用到站点')
    showSiteDialog.value = false
  } catch (e: any) {
    ElMessage.error(e?.message || '应用失败')
  } finally {
    applyingToSite.value = false
  }
}

// 工具函数
function formatDate(dateStr: string): string {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '-'
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '-'
  return d.toLocaleDateString('zh-CN')
}

function statusTagType(status: string) {
  switch (status) {
    case 'valid': return 'success'
    case 'about_to_expire': return 'warning'
    case 'expired': return 'danger'
    default: return 'info'
  }
}

function statusText(status: string) {
  switch (status) {
    case 'valid': return '有效'
    case 'about_to_expire': return '即将过期'
    case 'expired': return '已过期'
    default: return status
  }
}

onMounted(() => {
  loadCertificates()
})
</script>

<style scoped>
.certificate-page {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-left h2 {
  margin: 0 0 4px 0;
  font-size: 20px;
}

.subtitle {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.header-right {
  display: flex;
  gap: 10px;
}

/* 统计卡片 */
.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
}

.stat-icon.valid { background: #f0fdf4; color: #22c55e; }
.stat-icon.warning { background: #fffbeb; color: #f59e0b; }
.stat-icon.expired { background: #fef2f2; color: #ef4444; }
.stat-icon.total { background: #eff6ff; color: #3b82f6; }

.stat-value {
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

/* 表格 */
.table-card :deep(.el-card__body) {
  padding: 0;
}

.cert-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cert-icon {
  font-size: 16px;
}

.cert-icon.letsencrypt { color: #22c55e; }
.cert-icon.custom { color: #f59e0b; }

.issuer {
  font-size: 12px;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.validity {
  font-size: 12px;
  line-height: 1.6;
}

.date-sep {
  color: var(--el-text-color-placeholder);
  font-size: 11px;
}

.expired-text { color: #ef4444; font-weight: 600; }
.warning-text { color: #f59e0b; font-weight: 600; }
</style>
