<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1>Welcome to conx CMDB</h1>
        <p>Configuration Management Database</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <el-form-item label="Email" prop="email">
          <el-input
            v-model="form.email"
            type="email"
            placeholder="Enter your email"
            size="large"
            :prefix-icon="User"
          />
        </el-form-item>

        <el-form-item label="Password" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Enter your password"
            size="large"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleSubmit"
          />
        </el-form-item>

        <div class="form-options">
          <el-checkbox v-model="form.remember">Remember me</el-checkbox>
          <el-link type="primary" @click="goToForgotPassword">
            Forgot password?
          </el-link>
        </div>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            style="width: 100%"
            @click="handleSubmit"
            :loading="loading"
          >
            Sign In
          </el-button>
        </el-form-item>

        <div class="register-link">
          <span>Don't have an account?</span>
          <el-link type="primary" @click="goToRegister">
            Register here
          </el-link>
        </div>
      </el-form>

      <div v-if="error" class="error-message">
        <el-alert
          :title="error"
          type="error"
          :closable="false"
          show-icon
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref()
const loading = ref(false)
const error = ref('')

// Form data
const form = reactive({
  email: '',
  password: '',
  remember: false
})

// Validation rules
const rules = {
  email: [
    { required: true, message: 'Please enter your email', trigger: 'blur' },
    { type: 'email', message: 'Please enter a valid email address', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter your password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' }
  ]
}

// Methods
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    error.value = ''
    await formRef.value.validate()
    
    loading.value = true
    
    const credentials = {
      email: form.email,
      password: form.password
    }
    
    await authStore.login(credentials)
    
    ElMessage.success('Login successful')
    
    // Redirect to dashboard or intended destination
    const redirectPath = router.currentRoute.value.query.redirect || '/'
    router.push(redirectPath)
  } catch (err) {
    console.error('Login failed:', err)
    error.value = err.response?.data?.message || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}

const goToRegister = () => {
  router.push('/auth/register')
}

const goToForgotPassword = () => {
  router.push('/auth/forgot-password')
}

// Check if user is already authenticated
onMounted(() => {
  if (authStore.isAuthenticated) {
    router.push('/')
  }
})
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h1 {
  margin: 0 0 10px 0;
  font-size: 28px;
  color: #303133;
  font-weight: 600;
}

.login-header p {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.register-link {
  text-align: center;
  margin-top: 20px;
  color: #606266;
}

.register-link span {
  margin-right: 5px;
}

.error-message {
  margin-top: 20px;
}

.el-form-item {
  margin-bottom: 20px;
}

.el-button {
  margin-top: 10px;
}
</style>
