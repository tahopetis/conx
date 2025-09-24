<template>
  <div class="reset-password-container">
    <div class="reset-password-card">
      <div class="reset-password-header">
        <h1>Set New Password</h1>
        <p>Enter your new password below</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <el-form-item label="New Password" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Enter your new password"
            size="large"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item label="Confirm New Password" prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="Confirm your new password"
            size="large"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleSubmit"
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
            Reset Password
          </el-button>
        </el-form-item>

        <div class="back-link">
          <el-link type="primary" @click="goToLogin">
            <el-icon><ArrowLeft /></el-icon>
            Back to Login
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
          title="Password reset successful! You can now log in with your new password."
          type="success"
          :closable="false"
          show-icon
        />
      </div>

      <div v-if="tokenError" class="error-state">
        <el-result
          icon="error"
          title="Invalid or Expired Link"
          sub-title="The password reset link is invalid or has expired. Please request a new one."
        >
          <template #extra>
            <el-button type="primary" @click="goToForgotPassword">
              Request New Reset Link
            </el-button>
          </template>
        </el-result>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, ArrowLeft } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const formRef = ref()
const loading = ref(false)
const error = ref('')
const success = ref(false)
const tokenError = ref(false)

// Form data
const form = reactive({
  password: '',
  confirmPassword: ''
})

// Custom validation for password confirmation
const validateConfirmPassword = (rule, value, callback) => {
  if (value !== form.password) {
    callback(new Error('Passwords do not match'))
  } else {
    callback()
  }
}

// Validation rules
const rules = {
  password: [
    { required: true, message: 'Please enter a new password', trigger: 'blur' },
    { min: 8, message: 'Password must be at least 8 characters', trigger: 'blur' },
    { 
      pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/, 
      message: 'Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character', 
      trigger: 'blur' 
    }
  ],
  confirmPassword: [
    { required: true, message: 'Please confirm your new password', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// Methods
const handleSubmit = async () => {
  if (!formRef.value || !route.params.token) return
  
  try {
    error.value = ''
    success.value = false
    await formRef.value.validate()
    
    loading.value = true
    
    const passwordData = {
      password: form.password,
      confirm_password: form.confirmPassword
    }
    
    await authStore.resetPassword(route.params.token, passwordData)
    
    success.value = true
    ElMessage.success('Password reset successful!')
    
    // Clear form
    form.password = ''
    form.confirmPassword = ''
    
    // Redirect to login after a delay
    setTimeout(() => {
      router.push('/auth/login')
    }, 3000)
  } catch (err) {
    console.error('Password reset failed:', err)
    if (err.response?.status === 400 && err.response?.data?.message?.includes('invalid') || err.response?.data?.message?.includes('expired')) {
      tokenError.value = true
    } else {
      error.value = err.response?.data?.message || 'Failed to reset password. Please try again.'
    }
  } finally {
    loading.value = false
  }
}

const goToLogin = () => {
  router.push('/auth/login')
}

const goToForgotPassword = () => {
  router.push('/auth/forgot-password')
}

// Check if token is present on component mount
onMounted(() => {
  if (!route.params.token) {
    tokenError.value = true
  }
})
</script>

<style scoped>
.reset-password-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.reset-password-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.reset-password-header {
  text-align: center;
  margin-bottom: 30px;
}

.reset-password-header h1 {
  margin: 0 0 10px 0;
  font-size: 28px;
  color: #303133;
  font-weight: 600;
}

.reset-password-header p {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.back-link {
  text-align: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}

.error-message,
.success-message {
  margin-top: 20px;
}

.error-state {
  margin-top: 20px;
}

.el-form-item {
  margin-bottom: 20px;
}

.el-button {
  margin-top: 10px;
}
</style>
