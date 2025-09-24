<template>
  <div class="app-container">
    <header class="app-header">
      <div class="header-left">
        <button class="btn-icon" @click="toggleSidebar">
          <el-icon><Menu /></el-icon>
        </button>
        <router-link to="/" class="logo">
          <div class="logo-icon">C</div>
          <span>conx CMDB</span>
        </router-link>
      </div>
      
      <div class="header-right">
        <el-tooltip content="Search" placement="bottom">
          <button class="btn-icon" @click="goToSearch">
            <el-icon><Search /></el-icon>
          </button>
        </el-tooltip>
        
        <el-tooltip content="Notifications" placement="bottom">
          <button class="btn-icon">
            <el-icon><Bell /></el-icon>
          </button>
        </el-tooltip>
        
        <el-dropdown @command="handleUserMenuCommand">
          <div class="user-menu">
            <el-avatar :size="32" :src="userAvatar">
              {{ userInitials }}
            </el-avatar>
            <span class="user-name">{{ userName }}</span>
            <el-icon><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon>
                Profile
              </el-dropdown-item>
              <el-dropdown-item command="settings">
                <el-icon><Setting /></el-icon>
                Settings
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">
                <el-icon><SwitchButton /></el-icon>
                Logout
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>
    
    <div class="main-layout">
      <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-header">
          <h3>Navigation</h3>
        </div>
        
        <nav class="nav-menu">
          <router-link 
            v-for="item in navigationItems" 
            :key="item.path"
            :to="item.path" 
            class="nav-item"
            :class="{ active: isActiveRoute(item.path) }"
          >
            <el-icon>
              <component :is="item.icon" />
            </el-icon>
            <span v-show="!sidebarCollapsed">{{ item.name }}</span>
          </router-link>
        </nav>
        
        <div class="sidebar-footer">
          <div class="version-info">
            <span v-show="!sidebarCollapsed">v1.0.0</span>
          </div>
        </div>
      </aside>
      
      <main class="content-area">
        <div class="page-content">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  Menu,
  Search,
  Bell,
  User,
  Setting,
  SwitchButton,
  ArrowDown,
  Monitor,
  Connection,
  DataAnalysis,
  Document
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const sidebarCollapsed = ref(false)

// Navigation items
const navigationItems = [
  {
    path: '/',
    name: 'Dashboard',
    icon: Monitor
  },
  {
    path: '/cis',
    name: 'Configuration Items',
    icon: Document
  },
  {
    path: '/graph',
    name: 'Graph Visualization',
    icon: Connection
  },
  {
    path: '/search',
    name: 'Search',
    icon: Search
  }
]

// Computed properties
const userName = computed(() => {
  return authStore.user?.name || authStore.user?.email || 'User'
})

const userInitials = computed(() => {
  if (authStore.user?.name) {
    return authStore.user.name
      .split(' ')
      .map(word => word[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }
  return authStore.user?.email?.[0]?.toUpperCase() || 'U'
})

const userAvatar = computed(() => {
  return authStore.user?.avatar || ''
})

// Methods
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const isActiveRoute = (path) => {
  return route.path === path
}

const goToSearch = () => {
  router.push('/search')
}

const handleUserMenuCommand = async (command) => {
  switch (command) {
    case 'profile':
      router.push('/profile')
      break
    case 'settings':
      router.push('/settings')
      break
    case 'logout':
      try {
        await ElMessageBox.confirm(
          'Are you sure you want to logout?',
          'Confirm Logout',
          {
            confirmButtonText: 'Logout',
            cancelButtonText: 'Cancel',
            type: 'warning'
          }
        )
        
        await authStore.logout()
        ElMessage.success('Logged out successfully')
        router.push('/auth/login')
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('Failed to logout')
        }
      }
      break
  }
}
</script>

<style scoped>
.user-menu {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.3s ease;
}

.user-menu:hover {
  background-color: #f5f7fa;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: #2c3e50;
}

.sidebar-header {
  padding: 20px 24px;
  border-bottom: 1px solid #e4e7ed;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
}

.sidebar-footer {
  padding: 16px 24px;
  border-top: 1px solid #e4e7ed;
  margin-top: auto;
}

.version-info {
  text-align: center;
  font-size: 12px;
  color: #909399;
}

@media (max-width: 768px) {
  .user-name {
    display: none;
  }
}
</style>
