<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { login, getCaptcha } from '@/api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '', captcha_id: '', captcha_code: '' })
const captchaImg = ref('')

async function fetchCaptcha() {
  try {
    const res = await getCaptcha()
    form.captcha_id = res.data.captcha_id
    form.captcha_code = ''
    captchaImg.value = res.data.captcha
  } catch { /* ignore */ }
}

async function handleLogin() {
  if (!form.username || !form.password) { ElMessage.warning('请输入用户名和密码'); return }
  loading.value = true
  try {
    const res = await login(form)
    localStorage.setItem('token', res.data.token)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch { fetchCaptcha() }
  finally { loading.value = false }
}

onMounted(fetchCaptcha)
</script>

<template>
  <div class="login-wrapper">
    <div class="login-bg">
      <div class="login-orb orb-1"></div>
      <div class="login-orb orb-2"></div>
      <div class="login-orb orb-3"></div>
    </div>
    <div class="login-card fade-in-up">
      <div class="login-header">
        <div class="login-logo">
          <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
            <rect width="40" height="40" rx="12" fill="url(#logo-grad)"/>
            <path d="M12 20L18 14L26 22L20 28L12 20Z" fill="white" opacity="0.9"/>
            <path d="M18 20L22 16L28 22L22 28L18 20Z" fill="white" opacity="0.6"/>
            <defs><linearGradient id="logo-grad" x1="0" y1="0" x2="40" y2="40"><stop stop-color="#4f8cff"/><stop offset="1" stop-color="#6c5ce7"/></linearGradient></defs>
          </svg>
        </div>
        <h1 class="login-title">OpsManage</h1>
        <p class="login-subtitle">轻量级服务器运维管理面板</p>
      </div>
      <el-form :model="form" @keyup.enter="handleLogin" size="large">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <div class="captcha-row">
            <el-input v-model="form.captcha_code" placeholder="验证码" />
            <img v-if="captchaImg" :src="captchaImg" class="captcha-img" title="点击刷新" @click="fetchCaptcha" />
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" class="login-btn" @click="handleLogin">
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">首次登录后请立即修改默认密码</div>
    </div>
  </div>
</template>

<style scoped>
.login-wrapper {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f0f23;
  position: relative;
  overflow: hidden;
}
.login-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}
.login-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.5;
  animation: orbFloat 20s ease-in-out infinite;
}
.orb-1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #4f8cff 0%, transparent 70%);
  top: -10%; left: -10%;
  animation-delay: 0s;
}
.orb-2 {
  width: 400px; height: 400px;
  background: radial-gradient(circle, #6c5ce7 0%, transparent 70%);
  bottom: -10%; right: -10%;
  animation-delay: -7s;
}
.orb-3 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, #00b4d8 0%, transparent 70%);
  top: 50%; left: 60%;
  animation-delay: -14s;
}
@keyframes orbFloat {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -50px) scale(1.1); }
  66% { transform: translate(-20px, 20px) scale(0.9); }
}

.login-card {
  width: 420px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 48px 40px 36px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  position: relative;
  z-index: 1;
}

.login-header {
  text-align: center;
  margin-bottom: 36px;
}
.login-logo {
  display: inline-flex;
  margin-bottom: 16px;
}
.login-title {
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, #4f8cff, #6c5ce7);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 6px;
}
.login-subtitle {
  color: #86909c;
  font-size: 14px;
}

.captcha-row {
  display: flex;
  gap: 12px;
  width: 100%;
}
.captcha-img {
  height: 40px;
  cursor: pointer;
  border-radius: 8px;
  border: 1px solid #e5e6eb;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 10px !important;
  background: linear-gradient(135deg, #4f8cff, #6c5ce7) !important;
  border: none !important;
  box-shadow: 0 4px 16px rgba(79, 140, 255, 0.3);
  transition: all 0.3s ease;
}
.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(79, 140, 255, 0.45);
}

.login-footer {
  text-align: center;
  color: #c0c4cc;
  font-size: 12px;
  margin-top: 16px;
}
</style>
