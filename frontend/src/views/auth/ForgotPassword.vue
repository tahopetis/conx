<template>
  <div class="forgot-password-container">
    <div class="forgot-password-card">
      <div class="forgot-password-header">
        <h1>Reset Password</h1>
        <p>Enter your email address and we'll send you a link to reset your password</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <el-form-item label="Email Address" prop="email">
          <el-input
            v-model="form.email"
            type="email"
            placeholder="Enter your email address"
            size="large"
            :prefix-icon="Message"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            style="width: 100%"
            @click="handleSubmit"
            :loading="loading"
          >
            Send Reset Link
          </el-button>
        </el-form-item>

        <div class="back-links">
          <el-link type="primary" @click="goToLogin">
            <el-icon><ArrowLeft /></el-icon>
            Back to Login
          </el-link>
          <el-link type="primary" @click="goToRegister">
            Create an Account
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

      <div v-if="success" class="success-message">
        <el-alert
          :title="successMessage"
          type="success"
          :closable="false"
          show-icon
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Message, ArrowLeft } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref()
const loading = ref(false)
const error = ref('')
const success = ref(false)
const successMessage = ref('')

// Form data
const form = reactive({
  email: ''
})

// Validation rules
const rules = {
  email: [
    { required: true, message: 'Please enter your email address', trigger: 'blur' },
    { type: 'email', message: 'Please enter a valid email address', trigger: 'blur' }
  ]
}

// Methods
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    error.value = ''
    success.value = false
    await formRef.value.validate()
    
    loading.value = true
    
    await authStore.requestPasswordReset(form.email)
    
    success.value = true
    successMessage.value = `Password reset link has been sent to ${form.email}. Please check your inbox and follow the instructions.`
    
    ElMessage.success('Password reset email sent successfully!')
    
    // Clear form
    form.email = ''
    
    // Redirect to login after a delay
    setTimeout(() => {
      router.push('/auth/login')
    }, 5000)
  } catch (err) {
    console.error('Password reset request failed:', err)
    error.value = err.response?.data?.message || 'Failed to send password reset email. Please try again.'
  } finally {
    loading.value = false
  }
}

const goToLogin = () => {
  router.push('/auth/login')
}

const goToRegister = () => {
  router.push('/auth/register')
}
</script>

<style scoped>
.forgot-password-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.forgot-password-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.forgot-password-header {
  text-align: center;
  margin-bottom: 30px;
}

.forgot-password-header h1 {
  margin: 0 0 10px 0;
  font-size: 28px;
  color: #303133;
  font-weight: 600;
}

.forgot-password-header p {
  margin: 0;
  color: #909399;
  font-size: 14px;
  line-height: 1.5;
}

.back-links {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}

.error-message,
.success-message {
  margin-top: 20px;
}

.el-form-item {
  margin-bottom: 20px;
}

.el-button {
  margin-top: 10px;
}
</style>
