<template>
  <div class="register-container">
    <div class="register-card">
      <div class="register-header">
        <h1>Create Account</h1>
        <p>Join conx CMDB to manage your configuration items</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <!-- Personal Information -->
        <div class="form-section">
          <h3>Personal Information</h3>
          <el-form-item label="Full Name" prop="full_name">
            <el-input
              v-model="form.full_name"
              placeholder="Enter your full name"
              size="large"
              :prefix-icon="User"
            />
          </el-form-item>

          <el-form-item label="Email" prop="email">
            <el-input
              v-model="form.email"
              type="email"
              placeholder="Enter your email"
              size="large"
              :prefix-icon="Message"
            />
          </el-form-item>
        </div>

        <!-- Account Information -->
        <div class="form-section">
          <h3>Account Information</h3>
          <el-form-item label="Username" prop="username">
            <el-input
              v-model="form.username"
              placeholder="Choose a username"
              size="large"
              :prefix-icon="UserFilled"
            />
          </el-form-item>

          <el-form-item label="Password" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="Create a password"
              size="large"
              :prefix-icon="Lock"
              show-password
            />
          </el-form-item>

          <el-form-item label="Confirm Password" prop="confirmPassword">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              placeholder="Confirm your password"
              size="large"
              :prefix-icon="Lock"
              show-password
              @keyup.enter="handleSubmit"
            />
          </el-form-item>
        </div>

        <!-- Professional Information -->
        <div class="form-section">
          <h3>Professional Information</h3>
          <el-form-item label="Department" prop="department">
            <el-input
              v-model="form.department"
              placeholder="Enter your department"
              size="large"
              :prefix-icon="OfficeBuilding"
            />
          </el-form-item>

          <el-form-item label="Job Title" prop="job_title">
            <el-input
              v-model="form.job_title"
              placeholder="Enter your job title"
              size="large"
              :prefix-icon="Briefcase"
            />
          </el-form-item>
        </div>

        <!-- Terms and Conditions -->
        <el-form-item prop="agreeTerms">
          <el-checkbox v-model="form.agreeTerms">
            I agree to the <el-link type="primary" @click="showTerms">Terms of Service</el-link>
            and <el-link type="primary" @click="showPrivacy">Privacy Policy</el-link>
          </el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            style="width: 100%"
            @click="handleSubmit"
            :loading="loading"
          >
            Create Account
          </el-button>
        </el-form-item>

        <div class="login-link">
          <span>Already have an account?</span>
          <el-link type="primary" @click="goToLogin">
            Sign in here
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
          title="Registration successful! Please check your email to verify your account."
          type="success"
          :closable="false"
          show-icon
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { 
  User, 
  Message, 
  UserFilled, 
  Lock, 
  OfficeBuilding, 
  Briefcase 
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref()
const loading = ref(false)
const error = ref('')
const success = ref(false)

// Form data
const form = reactive({
  full_name: '',
  email: '',
  username: '',
  password: '',
  confirmPassword: '',
  department: '',
  job_title: '',
  agreeTerms: false
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
  full_name: [
    { required: true, message: 'Please enter your full name', trigger: 'blur' },
    { min: 2, max: 100, message: 'Length should be 2 to 100 characters', trigger: 'blur' }
  ],
  email: [
    { required: true, message: 'Please enter your email', trigger: 'blur' },
    { type: 'email', message: 'Please enter a valid email address', trigger: 'blur' }
  ],
  username: [
    { required: true, message: 'Please enter a username', trigger: 'blur' },
    { min: 3, max: 50, message: 'Username must be 3 to 50 characters', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: 'Username can only contain letters, numbers, and underscores', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter a password', trigger: 'blur' },
    { min: 8, message: 'Password must be at least 8 characters', trigger: 'blur' },
    { 
      pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/, 
      message: 'Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character', 
      trigger: 'blur' 
    }
  ],
  confirmPassword: [
    { required: true, message: 'Please confirm your password', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ],
  department: [
    { required: true, message: 'Please enter your department', trigger: 'blur' }
  ],
  job_title: [
    { required: true, message: 'Please enter your job title', trigger: 'blur' }
  ],
  agreeTerms: [
    { 
      validator: (rule, value, callback) => {
        if (!value) {
          callback(new Error('You must agree to the terms and conditions'))
        } else {
          callback()
        }
      }, 
      trigger: 'change' 
    }
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
    
    const userData = {
      full_name: form.full_name,
      email: form.email,
      username: form.username,
      password: form.password,
      department: form.department,
      job_title: form.job_title
    }
    
    await authStore.register(userData)
    
    success.value = true
    ElMessage.success('Registration successful! Please check your email to verify your account.')
    
    // Redirect to login after successful registration
    setTimeout(() => {
      router.push('/auth/login')
    }, 3000)
  } catch (err) {
    console.error('Registration failed:', err)
    error.value = err.response?.data?.message || 'Registration failed. Please try again.'
  } finally {
    loading.value = false
  }
}

const goToLogin = () => {
  router.push('/auth/login')
}

const showTerms = () => {
  // TODO: Implement terms of service modal or page
  ElMessage.info('Terms of Service will be available soon.')
}

const showPrivacy = () => {
  // TODO: Implement privacy policy modal or page
  ElMessage.info('Privacy Policy will be available soon.')
}

// Check if user is already authenticated
const isAuthenticated = computed(() => authStore.isAuthenticated)
if (isAuthenticated.value) {
  router.push('/')
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.register-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
}

.register-header {
  text-align: center;
  margin-bottom: 30px;
}

.register-header h1 {
  margin: 0 0 10px 0;
  font-size: 28px;
  color: #303133;
  font-weight: 600;
}

.register-header p {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.form-section {
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 1px solid #ebeef5;
}

.form-section:last-child {
  border-bottom: none;
  margin-bottom: 20px;
}

.form-section h3 {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: #606266;
  font-weight: 600;
}

.login-link {
  text-align: center;
  margin-top: 20px;
  color: #606266;
}

.login-link span {
  margin-right: 5px;
}

.error-message,
.success-message {
  margin-top: 20px;
}

.el-form-item {
  margin-bottom: 18px;
}

.el-button {
  margin-top: 10px;
}

/* Custom scrollbar for register card */
.register-card::-webkit-scrollbar {
  width: 6px;
}

.register-card::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.register-card::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.register-card::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>
