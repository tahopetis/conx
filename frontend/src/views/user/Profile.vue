<template>
  <div class="profile-container">
    <div class="profile-header">
      <h1>My Profile</h1>
      <p>Manage your personal information and account settings</p>
    </div>

    <el-row :gutter="20">
      <!-- Profile Information -->
      <el-col :span="16">
        <el-card class="profile-card">
          <template #header>
            <div class="card-header">
              <h3>Profile Information</h3>
              <el-button 
                type="primary" 
                size="small" 
                @click="isEditing = !isEditing"
                :disabled="loading"
              >
                {{ isEditing ? 'Cancel' : 'Edit' }}
              </el-button>
            </div>
          </template>

          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-width="120px"
            label-position="left"
            @submit.prevent="handleSubmit"
          >
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Full Name" prop="full_name">
                  <el-input
                    v-model="form.full_name"
                    :disabled="!isEditing"
                    placeholder="Enter your full name"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Email" prop="email">
                  <el-input
                    v-model="form.email"
                    disabled
                    placeholder="Your email address"
                  />
                  <div class="field-help">Email cannot be changed</div>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Username" prop="username">
                  <el-input
                    v-model="form.username"
                    :disabled="!isEditing"
                    placeholder="Enter your username"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Department" prop="department">
                  <el-input
                    v-model="form.department"
                    :disabled="!isEditing"
                    placeholder="Enter your department"
                  />
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Job Title" prop="job_title">
                  <el-input
                    v-model="form.job_title"
                    :disabled="!isEditing"
                    placeholder="Enter your job title"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Phone" prop="phone">
                  <el-input
                    v-model="form.phone"
                    :disabled="!isEditing"
                    placeholder="Enter your phone number"
                  />
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item label="Bio" prop="bio">
              <el-input
                v-model="form.bio"
                type="textarea"
                :rows="3"
                :disabled="!isEditing"
                placeholder="Tell us about yourself"
              />
            </el-form-item>

            <div v-if="isEditing" class="form-actions">
              <el-button @click="cancelEdit">Cancel</el-button>
              <el-button type="primary" @click="handleSubmit" :loading="loading">
                Save Changes
              </el-button>
            </div>
          </el-form>
        </el-card>
      </el-col>

      <!-- Account Actions -->
      <el-col :span="8">
        <el-card class="actions-card">
          <template #header>
            <h3>Account Actions</h3>
          </template>

          <div class="action-section">
            <h4>Change Password</h4>
            <p>Update your password to keep your account secure</p>
            <el-button type="primary" @click="showChangePassword = true" class="action-button">
              Change Password
            </el-button>
          </div>

          <el-divider />

          <div class="action-section">
            <h4>Two-Factor Authentication</h4>
            <p>Add an extra layer of security to your account</p>
            <el-button 
              :type="authStore.user?.two_factor_enabled ? 'danger' : 'success'"
              @click="toggle2FA"
              :loading="loading2FA"
              class="action-button"
            >
              {{ authStore.user?.two_factor_enabled ? 'Disable 2FA' : 'Enable 2FA' }}
            </el-button>
          </div>

          <el-divider />

          <div class="action-section">
            <h4>Account Activity</h4>
            <p>View your recent login sessions and activity</p>
            <el-button @click="goToSettings" class="action-button">
              View Activity
            </el-button>
          </div>

          <el-divider />

          <div class="action-section danger-section">
            <h4>Danger Zone</h4>
            <p>These actions are irreversible</p>
            <el-button type="danger" @click="confirmDeleteAccount" class="action-button">
              Delete Account
            </el-button>
          </div>
        </el-card>

        <!-- Profile Stats -->
        <el-card class="stats-card">
          <template #header>
            <h3>Profile Statistics</h3>
          </template>
          <div class="stats-list">
            <div class="stat-item">
              <span class="stat-label">Member Since</span>
              <span class="stat-value">{{ formatDate(authStore.user?.created_at) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Last Login</span>
              <span class="stat-value">{{ formatDate(authStore.user?.last_login) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Roles</span>
              <div class="stat-value">
                <el-tag
                  v-for="role in authStore.userRoles"
                  :key="role"
                  size="small"
                  class="role-tag"
                >
                  {{ role }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Change Password Dialog -->
    <el-dialog
      v-model="showChangePassword"
      title="Change Password"
      width="500px"
      @close="resetPasswordForm"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-width="120px"
        label-position="left"
      >
        <el-form-item label="Current Password" prop="current_password">
          <el-input
            v-model="passwordForm.current_password"
            type="password"
            placeholder="Enter your current password"
            show-password
          />
        </el-form-item>
        <el-form-item label="New Password" prop="new_password">
          <el-input
            v-model="passwordForm.new_password"
            type="password"
            placeholder="Enter your new password"
            show-password
          />
        </el-form-item>
        <el-form-item label="Confirm Password" prop="confirm_password">
          <el-input
            v-model="passwordForm.confirm_password"
            type="password"
            placeholder="Confirm your new password"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showChangePassword = false">Cancel</el-button>
        <el-button type="primary" @click="handleChangePassword" :loading="loadingPassword">
          Change Password
        </el-button>
      </template>
    </el-dialog>

    <!-- Delete Account Confirmation -->
    <el-dialog
      v-model="showDeleteConfirm"
      title="Delete Account"
      width="400px"
    >
      <p>Are you sure you want to delete your account?</p>
      <p>This action cannot be undone and all your data will be permanently removed.</p>
      <template #footer>
        <el-button @click="showDeleteConfirm = false">Cancel</el-button>
        <el-button type="danger" @click="deleteAccount" :loading="loadingDelete">
          Delete Account
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'
import dayjs from 'dayjs'

const router = useRouter()
const authStore = useAuthStore()

// State
const formRef = ref()
const passwordFormRef = ref()
const loading = ref(false)
const loadingPassword = ref(false)
const loading2FA = ref(false)
const loadingDelete = ref(false)
const isEditing = ref(false)
const showChangePassword = ref(false)
const showDeleteConfirm = ref(false)

// Form data
const form = reactive({
  full_name: '',
  email: '',
  username: '',
  department: '',
  job_title: '',
  phone: '',
  bio: ''
})

const passwordForm = reactive({
  current_password: '',
  new_password: '',
  confirm_password: ''
})

const originalForm = reactive({})

// Validation rules
const rules = {
  full_name: [
    { required: true, message: 'Please enter your full name', trigger: 'blur' },
    { min: 2, max: 100, message: 'Length should be 2 to 100 characters', trigger: 'blur' }
  ],
  username: [
    { required: true, message: 'Please enter your username', trigger: 'blur' },
    { min: 3, max: 50, message: 'Username must be 3 to 50 characters', trigger: 'blur' }
  ],
  department: [
    { required: true, message: 'Please enter your department', trigger: 'blur' }
  ],
  job_title: [
    { required: true, message: 'Please enter your job title', trigger: 'blur' }
  ]
}

const passwordRules = {
  current_password: [
    { required: true, message: 'Please enter your current password', trigger: 'blur' }
  ],
  new_password: [
    { required: true, message: 'Please enter your new password', trigger: 'blur' },
    { min: 8, message: 'Password must be at least 8 characters', trigger: 'blur' },
    { 
      pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/, 
      message: 'Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character', 
      trigger: 'blur' 
    }
  ],
  confirm_password: [
    { required: true, message: 'Please confirm your new password', trigger: 'blur' },
    { 
      validator: (rule, value, callback) => {
        if (value !== passwordForm.new_password) {
          callback(new Error('Passwords do not match'))
        } else {
          callback()
        }
      }, 
      trigger: 'blur' 
    }
  ]
}

// Methods
const loadUserProfile = () => {
  if (authStore.user) {
    Object.keys(form).forEach(key => {
      if (authStore.user[key] !== undefined) {
        form[key] = authStore.user[key]
      }
    })
    // Store original form data for cancel functionality
    Object.assign(originalForm, form)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    loading.value = true
    
    await authStore.updateProfile(form)
    
    ElMessage.success('Profile updated successfully')
    isEditing.value = false
    Object.assign(originalForm, form)
  } catch (error) {
    console.error('Failed to update profile:', error)
    ElMessage.error('Failed to update profile')
  } finally {
    loading.value = false
  }
}

const cancelEdit = () => {
  Object.assign(form, originalForm)
  isEditing.value = false
}

const handleChangePassword = async () => {
  if (!passwordFormRef.value) return
  
  try {
    await passwordFormRef.value.validate()
    loadingPassword.value = true
    
    await authStore.changePassword({
      current_password: passwordForm.current_password,
      new_password: passwordForm.new_password
    })
    
    ElMessage.success('Password changed successfully')
    showChangePassword.value = false
    resetPasswordForm()
  } catch (error) {
    console.error('Failed to change password:', error)
    ElMessage.error('Failed to change password')
  } finally {
    loadingPassword.value = false
  }
}

const resetPasswordForm = () => {
  passwordForm.current_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
}

const toggle2FA = async () => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to ${authStore.user?.two_factor_enabled ? 'disable' : 'enable'} two-factor authentication?`,
      'Confirm 2FA Action',
      {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning'
      }
    )
    
    loading2FA.value = true
    
    // TODO: Implement 2FA toggle API call
    // await api.toggle2FA()
    
    const newStatus = !authStore.user?.two_factor_enabled
    // Update user object (this would normally come from the API response)
    authStore.user.two_factor_enabled = newStatus
    
    ElMessage.success(`Two-factor authentication ${newStatus ? 'enabled' : 'disabled'} successfully`)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to toggle 2FA:', error)
      ElMessage.error('Failed to toggle two-factor authentication')
    }
  } finally {
    loading2FA.value = false
  }
}

const confirmDeleteAccount = () => {
  showDeleteConfirm.value = true
}

const deleteAccount = async () => {
  try {
    loadingDelete.value = true
    
    // TODO: Implement delete account API call
    // await api.deleteAccount()
    
    ElMessage.success('Account deleted successfully')
    authStore.logout()
    router.push('/auth/login')
  } catch (error) {
    console.error('Failed to delete account:', error)
    ElMessage.error('Failed to delete account')
  } finally {
    loadingDelete.value = false
    showDeleteConfirm.value = false
  }
}

const goToSettings = () => {
  router.push('/settings')
}

const formatDate = (date) => {
  return date ? dayjs(date).format('YYYY-MM-DD HH:mm:ss') : 'Never'
}

// Lifecycle
onMounted(() => {
  loadUserProfile()
})
</script>

<style scoped>
.profile-container {
  padding: 20px;
}

.profile-header {
  margin-bottom: 30px;
}

.profile-header h1 {
  margin: 0 0 5px 0;
  font-size: 28px;
  color: #303133;
}

.profile-header p {
  margin: 0;
  color: #909399;
  font-size: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  color: #303133;
}

.field-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}

.actions-card,
.stats-card {
  margin-bottom: 20px;
}

.action-section {
  margin-bottom: 20px;
}

.action-section h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #303133;
  font-weight: 600;
}

.action-section p {
  margin: 0 0 12px 0;
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.action-button {
  width: 100%;
}

.danger-section h4 {
  color: #f56c6c;
}

.stats-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-label {
  font-size: 14px;
  color: #606266;
}

.stat-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.role-tag {
  margin-left: 4px;
}
</style>
