<template>
  <div class="settings-container">
    <div class="settings-header">
      <h1>Account Settings</h1>
      <p>Manage your account preferences, security, and activity</p>
    </div>

    <el-row :gutter="20">
      <!-- Settings Navigation -->
      <el-col :span="6">
        <el-card class="nav-card">
          <el-menu
            v-model:default-active="activeTab"
            class="settings-menu"
            @select="handleTabSelect"
          >
            <el-menu-item index="sessions">
              <el-icon><Clock /></el-icon>
              <span>Active Sessions</span>
            </el-menu-item>
            <el-menu-item index="security">
              <el-icon><Lock /></el-icon>
              <span>Security Log</span>
            </el-menu-item>
            <el-menu-item index="preferences">
              <el-icon><Setting /></el-icon>
              <span>Preferences</span>
            </el-menu-item>
            <el-menu-item index="notifications">
              <el-icon><Bell /></el-icon>
              <span>Notifications</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- Settings Content -->
      <el-col :span="18">
        <!-- Active Sessions -->
        <div v-show="activeTab === 'sessions'" class="settings-content">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>Active Sessions</h3>
                <el-button type="danger" size="small" @click="revokeAllSessions" :loading="loadingRevokeAll">
                  Revoke All Sessions
                </el-button>
              </div>
            </template>
            
            <div v-if="loadingSessions" class="loading-container">
              <el-skeleton :rows="5" animated />
            </div>
            
            <div v-else-if="sessions.length > 0">
              <el-table :data="sessions" style="width: 100%">
                <el-table-column prop="device_info" label="Device" min-width="200">
                  <template #default="{ row }">
                    <div class="device-info">
                      <el-icon><Monitor /></el-icon>
                      <div>
                        <div class="device-name">{{ row.device_info || 'Unknown Device' }}</div>
                        <div class="device-details">{{ row.browser_info || 'Unknown Browser' }}</div>
                      </div>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="ip_address" label="IP Address" width="150" />
                <el-table-column prop="location" label="Location" width="150" />
                <el-table-column prop="created_at" label="Started" width="180">
                  <template #default="{ row }">
                    {{ formatDate(row.created_at) }}
                  </template>
                </el-table-column>
                <el-table-column prop="last_activity" label="Last Activity" width="180">
                  <template #default="{ row }">
                    {{ formatDate(row.last_activity) }}
                  </template>
                </el-table-column>
                <el-table-column label="Current Session" width="150">
                  <template #default="{ row }">
                    <el-tag v-if="row.is_current" type="success" size="small">
                      Current Session
                    </el-tag>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column label="Actions" width="100" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      v-if="!row.is_current"
                      size="small"
                      type="danger"
                      @click="revokeSession(row)"
                      :loading="row.loading"
                    >
                      Revoke
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            
            <div v-else class="empty-state">
              <el-empty description="No active sessions found" />
            </div>
          </el-card>
        </div>

        <!-- Security Log -->
        <div v-show="activeTab === 'security'" class="settings-content">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>Security Log</h3>
                <el-button size="small" @click="refreshSecurityLog" :loading="loadingSecurity">
                  <el-icon><RefreshRight /></el-icon>
                  Refresh
                </el-button>
              </div>
            </template>
            
            <div v-if="loadingSecurity" class="loading-container">
              <el-skeleton :rows="8" animated />
            </div>
            
            <div v-else-if="securityEvents.length > 0">
              <el-timeline>
                <el-timeline-item
                  v-for="event in securityEvents"
                  :key="event.id"
                  :timestamp="formatDate(event.created_at)"
                  :type="getSecurityEventType(event.event_type)"
                >
                  <h4>{{ event.event_type }}</h4>
                  <p>{{ event.description }}</p>
                  <div class="event-details">
                    <span class="event-ip">IP: {{ event.ip_address }}</span>
                    <span class="event-device">Device: {{ event.device_info || 'Unknown' }}</span>
                  </div>
                </el-timeline-item>
              </el-timeline>
              
              <!-- Pagination for security log -->
              <div class="pagination-container">
                <el-pagination
                  v-model:current-page="securityPagination.page"
                  v-model:page-size="securityPagination.limit"
                  :page-sizes="[10, 20, 50]"
                  :total="securityPagination.total"
                  layout="total, sizes, prev, pager, next"
                  @size-change="handleSecuritySizeChange"
                  @current-change="handleSecurityPageChange"
                />
              </div>
            </div>
            
            <div v-else class="empty-state">
              <el-empty description="No security events found" />
            </div>
          </el-card>
        </div>

        <!-- Preferences -->
        <div v-show="activeTab === 'preferences'" class="settings-content">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>User Preferences</h3>
                <el-button 
                  type="primary" 
                  size="small" 
                  @click="savePreferences"
                  :loading="loadingPreferences"
                  :disabled="!preferencesChanged"
                >
                  Save Changes
                </el-button>
              </div>
            </template>
            
            <el-form
              ref="preferencesFormRef"
              :model="preferencesForm"
              label-width="200px"
              label-position="left"
            >
              <el-form-item label="Theme">
                <el-select v-model="preferencesForm.theme" placeholder="Select theme">
                  <el-option label="Light" value="light" />
                  <el-option label="Dark" value="dark" />
                  <el-option label="System" value="system" />
                </el-select>
              </el-form-item>
              
              <el-form-item label="Language">
                <el-select v-model="preferencesForm.language" placeholder="Select language">
                  <el-option label="English" value="en" />
                  <el-option label="Spanish" value="es" />
                  <el-option label="French" value="fr" />
                  <el-option label="German" value="de" />
                </el-select>
              </el-form-item>
              
              <el-form-item label="Time Zone">
                <el-select v-model="preferencesForm.timezone" placeholder="Select time zone" filterable>
                  <el-option
                    v-for="tz in timeZones"
                    :key="tz.value"
                    :label="tz.label"
                    :value="tz.value"
                  />
                </el-select>
              </el-form-item>
              
              <el-form-item label="Date Format">
                <el-select v-model="preferencesForm.date_format" placeholder="Select date format">
                  <el-option label="MM/DD/YYYY" value="mm/dd/yyyy" />
                  <el-option label="DD/MM/YYYY" value="dd/mm/yyyy" />
                  <el-option label="YYYY-MM-DD" value="yyyy-mm-dd" />
                </el-select>
              </el-form-item>
              
              <el-form-item label="Items per page">
                <el-select v-model="preferencesForm.items_per_page" placeholder="Select items per page">
                  <el-option label="10" :value="10" />
                  <el-option label="20" :value="20" />
                  <el-option label="50" :value="50" />
                  <el-option label="100" :value="100" />
                </el-select>
              </el-form-item>
              
              <el-divider />
              
              <el-form-item label="Email Notifications">
                <el-checkbox-group v-model="preferencesForm.email_notifications">
                  <el-checkbox value="security_alerts">Security Alerts</el-checkbox>
                  <el-checkbox value="system_updates">System Updates</el-checkbox>
                  <el-checkbox value="weekly_summary">Weekly Summary</el-checkbox>
                  <el-checkbox value="mention_notifications">Mentions</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              
              <el-form-item label="In-App Notifications">
                <el-switch
                  v-model="preferencesForm.in_app_notifications"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-form-item label="Desktop Notifications">
                <el-switch
                  v-model="preferencesForm.desktop_notifications"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
            </el-form>
          </el-card>
        </div>

        <!-- Notifications -->
        <div v-show="activeTab === 'notifications'" class="settings-content">
          <el-card>
            <template #header>
              <div class="card-header">
                <h3>Notification Settings</h3>
                <el-button 
                  type="primary" 
                  size="small" 
                  @click="saveNotificationSettings"
                  :loading="loadingNotifications"
                  :disabled="!notificationsChanged"
                >
                  Save Changes
                </el-button>
              </div>
            </template>
            
            <el-form
              ref="notificationsFormRef"
              :model="notificationsForm"
              label-width="250px"
              label-position="left"
            >
              <h4>CI Management Notifications</h4>
              <el-form-item label="CI Created">
                <el-switch
                  v-model="notificationsForm.ci_created"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-form-item label="CI Updated">
                <el-switch
                  v-model="notificationsForm.ci_updated"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-form-item label="CI Deleted">
                <el-switch
                  v-model="notificationsForm.ci_deleted"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-divider />
              
              <h4>System Notifications</h4>
              <el-form-item label="System Maintenance">
                <el-switch
                  v-model="notificationsForm.system_maintenance"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-form-item label="New Features">
                <el-switch
                  v-model="notificationsForm.new_features"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-form-item label="Security Updates">
                <el-switch
                  v-model="notificationsForm.security_updates"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
              
              <el-divider />
              
              <h4>Notification Delivery</h4>
              <el-form-item label="Quiet Hours">
                <el-time-picker
                  v-model="notificationsForm.quiet_hours_start"
                  placeholder="Start time"
                  format="HH:mm"
                />
                <span class="time-separator">to</span>
                <el-time-picker
                  v-model="notificationsForm.quiet_hours_end"
                  placeholder="End time"
                  format="HH:mm"
                />
              </el-form-item>
              
              <el-form-item label="Weekend Notifications">
                <el-switch
                  v-model="notificationsForm.weekend_notifications"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
            </el-form>
          </el-card>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Clock, 
  Lock, 
  Setting, 
  Bell, 
  Monitor, 
  RefreshRight 
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'
import dayjs from 'dayjs'

const router = useRouter()
const authStore = useAuthStore()

// State
const activeTab = ref('sessions')
const loadingSessions = ref(false)
const loadingSecurity = ref(false)
const loadingRevokeAll = ref(false)
const loadingPreferences = ref(false)
const loadingNotifications = ref(false)

// Sessions
const sessions = ref([])

// Security Log
const securityEvents = ref([])
const securityPagination = reactive({
  page: 1,
  limit: 20,
  total: 0
})

// Preferences
const preferencesForm = reactive({
  theme: 'light',
  language: 'en',
  timezone: 'UTC',
  date_format: 'yyyy-mm-dd',
  items_per_page: 20,
  email_notifications: ['security_alerts'],
  in_app_notifications: true,
  desktop_notifications: false
})

const originalPreferences = reactive({})

// Notifications
const notificationsForm = reactive({
  ci_created: true,
  ci_updated: true,
  ci_deleted: true,
  system_maintenance: true,
  new_features: true,
  security_updates: true,
  quiet_hours_start: null,
  quiet_hours_end: null,
  weekend_notifications: false
})

const originalNotifications = reactive({})

// Time zones list
const timeZones = [
  { value: 'UTC', label: 'UTC' },
  { value: 'America/New_York', label: 'Eastern Time (ET)' },
  { value: 'America/Chicago', label: 'Central Time (CT)' },
  { value: 'America/Denver', label: 'Mountain Time (MT)' },
  { value: 'America/Los_Angeles', label: 'Pacific Time (PT)' },
  { value: 'Europe/London', label: 'Greenwich Mean Time (GMT)' },
  { value: 'Europe/Paris', label: 'Central European Time (CET)' },
  { value: 'Asia/Tokyo', label: 'Japan Standard Time (JST)' },
  { value: 'Asia/Shanghai', label: 'China Standard Time (CST)' },
  { value: 'Australia/Sydney', label: 'Australian Eastern Time (AET)' }
]

// Computed
const preferencesChanged = computed(() => {
  return JSON.stringify(preferencesForm) !== JSON.stringify(originalPreferences)
})

const notificationsChanged = computed(() => {
  return JSON.stringify(notificationsForm) !== JSON.stringify(originalNotifications)
})

// Methods
const fetchSessions = async () => {
  loadingSessions.value = true
  try {
    const response = await api.getSessions()
    sessions.value = response.data.data || []
  } catch (error) {
    console.error('Failed to fetch sessions:', error)
    ElMessage.error('Failed to load sessions')
  } finally {
    loadingSessions.value = false
  }
}

const fetchSecurityLog = async () => {
  loadingSecurity.value = true
  try {
    const params = {
      page: securityPagination.page,
      limit: securityPagination.limit
    }
    const response = await api.getSessionActivities(params)
    securityEvents.value = response.data.data || []
    securityPagination.total = response.data.total || 0
  } catch (error) {
    console.error('Failed to fetch security log:', error)
    ElMessage.error('Failed to load security log')
  } finally {
    loadingSecurity.value = false
  }
}

const revokeSession = async (session) => {
  try {
    session.loading = true
    await api.revokeSession(session.id)
    ElMessage.success('Session revoked successfully')
    fetchSessions()
  } catch (error) {
    console.error('Failed to revoke session:', error)
    ElMessage.error('Failed to revoke session')
  } finally {
    session.loading = false
  }
}

const revokeAllSessions = async () => {
  try {
    await ElMessageBox.confirm(
      'Are you sure you want to revoke all sessions except the current one?',
      'Revoke All Sessions',
      {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning'
      }
    )
    
    loadingRevokeAll.value = true
    await api.revokeAllUserSessions(authStore.user.id)
    ElMessage.success('All sessions revoked successfully')
    fetchSessions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to revoke all sessions:', error)
      ElMessage.error('Failed to revoke sessions')
    }
  } finally {
    loadingRevokeAll.value = false
  }
}

const refreshSecurityLog = () => {
  securityPagination.page = 1
  fetchSecurityLog()
}

const handleSecuritySizeChange = (size) => {
  securityPagination.limit = size
  securityPagination.page = 1
  fetchSecurityLog()
}

const handleSecurityPageChange = (page) => {
  securityPagination.page = page
  fetchSecurityLog()
}

const fetchPreferences = async () => {
  try {
    const response = await api.get('/user/preferences')
    Object.assign(preferencesForm, response.data)
    Object.assign(originalPreferences, response.data)
  } catch (error) {
    console.error('Failed to fetch preferences:', error)
  }
}

const savePreferences = async () => {
  try {
    loadingPreferences.value = true
    await api.put('/user/preferences', preferencesForm)
    ElMessage.success('Preferences saved successfully')
    Object.assign(originalPreferences, preferencesForm)
  } catch (error) {
    console.error('Failed to save preferences:', error)
    ElMessage.error('Failed to save preferences')
  } finally {
    loadingPreferences.value = false
  }
}

const fetchNotificationSettings = async () => {
  try {
    const response = await api.get('/user/notifications')
    Object.assign(notificationsForm, response.data)
    Object.assign(originalNotifications, response.data)
  } catch (error) {
    console.error('Failed to fetch notification settings:', error)
  }
}

const saveNotificationSettings = async () => {
  try {
    loadingNotifications.value = true
    await api.put('/user/notifications', notificationsForm)
    ElMessage.success('Notification settings saved successfully')
    Object.assign(originalNotifications, notificationsForm)
  } catch (error) {
    console.error('Failed to save notification settings:', error)
    ElMessage.error('Failed to save notification settings')
  } finally {
    loadingNotifications.value = false
  }
}

const handleTabSelect = (tab) => {
  activeTab.value = tab
  router.push({ query: { tab } })
}

const getSecurityEventType = (eventType) => {
  const types = {
    login: 'success',
    logout: 'info',
    failed_login: 'danger',
    password_change: 'warning',
    session_revoke: 'info',
    two_factor_enable: 'success',
    two_factor_disable: 'warning'
  }
  return types[eventType] || 'primary'
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Watch for tab changes in URL
watch(() => router.currentRoute.value.query.tab, (newTab) => {
  if (newTab && ['sessions', 'security', 'preferences', 'notifications'].includes(newTab)) {
    activeTab.value = newTab
  }
})

// Lifecycle
onMounted(async () => {
  // Set active tab from URL query
  const tab = router.currentRoute.value.query.tab
  if (tab && ['sessions', 'security', 'preferences', 'notifications'].includes(tab)) {
    activeTab.value = tab
  }

  // Load data based on active tab
  if (activeTab.value === 'sessions') {
    await fetchSessions()
  } else if (activeTab.value === 'security') {
    await fetchSecurityLog()
  } else if (activeTab.value === 'preferences') {
    await fetchPreferences()
  } else if (activeTab.value === 'notifications') {
    await fetchNotificationSettings()
  }
})
</script>

<style scoped>
.settings-container {
  padding: 20px;
}

.settings-header {
  margin-bottom: 30px;
}

.settings-header h1 {
  margin: 0 0 5px 0;
  font-size: 28px;
  color: #303133;
}

.settings-header p {
  margin: 0;
  color: #909399;
  font-size: 16px;
}

.nav-card {
  position: sticky;
  top: 20px;
}

.settings-menu {
  border-right: none;
}

.settings-content {
  min-height: 500px;
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

.loading-container {
  padding: 20px;
}

.device-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.device-name {
  font-weight: 500;
  color: #303133;
}

.device-details {
  font-size: 12px;
  color: #909399;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.event-details {
  display: flex;
  gap: 15px;
  margin-top: 5px;
}

.event-ip,
.event-device {
  font-size: 12px;
  color: #909399;
}

.time-separator {
  margin: 0 10px;
  color: #606266;
}

.el-form-item {
  margin-bottom: 18px;
}

h4 {
  margin: 0 0 15px 0;
  font-size: 14px;
  color: #303133;
  font-weight: 600;
}
</style>
