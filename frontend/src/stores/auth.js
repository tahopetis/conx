import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  // State
  const user = ref(null)
  const token = ref(localStorage.getItem('token'))
  const refreshToken = ref(localStorage.getItem('refreshToken'))
  const roles = ref([])
  const permissions = ref([])
  const loading = ref(false)
  const error = ref(null)

  // Computed
  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const userRoles = computed(() => roles.value || [])
  const userPermissions = computed(() => permissions.value || [])

  // Actions
  const initializeAuth = async () => {
    if (token.value) {
      try {
        await fetchUserProfile()
      } catch (err) {
        console.error('Failed to initialize auth:', err)
        logout()
      }
    }
  }

  const login = async (credentials) => {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.post('/auth/login', credentials)
      const { access_token, refresh_token, user: userData, roles: userRoles, permissions: userPermissions } = response.data
      
      // Store tokens
      token.value = access_token
      refreshToken.value = refresh_token
      localStorage.setItem('token', access_token)
      localStorage.setItem('refreshToken', refresh_token)
      
      // Store user data
      user.value = userData
      roles.value = userRoles || []
      permissions.value = userPermissions || []
      
      // Setup axios default auth header
      api.defaults.headers.common['Authorization'] = `Bearer ${access_token}`
      
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.message || 'Login failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const register = async (userData) => {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.post('/auth/register', userData)
      return { success: true, data: response.data }
    } catch (err) {
      error.value = err.response?.data?.message || 'Registration failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const logout = () => {
    // Clear state
    user.value = null
    token.value = null
    refreshToken.value = null
    roles.value = []
    permissions.value = []
    error.value = null
    
    // Clear localStorage
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    
    // Clear axios auth header
    delete api.defaults.headers.common['Authorization']
  }

  const fetchUserProfile = async () => {
    if (!token.value) return
    
    try {
      const response = await api.get('/auth/profile')
      user.value = response.data.user
      roles.value = response.data.roles || []
      permissions.value = response.data.permissions || []
      
      // Setup axios default auth header
      api.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
    } catch (err) {
      console.error('Failed to fetch user profile:', err)
      throw err
    }
  }

  const refreshTokens = async () => {
    if (!refreshToken.value) {
      logout()
      throw new Error('No refresh token available')
    }
    
    try {
      const response = await api.post('/auth/refresh', {
        refresh_token: refreshToken.value
      })
      
      const { access_token, refresh_token } = response.data
      
      // Update tokens
      token.value = access_token
      refreshToken.value = refresh_token
      localStorage.setItem('token', access_token)
      localStorage.setItem('refreshToken', refresh_token)
      
      // Update axios auth header
      api.defaults.headers.common['Authorization'] = `Bearer ${access_token}`
      
      return access_token
    } catch (err) {
      console.error('Failed to refresh tokens:', err)
      logout()
      throw err
    }
  }

  const updateProfile = async (profileData) => {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.put('/auth/profile', profileData)
      user.value = { ...user.value, ...response.data }
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.message || 'Profile update failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const changePassword = async (passwordData) => {
    loading.value = true
    error.value = null
    
    try {
      await api.post('/auth/change-password', passwordData)
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.message || 'Password change failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const requestPasswordReset = async (email) => {
    loading.value = true
    error.value = null
    
    try {
      await api.post('/auth/forgot-password', { email })
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.message || 'Password reset request failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const resetPassword = async (token, passwordData) => {
    loading.value = true
    error.value = null
    
    try {
      await api.post(`/auth/reset-password/${token}`, passwordData)
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.message || 'Password reset failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  const hasRole = (role) => {
    return userRoles.value.includes(role)
  }

  const hasAnyRole = (roles) => {
    return roles.some(role => hasRole(role))
  }

  const hasPermission = (permission) => {
    return userPermissions.value.includes(permission)
  }

  const hasAnyPermission = (permissions) => {
    return permissions.some(permission => hasPermission(permission))
  }

  const clearError = () => {
    error.value = null
  }

  return {
    // State
    user,
    token,
    refreshToken,
    roles,
    permissions,
    loading,
    error,
    
    // Computed
    isAuthenticated,
    userRoles,
    userPermissions,
    
    // Actions
    initializeAuth,
    login,
    register,
    logout,
    fetchUserProfile,
    refreshTokens,
    updateProfile,
    changePassword,
    requestPasswordReset,
    resetPassword,
    hasRole,
    hasAnyRole,
    hasPermission,
    hasAnyPermission,
    clearError
  }
})
