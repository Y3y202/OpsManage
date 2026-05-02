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
  } catch {
    // ignore
  }
}

async function handleLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  if (!form.captcha_code) {
    ElMessage.warning('请输入验证码')
    return
  }
  loading.value = true
  try {
    const res = await login(form)
    localStorage.setItem('token', res.data.token)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch {
    fetchCaptcha()
  } finally {
    loading.value = false
  }
}

onMounted(fetchCaptcha)
</script>

<template>
  <div style="height: 100vh; display: flex; align-items: center; justify-content: center; background: #2d3a4b">
    <el-card style="width: 400px">
      <template #header>
        <div style="text-align: center; font-size: 20px; font-weight: bold">OpsManage</div>
      </template>
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <div style="display: flex; gap: 10px; width: 100%">
            <el-input v-model="form.captcha_code" placeholder="验证码" size="large" style="flex: 1" />
            <img
              v-if="captchaImg"
              :src="captchaImg"
              style="height: 40px; cursor: pointer; border-radius: 4px"
              title="点击刷新验证码"
              @click="fetchCaptcha"
            />
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" size="large" @click="handleLogin">登 录</el-button>
        </el-form-item>
      </el-form>
      <div style="text-align: center; color: #999; font-size: 12px">默认账号: admin / admin123</div>
    </el-card>
  </div>
</template>
